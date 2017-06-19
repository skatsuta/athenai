package exec

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
)

const (
	defaultWaitInterval = 500 * time.Millisecond

	// The maximum number of results (rows) to return in a GetQueryResults API request.
	// See https://docs.aws.amazon.com/ja_jp/athena/latest/APIReference/API_GetQueryResults.html#API_GetQueryResults_RequestSyntax
	maxResults = 1000
)

var (
	// ErrQueryExecutionFailed represents an error where a query execution has been failed.
	ErrQueryExecutionFailed = errors.New("query execution has been failed")
	// ErrQueryExecutionCancelled represents an error where a query execution has been cancelled.
	ErrQueryExecutionCancelled = errors.New("query execution has been cancelled")
)

// QueryConfig is configurations for query executions.
type QueryConfig struct {
	Database string
	Location string
}

// Query represents a query to be executed.
// Query is NOT goroutine-safe so must be used in a single goroutine.
type Query struct {
	*QueryConfig
	*Result
	WaitInterval time.Duration

	client athenaiface.AthenaAPI
	query  string
	id     string
}

// NewQuery creates a new Query struct.
// query string must be a single SQL statement rather than multiple ones joined by semicolons.
func NewQuery(client athenaiface.AthenaAPI, query string, cfg *QueryConfig) *Query {
	if client == nil || cfg == nil {
		panic("client or cfg is nil") // it's a code bug so let's panic
	}

	q := &Query{
		QueryConfig:  cfg,
		Result:       &Result{},
		WaitInterval: defaultWaitInterval,
		client:       client,
		query:        query,
	}
	log.Printf("Created Query: %#v\n", q)
	return q
}

// Start starts the specified query but does not wait for it to complete.
func (q *Query) Start(ctx context.Context) error {
	params := &athena.StartQueryExecutionInput{
		QueryString:         &q.query,
		ResultConfiguration: &athena.ResultConfiguration{OutputLocation: &q.Location},
	}
	if q.Database != "" {
		params.QueryExecutionContext = &athena.QueryExecutionContext{Database: &q.Database}
	}

	qe, err := q.client.StartQueryExecutionWithContext(ctx, params)
	if err != nil {
		return errors.Wrap(err, "StartQueryExecution API error")
	}

	q.id = aws.StringValue(qe.QueryExecutionId)
	log.Printf("Query execution ID: %s\n", q.id)
	return nil
}

// Wait waits for the query execution until its state has become SUCCEEDED, FAILED or CANCELLED.
//
// If the given Context has been cancelled, it calls StopQueryExecution API and tries to cancel
// the query execution.
func (q *Query) Wait(ctx context.Context) error {
	if q.id == "" {
		return errors.New("query has not started yet or already failed to start")
	}

	input := &athena.GetQueryExecutionInput{QueryExecutionId: &q.id}
	for {
		select {
		case <-ctx.Done(): // Query execution has been cancelled by the user
			_, err := q.client.StopQueryExecution(&athena.StopQueryExecutionInput{QueryExecutionId: &q.id})
			if err != nil {
				return errors.Wrap(err, "StopQueryExecution API error")
			}
		default: // No-op here by default
		}

		// Call the API without context since do not want context to cancel the API call
		qeo, err := q.client.GetQueryExecution(input)
		if err != nil {
			return errors.Wrap(err, "GetQueryExecution API error")
		}

		qe := qeo.QueryExecution
		q.info = qe

		state := aws.StringValue(qe.Status.State)
		switch state {
		case athena.QueryExecutionStateSucceeded:
			log.Printf("Query execution %s has finished: %s\n", q.id, state)
			return nil
		case athena.QueryExecutionStateFailed:
			log.Printf("Query execution %s has finished: %s\n", q.id, state)
			return ErrQueryExecutionFailed
		case athena.QueryExecutionStateCancelled:
			log.Printf("Query execution %s has finished: %s\n", q.id, state)
			return ErrQueryExecutionCancelled
		}

		log.Printf("Query execution state: %s; sleeping %s\n", state, q.WaitInterval.String())
		time.Sleep(q.WaitInterval)
	}
}

// GetResults gets the results of the query execution.
func (q *Query) GetResults(ctx context.Context) error {
	params := &athena.GetQueryResultsInput{
		QueryExecutionId: &q.id,
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

	if err := q.client.GetQueryResultsPagesWithContext(ctx, params, callback); err != nil {
		return errors.Wrap(err, "GetQueryResults API error")
	}

	q.rs = rs
	return nil
}

// Run starts the specified query, waits for it to complete and fetch the results.
func (q *Query) Run(ctx context.Context) (*Result, error) {
	if err := q.Start(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to start query execution")
	}
	if err := q.Wait(ctx); err != nil {
		return nil, errors.Wrap(err, "error while waiting for the query execution")
	}
	if err := q.GetResults(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to get query results")
	}
	return q.Result, nil
}
