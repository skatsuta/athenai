package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/skatsuta/athenai/athenai"
	"github.com/skatsuta/athenai/internal/stub"
	"github.com/skatsuta/athenai/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

type stubStatReader struct {
	io.Reader
	isDataExist bool
}

func (s *stubStatReader) Stat() (os.FileInfo, error) {
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
	tests := []struct {
		args     []string
		id       string
		cfg      *athenai.Config
		stdin    statReader
		query    string
		rs       athena.ResultSet
		execTime int64
		scanned  int64
		want     []string
	}{
		// When only command line arguments are given
		{
			args: []string{"SHOW DATABASES"},
			id:   "ArgsOnly",
			cfg: &athenai.Config{
				Location: "s3://resultbucket/",
			},
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
			args: []string{},
			id:   "NoArgs",
			cfg: &athenai.Config{
				Location: "s3://TestRunRunNoArgsBucket/",
			},
			stdin: &stubStatReader{},
			rs:    athena.ResultSet{},
			want:  []string{""}, // No output in test
		},
		// When no arguments are given but queries are given via stdin
		{
			args: []string{},
			id:   "ViaStdin",
			cfg: &athenai.Config{
				Location: "s3://TestRunRunViaStdinBucket/",
			},
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
	}

	for _, tt := range tests {
		client := stub.NewClient(tt.id).WithResultSet(tt.rs).WithStats(tt.execTime, tt.scanned).WithQuery(tt.query)
		var out bytes.Buffer
		err := runRun(runCmd, tt.args, client, tt.cfg, tt.stdin, &out)
		got := out.String()

		assert.NoError(t, err, "Args: %#v, Id: %#v, Cfg: %#v, ResultSet: %#v", tt.args, tt.id, tt.cfg, tt.rs)
		for _, s := range tt.want {
			assert.Contains(t, got, s, "Args: %#v, Id: %#v, Cfg: %#v, ResultSet: %#v", tt.args, tt.id, tt.cfg, tt.rs)
		}
	}
}

func TestRunRunValidationError(t *testing.T) {
	tests := []struct {
		id   string
		cfg  *athenai.Config
		want *ValidationError
	}{
		{
			id:   "TestRunRunNoLocationError",
			cfg:  &athenai.Config{},
			want: &ValidationError{},
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		err := runRun(runCmd, []string{}, stub.NewClient(tt.id), tt.cfg, os.Stdin, &out)

		assert.Error(t, err)
		assert.IsType(t, tt.want, err, "Id: %#v, Config: %#v", tt.id, tt.cfg)
	}
}
