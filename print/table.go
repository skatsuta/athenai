package print

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	humanize "github.com/dustin/go-humanize" // TODO: drop go-humanize dependency
	"github.com/olekukonko/tablewriter"
	"github.com/skatsuta/athenai/exec"
)

// Table is a filter that formats its input as a table in the output.
type Table struct {
	t *tablewriter.Table
}

// NewTable creates a new Table which writes its output to w.
func NewTable(w io.Writer) *Table {
	return &Table{
		t: tablewriter.NewWriter(w),
	}
}

// Print prints the result in tabular form.
func (t *Table) Print(r *exec.Result) {
	for _, row := range r.ResultSet.Rows {
		tr := make([]string, len(row.Data))
		for i, d := range row.Data {
			tr[i] = aws.StringValue(d.VarCharValue)
		}
		t.t.Append(tr)
	}
	t.t.Render()

	stats := r.Info.Statistics
	runTime := float64(aws.Int64Value(stats.EngineExecutionTimeInMillis)) / 1000
	scannedBytes := uint64(aws.Int64Value(stats.DataScannedInBytes))
	fmt.Printf("Run time: %.3f seconds | Data scanned: %s\n", runTime, humanize.Bytes(scannedBytes))
}
