package athenai

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/chzyer/readline"
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
Run time: 12.35 seconds | Data scanned: 56.79 KB
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
			execTime: 12345,
			scanned:  56789,
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

		assert.Contains(t, out.String(), tt.want, "Query: %q, Id: %s", tt.query, tt.id)

		out.Reset()
	}
}

func TestRunREPL(t *testing.T) {
	tests := []struct {
		input    string
		id       string
		rs       athena.ResultSet
		execTime int64
		scanned  int64
		want     string
	}{
		{
			input: "\n",
			id:    "TestRunREPL_EmptyInput",
			want:  "\n",
		},
		{
			input: " ; ; \n",
			id:    "TestRunREPL_EmptyStmt",
			want:  "Nothing executed",
		},
		{
			input: "SHOW DATABASES\n",
			id:    "TestRunREPL_ShowDBs",
			rs: athena.ResultSet{
				Rows: testhelper.CreateRows(
					[][]string{
						{"cloudfront_logs"},
						{"elb_logs"},
						{"sampledb"},
					},
				),
			},
			execTime: 12345,
			scanned:  56789,
			want:     showDatabasesOutput,
		},
	}

	var out bytes.Buffer
	for _, tt := range tests {
		client := stub.NewClient(tt.id)
		client.ResultSet = tt.rs
		stats := new(athena.QueryExecutionStatistics).
			SetEngineExecutionTimeInMillis(tt.execTime).
			SetDataScannedInBytes(tt.scanned)
		client.QueryExecution.SetStatistics(stats).SetQuery(strings.TrimSpace(tt.input))

		in := strings.NewReader(tt.input)

		rl, err := readline.NewEx(&readline.Config{
			Stdin:               in,
			Stdout:              &out,
			ForceUseInteractive: true,
		})
		assert.NoError(t, err)

		a := New(&out, &Config{Silent: true})
		a.in = in
		a.client = client
		a.rl = rl
		err = a.RunREPL()

		assert.NoError(t, err)
		assert.Contains(t, out.String(), tt.want, "Input: %q, Id: %s", tt.input, tt.id)

		out.Reset()
	}
}
