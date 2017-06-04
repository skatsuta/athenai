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
