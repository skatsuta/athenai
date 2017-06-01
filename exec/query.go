package exec

import (
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
)

const (
	waitInterval = 500 * time.Millisecond

	// The maximum number of results (rows) to return in a GetQueryResults API request.
	// See https://docs.aws.amazon.com/ja_jp/athena/latest/APIReference/API_GetQueryResults.html#API_GetQueryResults_RequestSyntax
	maxResults = 1000
)

// Result represents results of a query execution.
// This struct must implement print.Result interface.
type Result struct {
	mu   sync.RWMutex
	info *athena.QueryExecution
	rs   *athena.ResultSet
}

// Info returns information of a query execution.
func (r *Result) Info() *athena.QueryExecution {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.info
}

func (r *Result) sendRows(ch chan []string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rw := range r.rs.Rows {
		row := make([]string, 0, len(rw.Data))
		for _, d := range rw.Data {
			row = append(row, aws.StringValue(d.VarCharValue))
		}
		ch <- row
	}
	close(ch)
}

// Rows returns a receive-only channel which receives each row of query results.
func (r *Result) Rows() <-chan []string {
	ch := make(chan []string)
	go r.sendRows(ch)
	return ch
}

// QueryConfig is configurations for query executions.
type QueryConfig struct {
	Database string
	Output   string
}

// Query represents a query to be executed.
// Query is NOT goroutine-safe so must be used in a single goroutine.
type Query struct {
	*QueryConfig
	*Result

	interval time.Duration
	client   athenaiface.AthenaAPI
	query    string
	id       string
}

// NewQuery creates a new Query struct.
// query string must be a single SQL statement rather than multiple ones joined by semicolons.
func NewQuery(client athenaiface.AthenaAPI, query string, cfg *QueryConfig) (*Query, error) {
	if client == nil || len(query) == 0 || cfg == nil {
		return nil, errors.New("NewQuery(): invalid argument(s)")
	}

	q := &Query{
		QueryConfig: cfg,
		Result:      &Result{},
		interval:    waitInterval,
		client:      client,
		query:       query,
	}
	log.Printf("Created %#v\n", q)
	return q, nil
}

// Start starts the specified query but does not wait for it to complete.
func (q *Query) Start() error {
	params := &athena.StartQueryExecutionInput{
		QueryString: aws.String(q.query),
		ResultConfiguration: &athena.ResultConfiguration{
			OutputLocation: aws.String(q.Output),
		},
	}
	if q.Database != "" {
		params.QueryExecutionContext = &athena.QueryExecutionContext{
			Database: aws.String(q.Database),
		}
	}

	qe, err := q.client.StartQueryExecution(params)
	if err != nil {
		return errors.Wrap(err, "StartQueryExecution API error")
	}

	q.id = aws.StringValue(qe.QueryExecutionId)
	log.Printf("Query execution ID: %s\n", q.id)
	return nil
}

// Wait waits for the query execution until its state has become SUCCEEDED, FAILED or CANCELLED.
func (q *Query) Wait() error {
	if q.id == "" {
		return errors.New("query has not started yet or already failed to start")
	}

	// TODO: timeout after 30 minutes using Context
	// See https://docs.aws.amazon.com/athena/latest/ug/service-limits.html
	for {
		qeo, err := q.client.GetQueryExecution(&athena.GetQueryExecutionInput{
			QueryExecutionId: aws.String(q.id),
		})
		if err != nil {
			return errors.Wrap(err, "GetQueryExecution API error")
		}

		qe := qeo.QueryExecution
		q.info = qe
		state := aws.StringValue(qe.Status.State)
		switch state {
		case athena.QueryExecutionStateSucceeded, athena.QueryExecutionStateFailed, athena.QueryExecutionStateCancelled:
			log.Printf("Query execution %s has finished: %s\n", q.id, state)
			return nil
		}

		log.Printf("Query execution state: %s; sleeping %s\n", state, q.interval.String())
		time.Sleep(q.interval)
	}
}

// GetResults gets the results of the query execution.
func (q *Query) GetResults() error {
	params := &athena.GetQueryResultsInput{
		QueryExecutionId: aws.String(q.id),
		MaxResults:       aws.Int64(maxResults),
	}

	rs := &athena.ResultSet{}
	callback := func(page *athena.GetQueryResultsOutput, lastPage bool) bool {
		if rs.ResultSetMetadata == nil {
			rs.ResultSetMetadata = page.ResultSet.ResultSetMetadata
		}
		rs.Rows = append(rs.Rows, page.ResultSet.Rows...)
		return !lastPage
	}

	if err := q.client.GetQueryResultsPages(params, callback); err != nil {
		return errors.Wrap(err, "GetQueryResults API error")
	}

	q.rs = rs
	return nil
}

// Run starts the specified query, waits for it to complete and fetch the results.
func (q *Query) Run() (*Result, error) {
	if err := q.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start query execution")
	}
	if err := q.Wait(); err != nil {
		return nil, errors.Wrap(err, "error while waiting for the query execution")
	}
	if err := q.GetResults(); err != nil {
		return nil, errors.Wrap(err, "failed to get query results")
	}
	return q.Result, nil
}
