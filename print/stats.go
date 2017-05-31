package print

import (
	"fmt"
	"io"
	"log"
	"math"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
)

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
