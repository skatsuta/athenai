package print

import (
	"bytes"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/stretchr/testify/assert"
)

const (
	showDatabasesTable = `
+-----------------+
| cloudfront_logs |
| elb_logs        |
| sampledb        |
+-----------------+`

	selectTable = `
+------------+----------+-------+
| date       | time     | bytes |
| 2014-07-05 | 15:00:00 |  4260 |
| 2014-07-05 | 15:00:00 |    10 |
| 2014-07-05 | 15:00:00 |  4252 |
+------------+----------+-------+`
)

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
	}

	for _, tt := range tests {
		var out bytes.Buffer
		out.WriteString("\n")

		tbl := NewTable(&out)
		tbl.Print(tt.r)

		assert.Contains(t, out.String(), tt.expected, "Result: %#v", tt.r)
	}
}
