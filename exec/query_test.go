package exec

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/skatsuta/athenai/internal/stub"
	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Location: "s3://bucket/prefix/",
	}

	tests := []struct {
		id    string
		query string
		want  string
	}{
		{
			id:    "TestStart1",
			query: "SELECT * FROM elb_logs",
			want:  "TestStart1",
		},
	}

	for _, tt := range tests {
		client := stub.NewStartQueryExecutionStub(&stub.Result{ID: tt.id, Query: tt.query})
		q := NewQuery(client, tt.query, cfg)
		err := q.Start()

		assert.NoError(t, err)
		assert.Equal(t, tt.want, q.id, "Query: %q", tt.query)
	}
}

func TestStartError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Location: "s3://bucket/prefix/",
	}

	tests := []struct {
		query   string
		errCode string
	}{
		{"", "InvalidRequestException"},
		{"SELET * FROM test", "InvalidRequestException"},
		{"CREATE INDEX", "InvalidRequestException"},
	}

	for _, tt := range tests {
		client := stub.NewStartQueryExecutionStub(&stub.Result{Query: tt.query})
		q := NewQuery(client, tt.query, cfg)
		err := q.Start()

		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), tt.errCode, "Query: %q", tt.query)
		}
	}
}

func TestWait(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Location: "s3://bucket/prefix/",
	}

	tests := []struct {
		id     string
		query  string
		status string
	}{
		{"SELECT * FROM cloudfront_logs", "TestWait1", athena.QueryExecutionStateSucceeded},
		{"SHOW TABLES", "TestWait2", athena.QueryExecutionStateSucceeded},
	}

	for _, tt := range tests {
		q := &Query{
			QueryConfig: cfg,
			Result:      &Result{},
			client:      stub.NewGetQueryExecutionStub(&stub.Result{ID: tt.id, Query: tt.query}),
			interval:    10 * time.Millisecond,
			query:       tt.query,
			id:          tt.id,
		}
		err := q.Wait()
		got := aws.StringValue(q.Info().Status.State)

		assert.NoError(t, err)
		assert.Equal(t, tt.status, got, "Query: %s, Id: %s", tt.query, tt.id)
	}
}

func TestWaitError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Location: "s3://bucket/prefix/",
	}

	tests := []struct {
		id     string
		query  string
		status string
	}{
		{"SELECT * FROM no_existent_table", "1", athena.QueryExecutionStateFailed},
	}

	for _, tt := range tests {
		q := &Query{
			QueryConfig: cfg,
			client: stub.NewGetQueryExecutionStub(&stub.Result{
				ID:     tt.id,
				Query:  tt.query,
				ErrMsg: "an internal error occurred",
			}),
			interval: 10 * time.Millisecond,
			query:    tt.query,
			id:       tt.id,
		}
		err := q.Wait()

		assert.Error(t, err)
	}
}

func TestGetResults(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Location: "s3://bucket/prefix/",
	}

	tests := []struct {
		id       string
		query    string
		info     *athena.QueryExecution
		maxPages int
		numRows  int
	}{
		{
			id:    "TestGetResults1",
			query: "SELECT * FROM cloudfront_logs LIMIT 10",
			info: &athena.QueryExecution{
				Status: &athena.QueryExecutionStatus{
					State: aws.String(athena.QueryExecutionStateSucceeded),
				},
			},
			maxPages: 2,
			numRows:  10,
		},
	}

	for _, tt := range tests {
		client := stub.NewGetQueryResultsStub(&stub.Result{
			ID:    tt.id,
			Query: tt.query,
			ResultSet: athena.ResultSet{
				ResultSetMetadata: &athena.ResultSetMetadata{},
				Rows:              []*athena.Row{{}, {}, {}, {}, {}},
			},
		})
		client.MaxPages = tt.maxPages

		q := &Query{
			QueryConfig: cfg,
			client:      client,
			interval:    10 * time.Millisecond,
			query:       tt.query,
			id:          tt.id,
			Result:      &Result{info: tt.info},
		}
		err := q.GetResults()

		assert.NoError(t, err)
		assert.Len(t, q.rs.Rows, tt.numRows, "Query: %s, Id: %s", tt.query, tt.id)
	}
}

func TestGetResultsError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Location: "s3://bucket/prefix/",
	}

	tests := []struct {
		id     string
		query  string
		errMsg string
	}{
		{
			id:     "no_existent_id",
			query:  "SELECT * FROM test_get_result_errors",
			errMsg: "InvalidRequestException",
		},
	}

	for _, tt := range tests {
		q := &Query{
			QueryConfig: cfg,
			client: stub.NewGetQueryResultsStub(&stub.Result{
				ID:     tt.id,
				Query:  tt.query,
				ErrMsg: tt.errMsg,
			}),
			interval: 10 * time.Millisecond,
			query:    tt.query,
			id:       tt.id,
		}
		err := q.GetResults()

		assert.Error(t, err)
	}
}

func TestRun(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Location: "s3://bucket/prefix/",
	}

	tests := []struct {
		id          string
		query       string
		rs          athena.ResultSet
		maxPages    int
		wantNumRows int
	}{
		{
			id:    "TestRun1",
			query: "SELECT * FROM cloudfront_logs LIMIT 5",
			rs: athena.ResultSet{
				ResultSetMetadata: &athena.ResultSetMetadata{},
				Rows:              []*athena.Row{{}, {}, {}, {}, {}},
			},
			maxPages:    2,
			wantNumRows: 10,
		},
	}

	for _, tt := range tests {
		client := stub.NewClient(&stub.Result{
			ID:        tt.id,
			Query:     tt.query,
			ResultSet: tt.rs,
		})
		client.MaxPages = tt.maxPages

		q := &Query{
			QueryConfig: cfg,
			Result:      &Result{},
			client:      client,
			interval:    10 * time.Millisecond,
			query:       tt.query,
		}
		r, err := q.Run()

		assert.NoError(t, err)
		assert.Len(t, r.rs.Rows, tt.wantNumRows, "Query: %#v, Id: %#v", tt.query, tt.id)
	}
}
