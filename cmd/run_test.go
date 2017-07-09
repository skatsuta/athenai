package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/athenai"
	"github.com/skatsuta/athenai/internal/stub"
	"github.com/skatsuta/athenai/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

type errReader struct {
	io.Reader
	errMsg string
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.errMsg != "" {
		return 0, errors.New(r.errMsg)
	}
	return r.Reader.Read(p)
}

type stubStatReader struct {
	io.Reader
	errMsg      string
	isDataExist bool
}

func (s *stubStatReader) Stat() (os.FileInfo, error) {
	if s.errMsg != "" {
		return nil, errors.New(s.errMsg)
	}
	return &stubFileInfo{isDataExist: s.isDataExist}, nil
}

type stubFileInfo struct {
	os.FileInfo
	isDataExist bool
}

func (fi *stubFileInfo) Mode() os.FileMode {
	if fi.isDataExist {
		return 0644 // -rw-r--r--
	}
	return 0x04200190 // Dcrw--w----
}

func TestRunRun(t *testing.T) {
	cfg := &athenai.Config{
		Location: "s3://TestRunRunBucket/",
	}

	tests := []struct {
		args     []string
		id       string
		stdin    statReader
		query    string
		rs       athena.ResultSet
		execTime int64
		scanned  int64
		want     []string
	}{
		// When only command line arguments are given
		{
			args:  []string{"SHOW DATABASES"},
			id:    "TestRunRun_ArgsOnly",
			stdin: &stubStatReader{},
			query: "SHOW DATABASES",
			rs: athena.ResultSet{
				ResultSetMetadata: &athena.ResultSetMetadata{},
				Rows: testhelper.CreateRows([][]string{
					{"sampledb"},
					{"s3_logs"},
				}),
			},
			execTime: 1234,
			scanned:  12345,
			want: []string{
				"SHOW DATABASES",
				"sampledb",
				"s3_logs",
				"1.23 seconds",
				"12.35 KB",
			},
		},
		// When no arguments are given (REPL)
		{
			args:  []string{},
			id:    "TestRunRun_NoArgs",
			stdin: &stubStatReader{},
			rs:    athena.ResultSet{},
			want:  []string{""}, // No output in test
		},
		// When no arguments are given but queries are given via stdin
		{
			args: []string{},
			id:   "TestRunRun_ViaStdin",
			stdin: &stubStatReader{
				Reader:      strings.NewReader("SHOW DATABASES;"),
				isDataExist: true,
			},
			query: "SHOW DATABASES",
			rs: athena.ResultSet{
				ResultSetMetadata: &athena.ResultSetMetadata{},
				Rows: testhelper.CreateRows([][]string{
					{"sampledb"},
					{"s3_logs"},
				}),
			},
			execTime: 56789,
			scanned:  123456789,
			want: []string{
				"SHOW DATABASES",
				"sampledb",
				"s3_logs",
				"56.79 seconds",
				"123.46 MB",
			},
		},
		// When an argument and stdin are given but stdin.Stat() fails
		{
			args: []string{"SHOW DATABASES"},
			id:   "TestRunRun_StatFails",
			stdin: &stubStatReader{
				Reader:      strings.NewReader("SHOW TABLES;"),
				errMsg:      "error readnig stdin",
				isDataExist: true,
			},
			query: "SHOW DATABASES",
			rs: athena.ResultSet{
				ResultSetMetadata: &athena.ResultSetMetadata{},
				Rows: testhelper.CreateRows([][]string{
					{"sampledb"},
					{"s3_logs"},
				}),
			},
			execTime: 56789,
			scanned:  123456789,
			want: []string{
				"SHOW DATABASES",
				"sampledb",
				"s3_logs",
				"56.79 seconds",
				"123.46 MB",
			},
		},
		// When an argument and stdin are given but stdin fails to be read
		{
			args: []string{"SHOW DATABASES"},
			id:   "TestRunRun_ReadFails",
			stdin: &stubStatReader{
				Reader: &errReader{
					Reader: strings.NewReader("SHOW TABLES;"),
					errMsg: "error reading stdin",
				},
				isDataExist: true,
			},
			query: "SHOW DATABASES",
			rs: athena.ResultSet{
				ResultSetMetadata: &athena.ResultSetMetadata{},
				Rows: testhelper.CreateRows([][]string{
					{"sampledb"},
					{"s3_logs"},
				}),
			},
			execTime: 56789,
			scanned:  123456789,
			want: []string{
				"SHOW DATABASES",
				"sampledb",
				"s3_logs",
				"56.79 seconds",
				"123.46 MB",
			},
		},
	}

	for _, tt := range tests {
		client := stub.NewClient(&stub.Result{
			ID:           tt.id,
			Query:        tt.query,
			ExecTime:     tt.execTime,
			ScannedBytes: tt.scanned,
			ResultSet:    tt.rs,
		})
		var out bytes.Buffer
		err := runRun(runCmd, tt.args, client, cfg, tt.stdin, &out)
		got := out.String()

		assert.NoError(t, err, "Args: %#v, Id: %#v, ResultSet: %#v", tt.args, tt.id, tt.rs)
		for _, s := range tt.want {
			assert.Contains(t, got, s, "Args: %#v, Id: %#v, ResultSet: %#v", tt.args, tt.id, tt.rs)
		}
	}
}

func TestRunRunValidationError(t *testing.T) {
	tests := []struct {
		id       string
		cfg      *athenai.Config
		wantType *ValidationError
		wantMsg  string
	}{
		{
			id:       "TestRunRunNoLocationError",
			cfg:      &athenai.Config{},
			wantType: &ValidationError{},
			wantMsg:  "location",
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		client := stub.NewClient(&stub.Result{ID: tt.id})
		err := runRun(runCmd, []string{}, client, tt.cfg, os.Stdin, &out)

		if assert.Error(t, err) {
			assert.IsType(t, tt.wantType, err, "Id: %#v, Config: %#v", tt.id, tt.cfg)
			assert.Contains(t, err.Error(), tt.wantMsg, "Id: %#v, Config: %#v", tt.id, tt.cfg)
		}
	}
}
