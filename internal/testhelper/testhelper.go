package testhelper

import (
	"github.com/aws/aws-sdk-go/service/athena"
)

// CreateRows creates an array of *athena.Row from an array of string arrays.
func CreateRows(rawRows [][]string) []*athena.Row {
	rows := make([]*athena.Row, len(rawRows))
	for i, row := range rawRows {
		r := &athena.Row{Data: make([]*athena.Datum, len(row))}
		for j, data := range row {
			r.Data[j] = new(athena.Datum).SetVarCharValue(data)
		}
		rows[i] = r
	}
	return rows
}
