package output

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	humanize "github.com/dustin/go-humanize" // TODO: drop go-humanize dependency
	"github.com/olekukonko/tablewriter"
	"github.com/skatsuta/athenai/exec"
)

// TablePrinter prints output as table format.
type TablePrinter struct {
	w io.Writer
}

// NewTablePrinter creates a new TablePrinter which writes output to w.
func NewTablePrinter(w io.Writer) *TablePrinter {
	return &TablePrinter{
		w: w,
	}
}

// Render renders the result as table format.
func (t *TablePrinter) Render(r *exec.Result) {
	table := tablewriter.NewWriter(t.w)
	rs := r.ResultSet
	for _, row := range rs.Rows {
		tabRow := make([]string, len(row.Data))
		for i, data := range row.Data {
			tabRow[i] = aws.StringValue(data.VarCharValue)
		}
		table.Append(tabRow)
	}
	table.Render()

	stats := r.Info.Statistics
	runTime := float64(aws.Int64Value(stats.EngineExecutionTimeInMillis)) / 1000
	scannedBytes := uint64(aws.Int64Value(stats.DataScannedInBytes))
	fmt.Printf("Run time: %.3f seconds | Data scanned: %s\n", runTime, humanize.Bytes(scannedBytes))
}
