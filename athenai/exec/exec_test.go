package exec

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockedStartQueryExecution struct {
	athenaiface.AthenaAPI
	id string
}

func (m *mockedStartQueryExecution) StartQueryExecution(input *athena.StartQueryExecutionInput) (*athena.StartQueryExecutionOutput, error) {
	query := aws.StringValue(input.QueryString)
	for _, kwd := range []string{"SELECT", "SHOW"} {
		if strings.HasPrefix(query, kwd) {
			resp := &athena.StartQueryExecutionOutput{
				QueryExecutionId: &m.id,
			}
			return resp, nil
		}
	}
	return nil, errors.Errorf("InvalidRequestException: %q", query)
}

func TestNewError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query string
	}{
		{""},
	}

	for _, tt := range tests {
		q, err := NewQuery(&mockedStartQueryExecution{}, tt.query, cfg)
		assert.NotNil(t, err, "Query: %#v", tt.query)
		assert.Nil(t, q)
	}
}

func TestStart(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query, id, expected string
	}{
		{"SELECT * FROM elb_logs", "1", "1"},
	}

	for _, tt := range tests {
		q, err := NewQuery(&mockedStartQueryExecution{id: tt.id}, tt.query, cfg)
		assert.Nil(t, err)

		err = q.Start()
		assert.Nil(t, err)
		assert.Equal(t, tt.expected, q.id, "Query: %q", tt.query)
	}
}

func TestStartError(t *testing.T) {
	cfg := &QueryConfig{
		Database: "sampledb",
		Output:   "s3://bucket/prefix/",
	}

	tests := []struct {
		query, errCode string
	}{
		{"SELET * FROM test", "InvalidRequestException"},
	}

	for _, tt := range tests {
		q, err := NewQuery(&mockedStartQueryExecution{}, tt.query, cfg)
		assert.Nil(t, err)

		err = q.Start()
		if assert.NotNil(t, err) {
			assert.Contains(t, err.Error(), tt.errCode, "Query: %q", tt.query)
		}
	}
}
