package athenai

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/skatsuta/athenai/internal/stub"
	"github.com/skatsuta/athenai/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

func TestSplitStmts(t *testing.T) {
	tests := []struct {
		query   string
		wantLen int
	}{
		{"", 0},
		{";", 0},
		{"; ; \n \t \r   ;", 0},
		{"   ; SELECT;   ; ", 1},
		{`	;
			SELECT *
			FROM test
			WHERE id = 1;
			SHOW
			TABLES;
			   ;
			`, 2},
	}

	for _, tt := range tests {
		got := splitStmts(tt.query)

		assert.Len(t, got, tt.wantLen, "Query: %q")
	}
}

func TestShowProgressMsg(t *testing.T) {
	want := "Running query."

	var out bytes.Buffer
	a := &Athenai{
		out:      &out,
		cfg:      &Config{},
		interval: 1 * time.Millisecond,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	a.showProgressMsg(ctx)

	assert.Contains(t, out.String(), want)
}

const showDatabasesOutput = `
SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| sampledb        |
+-----------------+
Run time: 1.00 seconds | Data scanned: 1.00 KB
`

func TestRunQuery(t *testing.T) {
	tests := []struct {
		query    string
		id       string
		rs       athena.ResultSet
		execTime int64
		scanned  int64
		want     string
	}{
		{
			query: "",
			id:    "TestRunQuery_EmptyStmt1",
			want:  "Nothing executed\n",
		},
		{
			query: "  ; ;  ",
			id:    "TestRunQuery_EmptyStmt2",
			want:  "Nothing executed\n",
		},
		{
			query: "SHOW DATABASES",
			id:    "TestRunQuery_ShowDBs",
			rs: athena.ResultSet{
				Rows: testhelper.CreateRows(
					[][]string{
						{"cloudfront_logs"},
						{"elb_logs"},
						{"sampledb"},
					},
				),
			},
			execTime: 1000,
			scanned:  1000,
			want:     showDatabasesOutput,
		},
	}

	var out bytes.Buffer
	for _, tt := range tests {
		a := New(&out, &Config{Silent: true})
		client := stub.NewClient(tt.id)
		client.ResultSet = tt.rs
		stats := new(athena.QueryExecutionStatistics).
			SetEngineExecutionTimeInMillis(tt.execTime).
			SetDataScannedInBytes(tt.scanned)
		client.QueryExecution.SetStatistics(stats).SetQuery(tt.query)
		a.client = client
		a.RunQuery(tt.query)

		assert.Equal(t, tt.want, out.String(), "Query: %q, Id: %s", tt.query, tt.id)

		out.Reset()
	}
}
