package athenai

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
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
		queries []string
		wantLen int
	}{
		{[]string{""}, 0},
		{[]string{";"}, 0},
		{[]string{"; ; \n \t \r   ;"}, 0},
		{[]string{"   ; SELECT;   ; "}, 1},
		{
			[]string{`	;
			SELECT *
			FROM test
			WHERE id = 1;
			SHOW
			TABLES;
			   ;
			`},
			2,
		},
		{[]string{"", ";", "SELECT; SHOW; ", "; DECRIBE"}, 3},
	}

	for _, tt := range tests {
		got := splitStmts(tt.queries)

		assert.Len(t, got, tt.wantLen, "Query: %q")
	}
}

func TestShowProgressMsg(t *testing.T) {
	want := "Running query"

	var out bytes.Buffer
	a := &Athenai{
		out:      &out,
		cfg:      &Config{},
		interval: 1 * time.Millisecond,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	a.showProgressMsg(ctx, runningQueryMsg)

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
		id       string
		query    string
		rs       athena.ResultSet
		execTime int64
		scanned  int64
		want     string
	}{
		{
			id:    "TestRunQuery_EmptyStmt1",
			query: "",
			want:  noStmtFound,
		},
		{
			id:    "TestRunQuery_EmptyStmt2",
			query: "  ; ;  ",
			want:  noStmtFound,
		},
		{
			id:    "TestRunQuery_ShowDBs",
			query: "SHOW DATABASES",
			rs: athena.ResultSet{
				Rows: testhelper.CreateRows([][]string{
					{"cloudfront_logs"},
					{"elb_logs"},
					{"sampledb"},
				}),
			},
			execTime: 12345,
			scanned:  56789,
			want:     showDatabasesOutput,
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		client := stub.NewClient(&stub.Result{
			ID:           tt.id,
			Query:        tt.query,
			ExecTime:     tt.execTime,
			ScannedBytes: tt.scanned,
			ResultSet:    tt.rs,
		})
		a := New(client, &out, &Config{Silent: true})
		a.RunQuery(tt.query)

		assert.Contains(t, out.String(), tt.want, "Query: %q, Id: %s", tt.query, tt.id)
	}
}

func TestRunQueryFromFile(t *testing.T) {
	tests := []struct {
		filename string
		id       string
		query    string
		execTime int64
		scanned  int64
		rs       athena.ResultSet
		want     string
	}{
		{
			filename: "TestRunQueryFromFile1.sql",
			id:       "TestRunQuery_ShowDBs",
			query:    "SHOW DATABASES",
			execTime: 12345,
			scanned:  56789,
			rs: athena.ResultSet{
				Rows: testhelper.CreateRows([][]string{
					{"cloudfront_logs"},
					{"elb_logs"},
					{"sampledb"},
				}),
			},
			want: showDatabasesOutput,
		},
	}

	for _, tt := range tests {
		// Write test SQL to a temporary file
		tmpFile, err := ioutil.TempFile("", tt.filename)
		assert.NoError(t, err)
		_, err = tmpFile.WriteString(tt.query)
		assert.NoError(t, err)

		var out bytes.Buffer
		client := stub.NewClient(&stub.Result{
			ID:           tt.id,
			Query:        tt.query,
			ExecTime:     tt.execTime,
			ScannedBytes: tt.scanned,
			ResultSet:    tt.rs,
		})
		a := New(client, &out, &Config{})
		a.RunQuery("file://" + tmpFile.Name())

		assert.Contains(t, out.String(), tt.want, "Query: %q, Id: %s", tt.query, tt.id)

		// Clean up
		err = tmpFile.Close()
		assert.NoError(t, err)
		err = os.Remove(tmpFile.Name())
		assert.NoError(t, err)
	}
}

const threeStmtsOutputOrdered = `
SELECT date, time, bytes FROM cloudfront_logs LIMIT 3;
+------------+----------+-------+
| date       | time     | bytes |
| 2014-07-05 | 15:00:00 |  4260 |
| 2014-07-05 | 15:00:00 |    10 |
| 2014-07-05 | 15:00:00 |  4252 |
+------------+----------+-------+
Run time: 5.56 seconds | Data scanned: 6.67 KB
.*
SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| sampledb        |
+-----------------+
Run time: 3.33 seconds | Data scanned: 4.44 KB
.*
SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 1.11 seconds | Data scanned: 2.22 KB
`

func TestRunQueryOrdered(t *testing.T) {
	tests := []struct {
		query   string
		results []*stub.Result
		want    string
	}{
		{
			query: "SELECT date, time, bytes FROM cloudfront_logs LIMIT 3; SHOW DATABASES; SHOW TABLES;",
			results: []*stub.Result{ // Arrange in descending order
				{
					ID:           "TestRunQueryOrderedShowTables",
					Query:        "SHOW TABLES",
					ExecTime:     1111,
					ScannedBytes: 2222,
					ResultSet: athena.ResultSet{
						ResultSetMetadata: &athena.ResultSetMetadata{},
						Rows: testhelper.CreateRows([][]string{
							{"cloudfront_logs"},
							{"elb_logs"},
							{"flights_parquet"},
						}),
					},
				},
				{
					ID:           "TestRunQueryOrderedShowDatabases",
					Query:        "SHOW DATABASES",
					ExecTime:     3333,
					ScannedBytes: 4444,
					ResultSet: athena.ResultSet{
						ResultSetMetadata: &athena.ResultSetMetadata{},
						Rows: testhelper.CreateRows([][]string{
							{"cloudfront_logs"},
							{"elb_logs"},
							{"sampledb"},
						}),
					},
				},
				{
					ID:           "TestRunQueryOrderedSelect",
					Query:        "SELECT date, time, bytes FROM cloudfront_logs LIMIT 3",
					ExecTime:     5555,
					ScannedBytes: 6666,
					ResultSet: athena.ResultSet{
						ResultSetMetadata: &athena.ResultSetMetadata{},
						Rows: testhelper.CreateRows([][]string{
							{"date", "time", "bytes"},
							{"2014-07-05", "15:00:00", "4260"},
							{"2014-07-05", "15:00:00", "10"},
							{"2014-07-05", "15:00:00", "4252"},
						}),
					},
				},
			},
			want: threeStmtsOutputOrdered,
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		client := stub.NewClient(tt.results...)
		a := New(client, &out, &Config{Order: true, Database: "sampledb"})
		a.RunQuery(tt.query)

		assert.Regexp(t, tt.want, out.String(), "Results: %#v", tt.results)
	}
}

func TestSetupREPL(t *testing.T) {
	var out bytes.Buffer
	client := stub.NewClient(&stub.Result{ID: "TestSetupREPL"})
	a := New(client, &out, &Config{})
	err := a.setupREPL()

	assert.NoError(t, err)
	assert.NotNil(t, a.rl)
}

func TestRunREPL(t *testing.T) {
	tests := []struct {
		input    string
		id       string
		execTime int64
		scanned  int64
		rs       athena.ResultSet
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
			want:  noStmtFound,
		},
		{
			input:    "SHOW DATABASES\n",
			id:       "TestRunREPL_ShowDBs",
			execTime: 12345,
			scanned:  56789,
			rs: athena.ResultSet{
				Rows: testhelper.CreateRows([][]string{
					{"cloudfront_logs"},
					{"elb_logs"},
					{"sampledb"},
				}),
			},
			want: showDatabasesOutput,
		},
	}

	for _, tt := range tests {
		in := strings.NewReader(tt.input)
		var out bytes.Buffer
		rl, err := readline.NewEx(&readline.Config{
			Stdin:               in,
			Stdout:              &out,
			ForceUseInteractive: true,
		})
		assert.NoError(t, err)

		client := stub.NewClient(&stub.Result{
			ID:           tt.id,
			Query:        strings.TrimSpace(tt.input),
			ExecTime:     tt.execTime,
			ScannedBytes: tt.scanned,
			ResultSet:    tt.rs,
		})
		a := New(client, &out, &Config{})
		a.in = in
		a.rl = rl
		err = a.RunREPL()

		assert.NoError(t, err)
		assert.Contains(t, out.String(), tt.want, "Input: %q, Id: %s", tt.input, tt.id)
	}
}
