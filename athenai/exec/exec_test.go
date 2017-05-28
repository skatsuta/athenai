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

var mockedResultSet = &athena.ResultSet{
	ResultSetMetadata: &athena.ResultSetMetadata{},
	Rows:              []*athena.Row{{}, {}, {}, {}, {}},
}

type mockedGetQueryResults struct {
	athenaiface.AthenaAPI
	page     int
	maxPages int
}

func (m *mockedGetQueryResults) GetQueryResults(input *athena.GetQueryResultsInput) (*athena.GetQueryResultsOutput, error) {
	m.page++
	resp := &athena.GetQueryResultsOutput{
		ResultSet: mockedResultSet,
	}
	if m.page < m.maxPages {
		resp.NextToken = aws.String("next")
	}
	return resp, nil
}

func (m *mockedGetQueryResults) GetQueryResultsPages(input *athena.GetQueryResultsInput, callback func(*athena.GetQueryResultsOutput, bool) bool) error {
	cont := true
	for cont {
		qr, err := m.GetQueryResults(input)
		if err != nil {
			return err
		}

		lastPage := qr.NextToken == nil
		cont = callback(qr, lastPage)
		cont = cont && !lastPage
	}

	return nil
}

func TestGetResults(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query    string
		id       string
		metadata *athena.QueryExecution
		maxPages int
		numRows  int
	}{
		{
			query: "SELECT * FROM cloudfront_logs LIMIT 10",
			id:    "1",
			metadata: &athena.QueryExecution{
				Status: &athena.QueryExecutionStatus{
					State: aws.String(athena.QueryExecutionStateSucceeded),
				},
			},
			maxPages: 2,
			numRows:  10,
		},
	}

	for _, tt := range tests {
		q := &Query{
			QueryConfig: cfg,
			client: &mockedGetQueryResults{
				maxPages: tt.maxPages,
			},
			interval: 0 * time.Millisecond,
			query:    tt.query,
			id:       tt.id,
			metadata: tt.metadata,
		}

		err := q.GetResults()
		assert.Nil(t, err)
		assert.Len(t, q.results.Rows, tt.numRows, "Query: %s, Id: %s", tt.query, tt.id)
	}
}

type mockedGetQueryResultsError struct {
	athenaiface.AthenaAPI
	errMsg string
}

func (m *mockedGetQueryResultsError) GetQueryResultsPages(input *athena.GetQueryResultsInput, callback func(*athena.GetQueryResultsOutput, bool) bool) error {
	return errors.New(m.errMsg)
}

func TestGetResultsError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query  string
		id     string
		errMsg string
	}{
		{
			query:  "SELECT * FROM test_get_result_errors",
			id:     "no_existent_id",
			errMsg: "InvalidRequestException",
		},
	}

	for _, tt := range tests {
		q := &Query{
			QueryConfig: cfg,
			client: &mockedGetQueryResultsError{
				errMsg: tt.errMsg,
			},
			interval: 0 * time.Millisecond,
			query:    tt.query,
			id:       tt.id,
		}

		err := q.GetResults()
		assert.NotNil(t, err)
	}
}
