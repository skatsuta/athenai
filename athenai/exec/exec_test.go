package exec

import (
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockedStartQueryExecution struct {
	athenaiface.AthenaAPI
	id string
}

func (m *mockedStartQueryExecution) StartQueryExecution(input *athena.StartQueryExecutionInput) (*athena.StartQueryExecutionOutput, error) {
	query := aws.StringValue(input.QueryString)
	for _, kwd := range []string{"SELECT", "SHOW"} {
		if strings.HasPrefix(query, kwd) {
			resp := &athena.StartQueryExecutionOutput{
				QueryExecutionId: &m.id,
			}
			return resp, nil
		}
	}
	return nil, errors.Errorf("InvalidRequestException: %q", query)
}

func TestNewError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query string
	}{
		{""},
	}

	for _, tt := range tests {
		q, err := NewQuery(&mockedStartQueryExecution{}, tt.query, cfg)
		assert.NotNil(t, err, "Query: %#v", tt.query)
		assert.Nil(t, q)
	}
}

func TestStart(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query    string
		id       string
		expected string
	}{
		{"SELECT * FROM elb_logs", "1", "1"},
	}

	for _, tt := range tests {
		q, err := NewQuery(&mockedStartQueryExecution{id: tt.id}, tt.query, cfg)
		assert.Nil(t, err)

		err = q.Start()
		assert.Nil(t, err)
		assert.Equal(t, tt.expected, q.id, "Query: %q", tt.query)
	}
}

func TestStartError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query   string
		errCode string
	}{
		{"SELET * FROM test", "InvalidRequestException"},
	}

	for _, tt := range tests {
		q, err := NewQuery(&mockedStartQueryExecution{}, tt.query, cfg)
		assert.Nil(t, err)

		err = q.Start()
		if assert.NotNil(t, err) {
			assert.Contains(t, err.Error(), tt.errCode, "Query: %q", tt.query)
		}
	}
}

var successfulQueryStateFlow = []string{
	athena.QueryExecutionStateQueued,
	athena.QueryExecutionStateRunning,
	athena.QueryExecutionStateSucceeded,
}

var failedQueryStateFlow = []string{
	athena.QueryExecutionStateQueued,
	athena.QueryExecutionStateRunning,
	athena.QueryExecutionStateFailed,
}

type mockedGetQueryExecution struct {
	*mockedStartQueryExecution
	queryStateFlow []string
	stateCnt       int
}

func (m *mockedGetQueryExecution) GetQueryExecution(input *athena.GetQueryExecutionInput) (*athena.GetQueryExecutionOutput, error) {
	l := len(m.queryStateFlow)
	var state string
	if m.stateCnt < l {
		state = m.queryStateFlow[m.stateCnt]
	} else {
		state = m.queryStateFlow[l-1]
	}

	m.stateCnt++

	resp := &athena.GetQueryExecutionOutput{
		QueryExecution: &athena.QueryExecution{
			QueryExecutionId: aws.String(m.mockedStartQueryExecution.id),
			Status: &athena.QueryExecutionStatus{
				State: aws.String(state),
			},
		},
	}
	return resp, nil
}

func TestWait(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query  string
		id     string
		status string
	}{
		{"SELECT * FROM cloudfront_logs", "1", athena.QueryExecutionStateSucceeded},
		{"SHOW TABLES", "2", athena.QueryExecutionStateSucceeded},
	}

	for _, tt := range tests {
		q := &Query{
			QueryConfig: cfg,
			client: &mockedGetQueryExecution{
				mockedStartQueryExecution: &mockedStartQueryExecution{
					id: tt.id,
				},
				queryStateFlow: successfulQueryStateFlow,
			},
			interval: 0 * time.Millisecond,
			query:    tt.query,
		}

		err := q.Start()
		assert.Nil(t, err)

		err = q.Wait()
		assert.Nil(t, err)
		assert.Equal(t, tt.id, aws.StringValue(q.metadata.QueryExecutionId), "Query: %s, Id: %s", tt.query, tt.id)
		assert.Equal(t, tt.status, aws.StringValue(q.metadata.Status.State), "Query: %s, Id: %s", tt.query, tt.id)
	}
}

type mockedGetQueryExecutionError struct {
	*mockedStartQueryExecution
	errMsg string
}

func (m *mockedGetQueryExecutionError) GetQueryExecution(input *athena.GetQueryExecutionInput) (*athena.GetQueryExecutionOutput, error) {
	return nil, errors.New(m.errMsg)
}

func TestWaitError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query  string
		id     string
		status string
	}{
		{"SELECT * FROM no_existent_table", "1", athena.QueryExecutionStateFailed},
	}

	for _, tt := range tests {
		q := &Query{
			QueryConfig: cfg,
			client: &mockedGetQueryExecutionError{
				mockedStartQueryExecution: &mockedStartQueryExecution{},
				errMsg: "an internal error occurred",
			},
			interval: 0 * time.Millisecond,
			query:    tt.query,
		}

		err := q.Start()
		assert.Nil(t, err)

		err = q.Wait()
		assert.NotNil(t, err)
	}
}
