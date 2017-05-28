package exec

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
)

const (
	getQueryExecutionAPICallInterval = 500 * time.Millisecond

	// The maximum number of results (rows) to return in a GetQueryResults API request.
	// See https://docs.aws.amazon.com/ja_jp/athena/latest/APIReference/API_GetQueryResults.html#API_GetQueryResults_RequestSyntax
	getQueryResultsAPIMaxResults = 1000
)

// QueryConfig is configurations for query executions.
type QueryConfig struct {
	Database string
	Output   string
}

// Query represents a query to be executed.
// Query is NOT goroutine-safe so must be used in a single goroutine.
type Query struct {
	*QueryConfig
	interval time.Duration
	client   athenaiface.AthenaAPI
	query    string
	id       string
	info     *athena.QueryExecution
	results  *athena.ResultSet
}

// NewQuery creates a new Query struct.
// query string must be a single SQL statement rather than multiple ones joined by semicolons.
func NewQuery(client athenaiface.AthenaAPI, query string, cfg *QueryConfig) (*Query, error) {
	if client == nil || len(query) == 0 || cfg == nil {
		return nil, errors.New("NewQuery(): invalid argument(s)")
	}

	q := &Query{
		QueryConfig: cfg,
		interval:    getQueryExecutionAPICallInterval,
		client:      client,
		query:       query,
	}
	log.Printf("created %#v\n", q)
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
	log.Printf("query execution id: %s\n", q.id)
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
			log.Printf("query execution %s has finished: %s\n", q.id, state)
			return nil
		}

		log.Printf("query execution state: %s; sleeping %s\n", state, q.interval.String())
		time.Sleep(q.interval)
	}
}

// GetResults gets the results of the query execution.
func (q *Query) GetResults() error {
	results := &athena.ResultSet{}

	params := &athena.GetQueryResultsInput{
		QueryExecutionId: aws.String(q.id),
		MaxResults:       aws.Int64(getQueryResultsAPIMaxResults),
	}
	callback := func(page *athena.GetQueryResultsOutput, lastPage bool) bool {
		if results.ResultSetMetadata == nil {
			results.ResultSetMetadata = page.ResultSet.ResultSetMetadata
		}

		results.Rows = append(results.Rows, page.ResultSet.Rows...)
		return !lastPage
	}

	if err := q.client.GetQueryResultsPages(params, callback); err != nil {
		return errors.Wrap(err, "GetQueryResults API error")
	}

	q.results = results
	return nil
}

// Run starts the specified query and waits for it to complete.
func (q *Query) Run() error {
	if err := q.Start(); err != nil {
		return errors.Wrap(err, "failed to start query execution")
	}
	if err := q.Wait(); err != nil {
		return errors.Wrap(err, "error while waiting for the query execution")
	}
	return q.GetResults()
}
