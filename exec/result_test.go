package exec

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/stretchr/testify/assert"
)

func TestRows(t *testing.T) {
	tests := []struct {
		result   *Result
		expected [][]string
	}{
		{
			result: &Result{
				rs: &athena.ResultSet{
					Rows: []*athena.Row{},
				},
			},
			expected: [][]string{},
		},
		{
			result: &Result{
				rs: &athena.ResultSet{
					Rows: []*athena.Row{
						{
							Data: []*athena.Datum{},
						},
					},
				},
			},
			expected: [][]string{[]string{}},
		},
		{
			result: &Result{
				rs: &athena.ResultSet{
					Rows: []*athena.Row{
						{
							Data: []*athena.Datum{
								{VarCharValue: aws.String("foo")},
								{VarCharValue: aws.String("bar")},
								{VarCharValue: aws.String("baz")},
							},
						},
						{
							Data: []*athena.Datum{
								{VarCharValue: aws.String("1")},
								{VarCharValue: aws.String("2")},
								{VarCharValue: aws.String("3")},
							},
						},
					},
				},
			},
			expected: [][]string{
				{"foo", "bar", "baz"},
				{"1", "2", "3"},
			},
		},
	}

	actual := make([][]string, 0, len(tests))
	for _, tt := range tests {
		for row := range tt.result.Rows() {
			actual = append(actual, row)
		}

		assert.Equal(t, tt.expected, actual, "Result: %#v", tt.result)

		actual = actual[:0] // reset
	}
}
