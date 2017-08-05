package print

import (
	"bytes"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/skatsuta/athenai/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

const outputLocation = "s3://samplebucket/"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{1, "1 B"},
		{12, "12 B"},
		{123, "123 B"},
		{1234, "1.23 KB"},
		{123456, "123.46 KB"},
		{1234567, "1.23 MB"},
		{123456789, "123.46 MB"},
		{1234567890, "1.23 GB"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, FormatBytes(tt.size), "Size: %d", tt.size)
	}
}

func TestPrintFooter(t *testing.T) {
	tests := []struct {
		info     *athena.QueryExecution
		expected string
	}{
		{
			info: &athena.QueryExecution{
				Statistics:          testhelper.CreateStats(1234, 987654321),
				ResultConfiguration: testhelper.CreateResultConfig(outputLocation),
			},
			expected: "Run time: 1.23 seconds | Data scanned: 987.65 MB\nLocation: s3://samplebucket/\n",
		},
		{
			info: &athena.QueryExecution{
				Statistics:          testhelper.CreateStats(10, 10),
				ResultConfiguration: testhelper.CreateResultConfig(outputLocation),
			},
			expected: "Run time: 0.01 seconds | Data scanned: 10 B\nLocation: s3://samplebucket/\n",
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		printFooter(&out, tt.info)

		assert.Equal(t, tt.expected, out.String(), "Info: %#v", tt.info)
	}
}

const (
	showDatabasesTable = `
Query: SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| sampledb        |
+-----------------+
Run time: 0.12 seconds | Data scanned: 0 B
Location: s3://samplebucket/
`

	selectTable = `
Query: SELECT date, time, bytes FROM cloudfront_logs LIMIT 3;
+------------+----------+-------+
| date       | time     | bytes |
| 2014-07-05 | 15:00:00 |  4260 |
| 2014-07-05 | 15:00:00 |    10 |
| 2014-07-05 | 15:00:00 |  4252 |
+------------+----------+-------+
Run time: 1.23 seconds | Data scanned: 56.79 KB
Location: s3://samplebucket/
`

	createDatabaseTable = `
Query: CREATE DATABASE test;
(No output)
Run time: 1.23 seconds | Data scanned: 0 B
Location: s3://samplebucket/
`
)

// mockedResult is a mock struct which implements Result interface for testing.
type mockedResult struct {
	info *athena.QueryExecution
	data [][]string
}

func (m *mockedResult) Info() *athena.QueryExecution {
	return m.info
}

func (m *mockedResult) Rows() [][]string {
	return m.data
}

func TestTablePrint(t *testing.T) {
	tests := []struct {
		r        Result
		expected string
	}{
		{
			r: &mockedResult{
				info: &athena.QueryExecution{
					QueryExecutionId:    aws.String("TestTablePrint_ShowDatabases"),
					Query:               aws.String("SHOW DATABASES"),
					Statistics:          testhelper.CreateStats(123, 0),
					ResultConfiguration: testhelper.CreateResultConfig(outputLocation),
				},
				data: [][]string{
					{"cloudfront_logs"},
					{"elb_logs"},
					{"sampledb"},
				},
			},
			expected: showDatabasesTable,
		},
		{
			r: &mockedResult{
				info: &athena.QueryExecution{
					QueryExecutionId:    aws.String("TestTablePrint_Select"),
					Query:               aws.String("SELECT date, time, bytes FROM cloudfront_logs LIMIT 3"),
					Statistics:          testhelper.CreateStats(1234, 56789),
					ResultConfiguration: testhelper.CreateResultConfig(outputLocation),
				},
				data: [][]string{
					{"date", "time", "bytes"},
					{"2014-07-05", "15:00:00", "4260"},
					{"2014-07-05", "15:00:00", "10"},
					{"2014-07-05", "15:00:00", "4252"},
				},
			},
			expected: selectTable,
		},
		{
			r: &mockedResult{
				info: &athena.QueryExecution{
					QueryExecutionId:    aws.String("TestTablePrint_CreateDatabase"),
					Query:               aws.String("CREATE DATABASE test"),
					Statistics:          testhelper.CreateStats(1234, 0),
					ResultConfiguration: testhelper.CreateResultConfig(outputLocation),
				},
				data: [][]string{},
			},
			expected: createDatabaseTable,
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		out.WriteString("\n")

		tbl := NewTable(&out)
		tbl.Print(tt.r)

		assert.Contains(t, out.String(), tt.expected, "Result: %#v", tt.r)
	}
}

const (
	showDatabasesCSV = `
Query: SHOW DATABASES;
cloudfront_logs
elb_logs
sampledb
Run time: 0.12 seconds | Data scanned: 0 B
Location: s3://samplebucket/
`

	selectCSV = `
Query: SELECT date, time, bytes FROM cloudfront_logs LIMIT 3;
date,time,bytes
2014-07-05,15:00:00,4260
2014-07-05,15:00:00,10
2014-07-05,15:00:00,4252
Run time: 1.23 seconds | Data scanned: 56.79 KB
Location: s3://samplebucket/
`

	createDatabaseCSV = `
Query: CREATE DATABASE test;
(No output)
Run time: 1.23 seconds | Data scanned: 0 B
Location: s3://samplebucket/
`
)

func TestCSVPrint(t *testing.T) {
	tests := []struct {
		r        Result
		expected string
	}{
		{
			r: &mockedResult{
				info: &athena.QueryExecution{
					QueryExecutionId:    aws.String("TestCSVPrint_ShowDatabases"),
					Query:               aws.String("SHOW DATABASES"),
					Statistics:          testhelper.CreateStats(123, 0),
					ResultConfiguration: testhelper.CreateResultConfig(outputLocation),
				},
				data: [][]string{
					{"cloudfront_logs"},
					{"elb_logs"},
					{"sampledb"},
				},
			},
			expected: showDatabasesCSV,
		},
		{
			r: &mockedResult{
				info: &athena.QueryExecution{
					QueryExecutionId:    aws.String("TestCSVPrint_Select"),
					Query:               aws.String("SELECT date, time, bytes FROM cloudfront_logs LIMIT 3"),
					Statistics:          testhelper.CreateStats(1234, 56789),
					ResultConfiguration: testhelper.CreateResultConfig(outputLocation),
				},
				data: [][]string{
					{"date", "time", "bytes"},
					{"2014-07-05", "15:00:00", "4260"},
					{"2014-07-05", "15:00:00", "10"},
					{"2014-07-05", "15:00:00", "4252"},
				},
			},
			expected: selectCSV,
		},
		{
			r: &mockedResult{
				info: &athena.QueryExecution{
					QueryExecutionId:    aws.String("TestCSVPrint_CreateDatabase"),
					Query:               aws.String("CREATE DATABASE test"),
					Statistics:          testhelper.CreateStats(1234, 0),
					ResultConfiguration: testhelper.CreateResultConfig(outputLocation),
				},
				data: [][]string{},
			},
			expected: createDatabaseCSV,
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		out.WriteString("\n")

		csv := NewCSV(&out)
		csv.Print(tt.r)

		assert.Contains(t, out.String(), tt.expected, "Result: %#v", tt.r)
	}
}
