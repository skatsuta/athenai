package exec

import (
	"context"
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
		err := q.Start(context.Background())

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
		err := q.Start(context.Background())

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
		{
			id:     "TestWait_SELECT",
			query:  "SELECT * FROM cloudfront_logs",
			status: athena.QueryExecutionStateSucceeded,
		},
		{
			id:     "TestWait_SHOW_TABLES",
			query:  "SHOW TABLES",
			status: athena.QueryExecutionStateSucceeded,
		},
	}

	for _, tt := range tests {
		client := stub.NewGetQueryExecutionStub(&stub.Result{ID: tt.id, Query: tt.query})
		q := NewQuery(client, tt.query, cfg)
		q.id = tt.id
		q.WaitInterval = 10 * time.Millisecond

		err := q.Wait(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, q.Info(), "ID: %s, Query: %s", tt.id, tt.query)
		got := aws.StringValue(q.Info().Status.State)
		assert.Equal(t, tt.status, got, "ID: %s, Query: %s", tt.id, tt.query)
	}
}

func TestWaitFailedError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Location: "s3://bucket/prefix/",
	}

	tests := []struct {
		id     string
		query  string
		errMsg string
		want   string
	}{
		{
			id:    "",
			query: "",
			want:  "", // Just any error is ok
		},
		{
			id:     "TestWaitFailedError_APIError",
			query:  "SELECT * FROM test_wait_error_table",
			errMsg: athena.ErrCodeInternalServerException,
			want:   athena.ErrCodeInternalServerException,
		},
		{
			id:    "TestWaitFailedError_QueryFailed",
			query: "SELECT * FROM test_wait_error_table",
			want:  ErrQueryExecutionFailed.Error(),
		},
	}

	for _, tt := range tests {
		client := stub.NewGetQueryExecutionStub(&stub.Result{
			ID:         tt.id,
			Query:      tt.query,
			FinalState: stub.Failed,
			ErrMsg:     tt.errMsg,
		})
		q := NewQuery(client, tt.query, cfg)
		q.id = tt.id
		q.WaitInterval = 10 * time.Millisecond

		err := q.Wait(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), tt.want, "ID: %s, Query: %q, ErrMsg: %q", tt.id, tt.query, tt.errMsg)
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
			QueryConfig:  cfg,
			client:       client,
			WaitInterval: 10 * time.Millisecond,
			query:        tt.query,
			id:           tt.id,
			Result:       &Result{info: tt.info},
		}
		err := q.GetResults(context.Background())

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
			WaitInterval: 10 * time.Millisecond,
			query:        tt.query,
			id:           tt.id,
		}
		err := q.GetResults(context.Background())

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
			QueryConfig:  cfg,
			Result:       &Result{},
			client:       client,
			WaitInterval: 10 * time.Millisecond,
			query:        tt.query,
		}
		r, err := q.Run(context.Background())

		assert.NoError(t, err)
		assert.Len(t, r.rs.Rows, tt.wantNumRows, "Query: %#v, Id: %#v", tt.query, tt.id)
	}
}

func TestRunCancelledError(t *testing.T) {
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
			id:    "TestRunCancelledError",
			query: "SELECT * FROM test_run_cancelled_error_table",
			want:  ErrQueryExecutionCancelled.Error(),
		},
	}

	for _, tt := range tests {
		client := stub.NewClient(&stub.Result{ID: tt.id, Query: tt.query})
		q := NewQuery(client, tt.query, cfg)
		q.id = tt.id
		q.WaitInterval = 10 * time.Millisecond

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r, err := q.Run(ctx)

		assert.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), tt.want, "ID: %s, Query: %q", tt.id, tt.query)
	}
}
