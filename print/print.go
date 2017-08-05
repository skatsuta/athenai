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

// printer is a filter that formats its input as a table in the output.
type printer struct {
	out io.Writer
	fn  func(w io.Writer, rows [][]string)
}

// New returns a new Printer which prints to out corresponding to format.
func New(out io.Writer, format string) Printer {
	fn := printTable
	if format == "csv" {
		fn = printCSV
	}

	return &printer{
		out: out,
		fn:  fn,
	}
}

func (p *printer) Print(r Result) {
	info := r.Info()
	rows := r.Rows()
	if info == nil || rows == nil {
		return
	}

	printHeader(p.out, info)

	if len(rows) == 0 {
		fmt.Fprintln(p.out, noOutput)
	} else {
		p.fn(p.out, rows)
	}

	printFooter(p.out, info)
}

// printTable prints the results in tabular form.
func printTable(out io.Writer, rows [][]string) {
	tw := tablewriter.NewWriter(out)
	tw.AppendBulk(rows)
	tw.Render()
}

// printCSV prints the results in CSV format.
func printCSV(out io.Writer, rows [][]string) {
	w := csv.NewWriter(out)
	w.WriteAll(rows)
	w.Flush()
}

// printHeader prints query information.
func printHeader(w io.Writer, info *athena.QueryExecution) {
	fmt.Fprintf(w, "Query: %s;\n", aws.StringValue(info.Query))
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

// printFooter prints a footer for a query execution.
func printFooter(w io.Writer, info *athena.QueryExecution) {
	stats := info.Statistics
	runTimeMs := aws.Int64Value(stats.EngineExecutionTimeInMillis)
	scannedBytes := aws.Int64Value(stats.DataScannedInBytes)
	loc := aws.StringValue(info.ResultConfiguration.OutputLocation)
	log.Printf("EngineExecutionTimeInMillis: %d milliseconds\n", runTimeMs)
	log.Printf("DataScannedInBytes: %d bytes\n", scannedBytes)
	log.Printf("OutputLocation: %s\n", loc)
	fmt.Fprintf(w, "Run time: %.2f seconds | Data scanned: %s\nLocation: %s\n",
		float64(runTimeMs)/1000, FormatBytes(scannedBytes), loc)
}
