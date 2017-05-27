package athenai

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	a := New("us-east-1", "sampledb", "s3://bucket/prefix/")

	assert.Equal(t, "us-east-1", *a.client.(*athena.Athena).Config.Region)
	assert.Equal(t, "sampledb", a.database)
	assert.Equal(t, "s3://bucket/prefix/", a.output)
}

type mockedStartQueryExecution struct {
	athenaiface.AthenaAPI
	id string
}

func (m *mockedStartQueryExecution) StartQueryExecution(input *athena.StartQueryExecutionInput) (*athena.StartQueryExecutionOutput, error) {
	resp := &athena.StartQueryExecutionOutput{
		QueryExecutionId: &m.id,
	}
	return resp, nil
}

func TestRunSingleQuery(t *testing.T) {
	tests := []struct {
		query, id, expected string
	}{
		{"SELECT * FROM elb_logs", "1", "1"},
	}

	for _, tt := range tests {
		a := &Athenai{
			client:   &mockedStartQueryExecution{id: tt.id},
			database: "sampledb",
			output:   "s3://bucket/prefix/",
		}

		id, err := a.RunSingleQuery(tt.query)

		assert.Nil(t, err)
		assert.Equal(t, tt.expected, id, "Query %q", tt.query)
	}
}
