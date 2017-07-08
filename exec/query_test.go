package exec

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/skatsuta/athenai/internal/stub"
	"github.com/stretchr/testify/assert"
)

const testWaitInterval = 10 * time.Millisecond

var cfg = &QueryConfig{
	Database: "sampledb",
	Location: "s3://bucket/prefix/",
}

func newQuery(client athenaiface.AthenaAPI, cfg *QueryConfig, query string) *Query {
	return NewQuery(client, cfg, query).WithWaitInterval(testWaitInterval)
}

func TestStart(t *testing.T) {
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
		q := NewQuery(client, cfg, tt.query)
		err := q.Start(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, tt.want, q.id, "Query: %q", tt.query)
	}
}

func TestStartError(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{
			query: "",
			want:  athena.ErrCodeInvalidRequestException,
		},
		{
			query: "SELET * FROM test",
			want:  athena.ErrCodeInvalidRequestException,
		},
		{
			query: "CREATE INDEX",
			want:  athena.ErrCodeInvalidRequestException,
		},
	}

	for _, tt := range tests {
		client := stub.NewStartQueryExecutionStub(&stub.Result{Query: tt.query})
		q := NewQuery(client, cfg, tt.query)
		err := q.Start(context.Background())

		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), tt.want, "Query: %q", tt.query)
		}
	}
}

func TestWait(t *testing.T) {
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
		q := newQuery(client, cfg, tt.query)
		q.id = tt.id

		err := q.Wait(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, q.Info(), "ID: %s, Query: %s", tt.id, tt.query)
		got := aws.StringValue(q.Info().Status.State)
		assert.Equal(t, tt.status, got, "ID: %s, Query: %s", tt.id, tt.query)
	}
}

func TestWaitFailedError(t *testing.T) {
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
			want:  "failed",
		},
	}

	for _, tt := range tests {
		client := stub.NewGetQueryExecutionStub(&stub.Result{
			ID:         tt.id,
			Query:      tt.query,
			FinalState: stub.Failed,
			ErrMsg:     tt.errMsg,
		})
		q := newQuery(client, cfg, tt.query)
		q.id = tt.id

		err := q.Wait(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), tt.want, "ID: %s, Query: %q, ErrMsg: %q", tt.id, tt.query, tt.errMsg)
	}
}

func TestGetResults(t *testing.T) {
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

		q := newQuery(client, cfg, tt.query)
		q.id = tt.id
		q.info = tt.info
		err := q.GetResults(context.Background())

		assert.NoError(t, err)
		assert.Len(t, q.rs.Rows, tt.numRows, "Query: %s, Id: %s", tt.query, tt.id)
	}
}

func TestFromQxGetResults(t *testing.T) {
	tests := []struct {
		qx       *athena.QueryExecution
		maxPages int
		numRows  int
	}{
		{
			qx: &athena.QueryExecution{
				QueryExecutionId: aws.String("TestFromQxGetResults1"),
				Query:            aws.String("SELECT * FROM cloudfront_logs LIMIT 10"),
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
			ID:    aws.StringValue(tt.qx.QueryExecutionId),
			Query: aws.StringValue(tt.qx.Query),
			ResultSet: athena.ResultSet{
				ResultSetMetadata: &athena.ResultSetMetadata{},
				Rows:              []*athena.Row{{}, {}, {}, {}, {}},
			},
		})
		client.MaxPages = tt.maxPages

		q := NewQueryFromQx(client, cfg, tt.qx).WithWaitInterval(testWaitInterval)
		err := q.GetResults(context.Background())

		assert.NoError(t, err)
		assert.Len(t, q.rs.Rows, tt.numRows, "Qx: %#v", tt.qx)
	}
}

func TestGetResultsError(t *testing.T) {
	tests := []struct {
		id     string
		query  string
		errMsg string
	}{
		{
			id:     "no_existent_id",
			query:  "SELECT * FROM test_get_result_errors",
			errMsg: athena.ErrCodeInvalidRequestException,
		},
	}

	for _, tt := range tests {
		client := stub.NewGetQueryResultsStub(&stub.Result{
			ID:     tt.id,
			Query:  tt.query,
			ErrMsg: tt.errMsg,
		})
		q := newQuery(client, cfg, tt.query)
		q.id = tt.id
		err := q.GetResults(context.Background())

		assert.Error(t, err)
	}
}

func TestRun(t *testing.T) {
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
		q := newQuery(client, cfg, tt.query)
		r, err := q.Run(context.Background())

		assert.NoError(t, err)
		assert.Len(t, r.rs.Rows, tt.wantNumRows, "Query: %#v, Id: %#v", tt.query, tt.id)
	}
}

func TestRunCanceledError(t *testing.T) {
	tests := []struct {
		id    string
		query string
		want  string
	}{
		{
			id:    "TestRunCanceledError",
			query: "SELECT * FROM test_run_canceled_error_table",
			want:  "canceled",
		},
	}

	for _, tt := range tests {
		client := stub.NewClient(&stub.Result{ID: tt.id, Query: tt.query})
		q := newQuery(client, cfg, tt.query)
		q.id = tt.id

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r, err := q.Run(ctx)

		assert.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), tt.want, "ID: %s, Query: %q", tt.id, tt.query)
	}
}
