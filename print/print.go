package print

import (
	"encoding/csv"
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

// FormatBytes produces a human readable representation of an SI size.
// e.g. Bytes(82854982) -> 82.85 MB
func FormatBytes(s int64) string {
	// Implementation of formatBytes is based on github.com/dustin/go-humanize.Bytes().

	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1000, sizes)
}

func printStats(w io.Writer, stats *athena.QueryExecutionStatistics) {
	runTimeMs := aws.Int64Value(stats.EngineExecutionTimeInMillis)
	scannedBytes := aws.Int64Value(stats.DataScannedInBytes)
	log.Printf("EngineExecutionTimeInMillis: %d milliseconds\n", runTimeMs)
	log.Printf("DataScannedInBytes: %d bytes\n", scannedBytes)
	fmt.Fprintf(w, "Run time: %.2f seconds | Data scanned: %s\n", float64(runTimeMs)/1000, FormatBytes(scannedBytes))
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

// Print prints a query executed, its results in tabular form, and the query statistics.
func (t *Table) Print(r Result) {
	if r.Info() == nil || r.Rows() == nil {
		return
	}

	printInfo(t.w, r.Info())
	t.printTable(r.Rows())
	printStats(t.w, r.Info().Statistics)
}

// printTable prints the results in tabular form.
func (t *Table) printTable(rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(t.w, noOutput)
		return
	}

	tw := tablewriter.NewWriter(t.w)
	tw.AppendBulk(rows)
	tw.Render()
}

// CSV writes records in CSV format.
type CSV struct {
	w io.Writer
}

// NewCSV creates a new CSV which writes its output to w.
func NewCSV(w io.Writer) *CSV {
	return &CSV{w: w}
}

// Print prints a query execution id, an executed query, its results in CSV form
// and the query statistics.
func (c *CSV) Print(r Result) {
	if r.Info() == nil || r.Rows() == nil {
		return
	}

	printInfo(c.w, r.Info())
	c.printCSV(r.Rows())
	printStats(c.w, r.Info().Statistics)
}

func (c *CSV) printCSV(rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(c.w, noOutput)
		return
	}

	writer := csv.NewWriter(c.w)
	writer.WriteAll(rows)
	writer.Flush()
}

// printInfo prints query information.
func printInfo(w io.Writer, info *athena.QueryExecution) {
	fmt.Fprintf(w, "QueryExecutionId: %s\nQuery: %s;\n",
		aws.StringValue(info.QueryExecutionId), aws.StringValue(info.Query))
}
