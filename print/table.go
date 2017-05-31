package print

import (
	"io"

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

// Print prints the result in tabular form.
func (t *Table) Print(r Result) {
	t.printTable(r)
	printStats(t.w, r.Info().Statistics)
}

func (t *Table) printTable(r Result) {
	tw := tablewriter.NewWriter(t.w)
	for row := range r.Rows() {
		tw.Append(row)
	}
	tw.Render()
}
