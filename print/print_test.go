package print

import (
	"bytes"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/stretchr/testify/assert"
)

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
		assert.Equal(t, tt.expected, formatBytes(tt.size), "Size: %d", tt.size)
	}
}

func TestPrintStats(t *testing.T) {
	tests := []struct {
		info     *athena.QueryExecutionStatistics
		expected string
	}{
		{
			info: &athena.QueryExecutionStatistics{
				EngineExecutionTimeInMillis: aws.Int64(1234),
				DataScannedInBytes:          aws.Int64(987654321),
			},
			expected: "Run time: 1.23 seconds | Data scanned: 987.65 MB\n",
		},
		{
			info: &athena.QueryExecutionStatistics{
				EngineExecutionTimeInMillis: aws.Int64(10),
				DataScannedInBytes:          aws.Int64(10),
			},
			expected: "Run time: 0.01 seconds | Data scanned: 10 B\n",
		},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		printStats(&out, tt.info)
		assert.Equal(t, tt.expected, out.String(), "Info: %#v", tt.info)
	}
}

const (
	showDatabasesTable = `
SHOW DATABASES;
+-----------------+
| cloudfront_logs |
| elb_logs        |
| sampledb        |
+-----------------+`

	selectTable = `
SELECT date, time, bytes FROM cloudfront_logs LIMIT 3;
+------------+----------+-------+
| date       | time     | bytes |
| 2014-07-05 | 15:00:00 |  4260 |
| 2014-07-05 | 15:00:00 |    10 |
| 2014-07-05 | 15:00:00 |  4252 |
+------------+----------+-------+`

	createDatabaseTable = `
CREATE DATABASE test;
(No output)`
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
	stats := &athena.QueryExecutionStatistics{
		EngineExecutionTimeInMillis: aws.Int64(1234),
		DataScannedInBytes:          aws.Int64(987654321),
	}

	tests := []struct {
		r        Result
		expected string
	}{
		{
			r: &mockedResult{
				info: &athena.QueryExecution{
					Query:      aws.String("SHOW DATABASES"),
					Statistics: stats,
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
					Query:      aws.String("SELECT date, time, bytes FROM cloudfront_logs LIMIT 3"),
					Statistics: stats,
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
					Query:      aws.String("CREATE DATABASE test"),
					Statistics: testhelper.CreateStats(1234, 0),
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
