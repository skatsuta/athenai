package testhelper

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/stretchr/testify/assert"
)

func TestCreateRows(t *testing.T) {
	tests := []struct {
		rawRows [][]string
		want    []*athena.Row
	}{
		{[][]string{}, []*athena.Row{}},
		{[][]string{{}}, []*athena.Row{{Data: []*athena.Datum{}}}},
		{[][]string{{}, {}}, []*athena.Row{{Data: []*athena.Datum{}}, {Data: []*athena.Datum{}}}},
		{
			rawRows: [][]string{{"foo"}},
			want:    []*athena.Row{{Data: []*athena.Datum{new(athena.Datum).SetVarCharValue("foo")}}},
		},
		{
			rawRows: [][]string{{"foo", "bar"}},
			want: []*athena.Row{
				{
					Data: []*athena.Datum{
						new(athena.Datum).SetVarCharValue("foo"),
						new(athena.Datum).SetVarCharValue("bar"),
					},
				},
			},
		},
		{
			rawRows: [][]string{{"foo"}, {"bar"}},
			want: []*athena.Row{
				{Data: []*athena.Datum{new(athena.Datum).SetVarCharValue("foo")}},
				{Data: []*athena.Datum{new(athena.Datum).SetVarCharValue("bar")}},
			},
		},
		{
			rawRows: [][]string{{"foo", "bar"}, {"baz", "foobar"}},
			want: []*athena.Row{
				{
					Data: []*athena.Datum{
						new(athena.Datum).SetVarCharValue("foo"),
						new(athena.Datum).SetVarCharValue("bar"),
					},
				},
				{
					Data: []*athena.Datum{
						new(athena.Datum).SetVarCharValue("baz"),
						new(athena.Datum).SetVarCharValue("foobar"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		got := CreateRows(tt.rawRows)

		assert.Equal(t, tt.want, got, "Raw rows: %#v", tt.rawRows)
	}
}

func TestCreateStats(t *testing.T) {
	tests := []struct {
		execTime int64
		scanned  int64
		want     *athena.QueryExecutionStatistics
	}{
		{
			execTime: 1111,
			scanned:  2222,
			want: &athena.QueryExecutionStatistics{
				EngineExecutionTimeInMillis: aws.Int64(1111),
				DataScannedInBytes:          aws.Int64(2222),
			},
		},
	}

	for _, tt := range tests {
		got := CreateStats(tt.execTime, tt.scanned)

		assert.Equal(t, tt.want, got, "ExecTime: %v, ScannedBytes: %v", tt.execTime, tt.scanned)
	}
}

func TestCreateResultConfig(t *testing.T) {
	want := &athena.ResultConfiguration{
		OutputLocation: aws.String("s3://samplebucket/"),
	}

	loc := "s3://samplebucket/"
	got := CreateResultConfig(loc)

	assert.Equal(t, want, got, "OutputLocation: %s", loc)
}

func TestCreateConfigFile(t *testing.T) {
	cfg := &struct {
		Debug    bool
		Silent   bool
		Section  string
		Profile  string
		Region   string
		Database string
		Location string
		Encrypt  string
		KMS      string
	}{
		Debug:    true,
		Silent:   true,
		Section:  "default",
		Profile:  "default",
		Region:   "us-east-1",
		Database: "sampledb",
		Location: "s3://testbucket/",
		Encrypt:  "SSE_KMS",
		KMS:      "test-kms-key",
	}

	dir, file, cleanup, err := CreateConfigFile("TestCreateConfigFile", cfg)
	defer cleanup()

	assert.NoError(t, err)
	assert.Contains(t, dir, os.TempDir())
	assert.NotNil(t, file)
	assert.Contains(t, file.Name(), "config")

	b, err := ioutil.ReadFile(file.Name()) // somehow ioutil.ReadAll does not work
	data := string(b)

	assert.NoError(t, err)
	assert.Contains(t, data, "[default]")
	assert.Contains(t, data, "profile = default")
}
