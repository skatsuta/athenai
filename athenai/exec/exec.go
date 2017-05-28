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
)

// QueryConfig is configurations for query executions.
type QueryConfig struct {
	Database string
	Output   string
}

// Query represents a query to be executed.
type Query struct {
	*QueryConfig
	interval time.Duration
	client   athenaiface.AthenaAPI
	query    string
	id       string
	metadata *athena.QueryExecution
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

	for {
		qeo, err := q.client.GetQueryExecution(&athena.GetQueryExecutionInput{
			QueryExecutionId: aws.String(q.id),
		})
		if err != nil {
			return errors.Wrap(err, "GetQueryExecution API error")
		}

		qe := qeo.QueryExecution
		q.metadata = qe
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

// Run starts the specified query and waits for it to complete.
func (q *Query) Run() error {
	if err := q.Start(); err != nil {
		return errors.Wrap(err, "failed to start query execution")
	}
	return q.Wait()
}
