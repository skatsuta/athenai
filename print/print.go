package print

import (
	"fmt"
	"io"
	"log"
	"math"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/olekukonko/tablewriter"
)

const noOutput = "(No output)"

// Result represents an interface that holds information of a query execution and its results.
type Result interface {
	Info() *athena.QueryExecution
	Rows() [][]string
}

// Printer represents an interface that prints a result.
type Printer interface {
	Print(Result)
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanateBytes(s int64, base float64, sizes []string) string {
	if s < 1000 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := float64(s) / math.Pow(base, e)
	return fmt.Sprintf("%.2f %s", val, suffix)
}

// formatBytes produces a human readable representation of an SI size.
// e.g. Bytes(82854982) -> 82.85 MB
func formatBytes(s int64) string {
	// Implementation of formatBytes is based on github.com/dustin/go-humanize.Bytes().

	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1000, sizes)
}

func printStats(w io.Writer, stats *athena.QueryExecutionStatistics) {
	runTimeMs := aws.Int64Value(stats.EngineExecutionTimeInMillis)
	scannedBytes := aws.Int64Value(stats.DataScannedInBytes)
	log.Printf("EngineExecutionTimeInMillis: %d milliseconds\n", runTimeMs)
	log.Printf("DataScannedInBytes: %d bytes\n", scannedBytes)
	fmt.Fprintf(w, "Run time: %.2f seconds | Data scanned: %s\n", float64(runTimeMs)/1000, formatBytes(scannedBytes))
}

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

func (t *Table) printf(format string, a ...interface{}) {
	fmt.Fprintf(t.w, format, a...)
}

func (t *Table) println(a ...interface{}) {
	fmt.Fprintln(t.w, a...)
}

// Print prints a query executed, its results in tabular form, and the query statistics.
func (t *Table) Print(r Result) {
	if r.Info() == nil || r.Rows() == nil {
		return
	}

	t.printInfo(r.Info())
	t.printTable(r.Rows())
	printStats(t.w, r.Info().Statistics)
}

// printQuery prints a query executed.
func (t *Table) printInfo(info *athena.QueryExecution) {
	t.printf("QueryExecutionId: %s\nQuery: %s;\n",
		aws.StringValue(info.QueryExecutionId), aws.StringValue(info.Query))
}

// printTable prints the results in tabular form.
func (t *Table) printTable(rows [][]string) {
	if len(rows) == 0 {
		t.println(noOutput)
		return
	}

	tw := tablewriter.NewWriter(t.w)
	tw.AppendBulk(rows)
	tw.Render()
}
