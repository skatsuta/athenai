package core

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/chzyer/readline"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/internal/bytes"
	"github.com/skatsuta/athenai/internal/stub"
	"github.com/skatsuta/athenai/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

const testWaitInterval = 10 * time.Millisecond

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
		got := new(Athenai).splitStmts(tt.queries)

		assert.Len(t, got, tt.wantLen, "Query: %q")
	}
}

func TestShowProgressMsg(t *testing.T) {
	want := "Running query"

	var out bytes.Buffer
	a := &Athenai{
		stdout:          &out,
		stderr:          &out,
		cfg:             &Config{},
		refreshInterval: 5 * time.Millisecond,
		waitInterval:    testWaitInterval,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	a.showProgressMsg(ctx, runningQueryMsg)
	got := out.String()

	assert.Contains(t, got, want)
}

const showDatabasesOutput = `
Query: SHOW DATABASES;
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
		a := New(client, &Config{Silent: true}, &out).WithWaitInterval(testWaitInterval)
		a.RunQuery(tt.query)
		got := out.String()

		assert.Contains(t, got, tt.want, "Query: %q, Id: %s", tt.query, tt.id)
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
		a := New(client, &Config{}, &out).WithWaitInterval(testWaitInterval)
		a.RunQuery("file://" + tmpFile.Name())
		got := out.String()

		assert.Contains(t, got, tt.want, "Query: %q, Id: %s", tt.query, tt.id)

		// Clean up
		err = tmpFile.Close()
		assert.NoError(t, err)
		err = os.Remove(tmpFile.Name())
		assert.NoError(t, err)
	}
}

const selectOutput = `
Query: SELECT date, time, bytes FROM cloudfront_logs LIMIT 3;
+------------+----------+-------+
| date       | time     | bytes |
| 2014-07-05 | 15:00:00 |  4260 |
| 2014-07-05 | 15:00:00 |    10 |
| 2014-07-05 | 15:00:00 |  4252 |
+------------+----------+-------+
Run time: 5.55 seconds | Data scanned: 6.67 KB`

const showTablesOutput = `
Query: SHOW TABLES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| flights_parquet |
+-----------------+
Run time: 1.11 seconds | Data scanned: 2.22 KB`

func TestRunQueryOrdered(t *testing.T) {
	tests := []struct {
		query   string
		results []*stub.Result
		wants   []string
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
					ExecTime:     12345,
					ScannedBytes: 56789,
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
			wants: []string{selectOutput, showDatabasesOutput, showTablesOutput},
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		client := stub.NewClient(tt.results...)
		a := New(client, &Config{Database: "sampledb"}, &out).WithWaitInterval(testWaitInterval)
		a.RunQuery(tt.query)
		got := out.String()

		for _, want := range tt.wants {
			assert.Contains(t, got, want, "Results: %#v", tt.results)
		}
	}
}

const selectCSVOutput = `
Query: SELECT date, time, bytes FROM cloudfront_logs LIMIT 3;
date,time,bytes
2014-07-05,15:00:00,4260
2014-07-05,15:00:00,10
2014-07-05,15:00:00,4252
Run time: 1.11 seconds | Data scanned: 2.22 KB`

func TestRunQueryCSV(t *testing.T) {
	tests := []struct {
		query   string
		results []*stub.Result
		wants   []string
	}{
		{
			query: "SELECT date, time, bytes FROM cloudfront_logs LIMIT 3;",
			results: []*stub.Result{
				{
					ID:           "TestRunQueryCSV_Select",
					Query:        "SELECT date, time, bytes FROM cloudfront_logs LIMIT 3",
					ExecTime:     1111,
					ScannedBytes: 2222,
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
			wants: []string{selectCSVOutput},
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		client := stub.NewClient(tt.results...)
		a := New(client, &Config{Database: "sampledb", Format: "csv"}, &out).
			WithWaitInterval(testWaitInterval)
		a.RunQuery(tt.query)
		got := out.String()

		for _, want := range tt.wants {
			assert.Contains(t, got, want, "Results: %#v", tt.results)
		}
	}
}

func TestRunQueryCanceled(t *testing.T) {
	tests := []struct {
		query   string
		results []*stub.Result
		delay   time.Duration
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
			},
			delay: 10 * time.Millisecond,
			want:  cancelingMsg,
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		client := stub.NewClient(tt.results...)
		a := New(client, &Config{Database: "sampledb"}, &out).
			WithStderr(&out).
			WithWaitInterval(testWaitInterval)

		timer := time.NewTimer(tt.delay)
		go func() {
			<-timer.C
			a.signalCh <- os.Interrupt // Send SIGINT signal to cancel after delay
		}()
		a.RunQuery(tt.query)
		got := out.String()

		assert.Contains(t, got, tt.want, "Query: %q, Results: %#v", tt.query, tt.results)
		for _, r := range tt.results {
			assert.NotContains(t, got, r.Query, "Query: %q, Result: %#v", tt.query, r)
		}
	}
}

func TestSetupREPL(t *testing.T) {
	var out bytes.Buffer
	client := stub.NewClient(&stub.Result{ID: "TestSetupREPL"})
	a := New(client, &Config{}, &out)
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
		a := New(client, &Config{}, &out).WithWaitInterval(testWaitInterval)
		a.stdin = in
		a.rl = rl
		err = a.RunREPL()
		got := out.String()

		assert.NoError(t, err)
		assert.Contains(t, got, tt.want, "Input: %q, Id: %s", tt.input, tt.id)
	}
}

type stubReadline struct {
	query string
	err   error
	cnt   int
}

func (r *stubReadline) Readline() (string, error) {
	if r.cnt > 0 {
		return "", io.EOF
	}
	r.cnt++
	return r.query, r.err
}

func (r *stubReadline) Close() error {
	return nil
}

func TestRunREPLError(t *testing.T) {
	tests := []struct {
		rl   readlineCloser
		want string
	}{
		{
			rl:   &stubReadline{query: "", err: readline.ErrInterrupt},
			want: "",
		},
		{
			rl:   &stubReadline{query: "foo", err: readline.ErrInterrupt},
			want: "To exit,",
		},
		{
			rl:   &stubReadline{query: "", err: io.EOF},
			want: "",
		},
		{
			rl:   &stubReadline{query: "error", err: errors.New("unknown error")},
			want: "",
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		a := New(stub.NewClient(), &Config{}, &out).WithWaitInterval(testWaitInterval)
		a.rl = tt.rl
		err := a.RunREPL()
		got := out.String()

		assert.NoError(t, err)
		assert.Contains(t, got, tt.want, "Readline: %#v", tt.rl)
	}
}

type stubFilter struct {
	lines    []string
	idxs     []int
	selected []string
	errMsg   string
}

func newStubFilter(idxs ...int) *stubFilter {
	f := &stubFilter{
		idxs: idxs,
	}
	return f
}

func (f *stubFilter) SetInput(input string) {
	f.lines = strings.Split(input, "\n")
}

func (f *stubFilter) Run(ctx context.Context) error {
	if f.errMsg != "" {
		return errors.New(f.errMsg)
	}

	for _, idx := range f.idxs {
		f.selected = append(f.selected, f.lines[idx])
	}

	return nil
}

func (f *stubFilter) Len() int {
	return len(f.selected)
}

func (f *stubFilter) Each(fn func(item string) bool) {
	for _, s := range f.selected {
		fn(s)
	}
}

func TestSelectQueryExecutions(t *testing.T) {
	tests := []struct {
		count uint
		idxs  []int
		want  int
	}{
		{
			count: 0,
			idxs:  []int{0, 1, 2},
			want:  3,
		},
		{
			count: 1,
			idxs:  []int{0},
			want:  1,
		},
		{
			count: 50,
			idxs:  []int{0, 1, 2},
			want:  3,
		},
	}

	results := []*stub.Result{
		{
			ID:           "TestFetchQueryExecutions_ShowTables",
			Query:        "SHOW TABLES",
			SubmitTime:   time.Date(2017, 7, 1, 0, 0, 0, 0, time.UTC),
			ExecTime:     1111,
			ScannedBytes: 2222,
		},
		{
			ID:           "TestFetchQueryExecutions_ShowDatabases",
			Query:        "SHOW DATABASES",
			SubmitTime:   time.Date(2017, 7, 1, 1, 0, 0, 0, time.UTC),
			ExecTime:     12345,
			ScannedBytes: 56789,
		},
		{
			ID:           "TestFetchQueryExecutions_Select",
			Query:        "SELECT date, time, bytes FROM cloudfront_logs LIMIT 3",
			SubmitTime:   time.Date(2017, 7, 1, 2, 0, 0, 0, time.UTC),
			ExecTime:     5555,
			ScannedBytes: 6666,
		},
	}

	for _, tt := range tests {
		client := stub.NewClient(results...)
		cfg := &Config{Count: tt.count}
		var out bytes.Buffer
		a := New(client, cfg, &out).WithWaitInterval(testWaitInterval)
		a.f = newStubFilter(tt.idxs...) // Select all items
		got, err := a.selectQueryExecutions(context.Background())

		assert.NoError(t, err)
		assert.Len(t, got, tt.want, "Count: %#v", tt.count)
	}
}

func TestShowResults(t *testing.T) {
	tests := []struct {
		results  []*stub.Result
		idxs     []int
		wants    []string
		notWants []string
	}{
		{
			// When three entries are selected
			results: []*stub.Result{
				{
					ID:           "TestShowResults_ShowTables",
					Query:        "SHOW TABLES",
					SubmitTime:   time.Date(2017, 7, 1, 0, 0, 0, 0, time.UTC),
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
					ID:           "TestShowResults_ShowDatabases",
					Query:        "SHOW DATABASES",
					SubmitTime:   time.Date(2017, 7, 1, 1, 0, 0, 0, time.UTC),
					ExecTime:     12345,
					ScannedBytes: 56789,
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
					ID:           "TestShowResults_Select",
					Query:        "SELECT date, time, bytes FROM cloudfront_logs LIMIT 3",
					SubmitTime:   time.Date(2017, 7, 1, 2, 0, 0, 0, time.UTC),
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
				{ // This entry should be skipped as it's failed
					ID:         "TestShowResults_Failed",
					Query:      "SELECT * FROM err_table",
					FinalState: stub.Failed,
					SubmitTime: time.Date(2017, 7, 1, 3, 0, 0, 0, time.UTC),
				},
			},
			idxs:  []int{0, 1, 2},
			wants: []string{selectOutput, showDatabasesOutput, showTablesOutput},
		},
		// When no entry is selected
		{
			results: []*stub.Result{
				{
					ID:           "TestShowResults_ShowTables",
					Query:        "SHOW TABLES",
					SubmitTime:   time.Date(2017, 7, 1, 0, 0, 0, 0, time.UTC),
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
					ID:           "TestShowResults_ShowDatabases",
					Query:        "SHOW DATABASES",
					SubmitTime:   time.Date(2017, 7, 1, 1, 0, 0, 0, time.UTC),
					ExecTime:     12345,
					ScannedBytes: 56789,
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
					ID:           "TestShowResults_Select",
					Query:        "SELECT date, time, bytes FROM cloudfront_logs LIMIT 3",
					SubmitTime:   time.Date(2017, 7, 1, 2, 0, 0, 0, time.UTC),
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
			idxs:     []int{0},
			wants:    []string{selectOutput},
			notWants: []string{showDatabasesOutput, showTablesOutput},
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		client := stub.NewClient(tt.results...)
		cfg := &Config{Database: "sampledb"}
		a := New(client, cfg, &out).WithWaitInterval(2 * testWaitInterval)
		a.f = newStubFilter(tt.idxs...)
		a.ShowResults()
		got := out.String()

		for _, want := range tt.wants {
			assert.Contains(t, got, want, "Results: %#v", tt.results)
		}
		for _, notWant := range tt.notWants {
			assert.NotContains(t, got, notWant, "Results: %#v", tt.results)
		}
	}
}

func TestShowResultsError(t *testing.T) {
	tests := []struct {
		results []*stub.Result
		errMsg  string
		want    string
	}{
		{
			results: []*stub.Result{
				{
					ID:     "TestShowResultsError_APIError",
					Query:  "SHOW DATABASES",
					ErrMsg: athena.ErrCodeInternalServerException,
				},
			},
			want: athena.ErrCodeInternalServerException,
		},
		{
			results: []*stub.Result{
				{
					ID:    "TestShowResultsError_APIError",
					Query: "SHOW DATABASES",
				},
			},
			errMsg: "temporary error",
			want:   "temporary error",
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		client := stub.NewClient(tt.results...)
		a := New(client, &Config{Database: "sampledb"}, &out).
			WithStderr(&out).
			WithWaitInterval(testWaitInterval)
		f := newStubFilter()
		f.errMsg = tt.errMsg
		a.f = f
		a.ShowResults()
		got := out.String()

		assert.Contains(t, got, tt.want, "Results: %#v", tt.results)
	}
}

func TestShowResultsCanceled(t *testing.T) {
	notWant := "Data scanned"

	var out bytes.Buffer
	r := &stub.Result{
		ID:    "TestShowResultsCanceled",
		Query: "SHOW DATABASES",
	}
	client := stub.NewClient(r)
	a := New(client, &Config{Database: "sampledb"}, &out).
		WithStderr(&out).
		WithWaitInterval(testWaitInterval)
	a.f = newStubFilter()
	a.signalCh <- os.Interrupt
	a.ShowResults()
	got := out.String()

	assert.NotContains(t, got, notWant, "Result: %#v", r)
}

func TestGenerateEntry(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{
			query: "SELECT * FROM table",
			want:  "SELECT * FROM table",
		},
		{
			query: `SELECT *
FROM table
WHERE id = 1
LIMIT 10`,
			want: "SELECT * FROM table WHERE id = 1 LIMIT 10",
		},
	}

	dt := time.Date(2017, 7, 1, 0, 0, 0, 0, time.UTC)
	for _, tt := range tests {
		qx := &athena.QueryExecution{
			Query: &tt.query,
			Status: &athena.QueryExecutionStatus{
				SubmissionDateTime: &dt,
				State:              aws.String(athena.QueryExecutionStateSucceeded),
			},
			Statistics: testhelper.CreateStats(1000, 2000),
		}
		got := generateEntry(qx)

		assert.Contains(t, got, tt.want, "Query: %q", tt.query)
	}
}
