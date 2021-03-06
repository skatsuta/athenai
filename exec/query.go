package exec

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
)

const (
	// DefaultWaitInterval is a default value of wait interval.
	DefaultWaitInterval = 1000 * time.Millisecond

	// The maximum number of results (rows) to return in a GetQueryResults API request.
	// See https://docs.aws.amazon.com/ja_jp/athena/latest/APIReference/API_GetQueryResults.html#API_GetQueryResults_RequestSyntax
	maxResults = 1000

	queryExecutionCanceled = "query execution request has been canceled"
)

// CanceledError represents an error that a query execution been canceled.
type CanceledError struct {
	Query string
	ID    string
}

func (e *CanceledError) Error() string {
	if e.ID == "" {
		return queryExecutionCanceled
	}
	return fmt.Sprintf("query execution %s has been canceled", e.ID)
}

func (e *CanceledError) String() string {
	return e.Error()
}

// QueryConfig is configurations for query executions.
type QueryConfig struct {
	Database string
	Location string
	Encrypt  string
	KMS      string // Required only if Encrypt = (SSE_KMS|CSE_KMS)
}

// Query represents a query to be executed.
// Query is NOT goroutine-safe so must be used in a single goroutine.
type Query struct {
	*QueryConfig
	*Result

	client       athenaiface.AthenaAPI
	waitInterval time.Duration
	query        string
	id           string
}

// NewQuery creates a new Query struct.
// `query` string must be a single SQL statement rather than multiple ones joined by semicolons.
func NewQuery(client athenaiface.AthenaAPI, cfg *QueryConfig, query string) *Query {
	q := &Query{
		QueryConfig:  cfg,
		Result:       &Result{},
		client:       client,
		waitInterval: DefaultWaitInterval,
		query:        query,
	}
	log.Printf("Created Query: %#v\n", q)
	return q
}

// NewQueryFromQx creates a new Query struct from information about a query execution.
func NewQueryFromQx(client athenaiface.AthenaAPI, cfg *QueryConfig, qx *athena.QueryExecution) *Query {
	q := &Query{
		QueryConfig:  cfg,
		Result:       &Result{info: qx},
		client:       client,
		waitInterval: DefaultWaitInterval,
		query:        aws.StringValue(qx.Query),
		id:           aws.StringValue(qx.QueryExecutionId),
	}
	log.Printf("Created Query: %#v\n", q)
	return q
}

// WithWaitInterval sets wait interval to q.
func (q *Query) WithWaitInterval(interval time.Duration) *Query {
	q.waitInterval = interval
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
	if q.Encrypt != "" {
		params.ResultConfiguration.EncryptionConfiguration = &athena.EncryptionConfiguration{
			EncryptionOption: &q.Encrypt,
			KmsKey:           &q.KMS,
		}
	}

	qx, err := q.client.StartQueryExecutionWithContext(ctx, params)
	if err != nil {
		if cerr, ok := err.(awserr.Error); ok && cerr.Code() == request.CanceledErrorCode {
			return &CanceledError{Query: q.query}
		}
		return errors.Wrap(err, "StartQueryExecution API error")
	}

	q.id = aws.StringValue(qx.QueryExecutionId)
	log.Printf("Query execution ID: %s\n", q.id)
	return nil
}

// Wait waits for the query execution until its state has become SUCCEEDED, FAILED or CANCELLED.
//
// If the given Context has been canceled, it calls StopQueryExecution API and tries to cancel
// the query execution.
func (q *Query) Wait(ctx context.Context) error {
	if q.id == "" {
		return errors.New("query execution has not started yet or already failed to start")
	}

	input := &athena.GetQueryExecutionInput{QueryExecutionId: &q.id}
	for {
		select {
		case <-ctx.Done(): // Query execution has been canceled by user
			_, err := q.client.StopQueryExecution(&athena.StopQueryExecutionInput{QueryExecutionId: &q.id})
			if err != nil {
				return errors.Wrap(err, "StopQueryExecution API error")
			}
		default: // No op here by default
		}

		// Call the API without context since do not want context to cancel the API call
		qxo, err := q.client.GetQueryExecution(input)
		if err != nil {
			return errors.Wrap(err, "GetQueryExecution API error")
		}

		qx := qxo.QueryExecution
		q.info = qx
		state := aws.StringValue(qx.Status.State)
		log.Printf("State of query execution %s: %s\n", q.id, state)

		switch state {
		case athena.QueryExecutionStateSucceeded:
			return nil
		case athena.QueryExecutionStateFailed:
			reason := aws.StringValue(qx.Status.StateChangeReason)
			return errors.Errorf("query execution %s has failed. Reason: %s", q.id, reason)
		case athena.QueryExecutionStateCancelled:
			return &CanceledError{Query: q.query, ID: q.id}
		}

		log.Printf("Query execution %s has not finished yet; Sleeping %s\n", q.id, q.waitInterval)
		time.Sleep(q.waitInterval)
	}
}

// GetResults gets the results of the query execution.
func (q *Query) GetResults(ctx context.Context) error {
	if q.id == "" {
		return errors.New("query execution has not started yet or already failed to start")
	}

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
		if cerr, ok := err.(awserr.Error); ok && cerr.Code() == request.CanceledErrorCode {
			return &CanceledError{Query: q.query, ID: q.id}
		}
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
