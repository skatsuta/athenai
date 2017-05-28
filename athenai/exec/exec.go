package exec

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
)

const (
	sleepDuration = 500 // milliseconds
)

// QueryConfig is configurations for query executions.
type QueryConfig struct {
	Database string
	Output   string
}

// Query represents a query to be executed.
type Query struct {
	*QueryConfig
	// sleep duration in milliseconds
	sleep    time.Duration
	client   athenaiface.AthenaAPI
	query    string
	id       string
	metadata *athena.QueryExecution
}

// NewQuery creates a new Query struct.
func NewQuery(client athenaiface.AthenaAPI, query string, cfg *QueryConfig) (*Query, error) {
	if client == nil || len(query) == 0 || cfg == nil {
		return nil, errors.New("NewQuery(): invalid argument(s)")
	}

	q := &Query{
		QueryConfig: cfg,
		sleep:       sleepDuration,
		client:      client,
		query:       query,
	}
	return q, nil
}

// RunQuery runs queries.
// func (q *Query) RunQuery(query string) ([]string, error) {
// if q == nil {
// return nil, errors.New("receiver *Athenai is nil")
// }

// // TODO: run multiple queries concurrently using goroutines

// queries := strings.Split(query, ";")
// ids := make([]string, len(queries))

// for i, query := range queries {
// id, err := q.RunSingleQuery(query)
// if err != nil {
// return nil, errors.Wrap(err, "query execution failed")
// }
// ids[i] = id
// }

// return ids, nil
// }

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
			return nil
		}

		time.Sleep(q.sleep * time.Millisecond)
	}
}
