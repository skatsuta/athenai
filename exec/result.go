package exec

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
)

// Result represents results of a query execution.
// This struct must implement print.Result interface.
type Result struct {
	info *athena.QueryExecution
	rs   *athena.ResultSet
}

// Info returns information of a query execution.
func (r *Result) Info() *athena.QueryExecution {
	return r.info
}

// Rows returns an array of all rows of the result which contain arrays of columns.
func (r *Result) Rows() [][]string {
	if r == nil || r.rs == nil {
		return nil
	}

	rows := make([][]string, 0, len(r.rs.Rows))
	for _, row := range r.rs.Rows {
		rw := make([]string, len(row.Data))
		for i, d := range row.Data {
			rw[i] = aws.StringValue(d.VarCharValue)
		}
		rows = append(rows, rw)
	}

	return rows
}
