package athenai

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
)

const (
	sleepDuration = 500 // milliseconds
)

// Athenai is a client that interacts with Amazon Athena.
type Athenai struct {
	client   athenaiface.AthenaAPI
	database string
	output   string
}

// New creates a new Athenai object.
func New(region, database, output string) *Athenai {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
	return &Athenai{
		client:   athena.New(sess),
		database: database,
		output:   output,
	}
}

// RunQuery runs queries.
func (a *Athenai) RunQuery(query string) ([]string, error) {
	if a == nil {
		return nil, errors.New("receiver *Athenai is nil")
	}

	// TODO: run multiple queries concurrently using goroutines

	queries := strings.Split(query, ";")
	ids := make([]string, len(queries))

	for i, query := range queries {
		id, err := a.RunSingleQuery(query)
		if err != nil {
			return nil, errors.Wrap(err, "query execution failed")
		}
		ids[i] = id
	}

	return ids, nil
}

// RunSingleQuery runs a simple query and returns its query execution id.
func (a *Athenai) RunSingleQuery(query string) (string, error) {
	params := &athena.StartQueryExecutionInput{
		QueryString: aws.String(query),
		ResultConfiguration: &athena.ResultConfiguration{
			OutputLocation: aws.String(a.output),
		},
	}
	if a.database != "" {
		params.QueryExecutionContext = &athena.QueryExecutionContext{
			Database: aws.String(a.database),
		}
	}

	qe, err := a.client.StartQueryExecution(params)
	if err != nil {
		return "", errors.Wrap(err, "StartQueryExecution API error")
	}
	return aws.StringValue(qe.QueryExecutionId), nil
}

// WaitSingleExecution waits the query execution of the id and returns the excution metadata
// after it has finished.
func (a *Athenai) WaitSingleExecution(id string) (*athena.QueryExecution, error) {
	if a == nil {
		return nil, errors.New("receiver *Athenai is nil")
	}

	for {
		qeo, err := a.client.GetQueryExecution(&athena.GetQueryExecutionInput{
			QueryExecutionId: aws.String(id),
		})
		if err != nil {
			return nil, errors.Wrap(err, "GetQueryExecution API error")
		}

		qe := qeo.QueryExecution
		status := qe.Status
		switch aws.StringValue(status.State) {
		case athena.QueryExecutionStateSucceeded:
			return qe, nil
		case athena.QueryExecutionStateFailed:
			return qe, errors.Errorf("query failed: %s", aws.StringValue(status.StateChangeReason))
		case athena.QueryExecutionStateCancelled:
			return qe, errors.Errorf("query cancelled: %s", aws.StringValue(status.StateChangeReason))
		}

		time.Sleep(sleepDuration * time.Millisecond)
	}
}
