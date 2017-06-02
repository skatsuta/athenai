package print

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/olekukonko/tablewriter"
)

// Table is a filter that formats its input as a table in the output.
type Table struct {
	w io.Writer
}

// NewTable creates a new Table which writes its output to w.
func NewTable(w io.Writer) *Table {
	return &Table{
		w: w,
	}
}

// Print prints a query executed, its results in tabular form, and the query statistics.
func (t *Table) Print(r Result) {
	if r.Info() == nil || r.Rows() == nil {
		return
	}

	t.printQuery(r.Info().Query)
	t.printTable(r.Rows())
	printStats(t.w, r.Info().Statistics)
}

// printQuery prints a query executed.
func (t *Table) printQuery(query *string) {
	fmt.Fprintf(t.w, "%s;\n", aws.StringValue(query))
}

// printTable prints the results in tabular form.
func (t *Table) printTable(rows [][]string) {
	tw := tablewriter.NewWriter(t.w)
	tw.AppendBulk(rows)
	tw.Render()
}
