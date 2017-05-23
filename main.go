package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	humanize "github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

// SUCCEEDED represents a query is succeeded.
const (
	succeeded = "SUCCEEDED"
	failed    = "FAILED"
	cancelled = "CANCELLED"

	sleepDuration = 1
)

func main() {
	region := flag.String("r", "us-east-1", "AWS region")
	query := flag.String("q", "", "The SQL query statement to be executed")
	database := flag.String("d", "", "The database in which the query execution occurs")
	output := flag.String("o", "", "The location in S3 where query results are stored")
	json := flag.Bool("json", false, "Show output as JSON instead of table")
	flag.Parse()

	sess := session.Must(session.NewSession(&aws.Config{Region: region}))
	svc := athena.New(sess)

	startInput := &athena.StartQueryExecutionInput{
		QueryString: query,
		ResultConfiguration: &athena.ResultConfiguration{
			OutputLocation: output,
		},
	}
	if *database != "" {
		startInput.QueryExecutionContext = &athena.QueryExecutionContext{Database: database}
	}

	startOutput, err := svc.StartQueryExecution(startInput)
	if err != nil {
		fail(err)
	}

	id := startOutput.QueryExecutionId
	fmt.Println("[DEBUG] Query execution ID:", *id)
	getOutput, err := waitQuery(svc, id)
	if err != nil {
		fail(err)
	}

	results, err := svc.GetQueryResults(&athena.GetQueryResultsInput{QueryExecutionId: id})
	if err != nil {
		fail(err)
	}

	rs := results.ResultSet
	if len(rs.Rows) == 0 {
		if len(rs.ResultSetMetadata.ColumnInfo) > 0 {
			fmt.Println("No records found")
		}
		return
	}

	if *json {
		fmt.Printf("%#v\n", rs)
	} else {
		table := tablewriter.NewWriter(os.Stdout)
		for _, row := range rs.Rows {
			tabRow := make([]string, len(row.Data))
			for i, data := range row.Data {
				tabRow[i] = *data.VarCharValue
			}
			table.Append(tabRow)
		}
		table.Render()
	}

	stats := getOutput.QueryExecution.Statistics
	scannedBytes := uint64(*stats.DataScannedInBytes)
	execTime := float64(*stats.EngineExecutionTimeInMillis) / 1000
	fmt.Printf("Run time: %f seconds | Data scanned: %s\n", execTime, humanize.Bytes(scannedBytes))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func waitQuery(svc *athena.Athena, queryID *string) (*athena.GetQueryExecutionOutput, error) {
	fmt.Print("Running Query")

	for {
		getOutput, err := svc.GetQueryExecution(&athena.GetQueryExecutionInput{
			QueryExecutionId: queryID,
		})
		if err != nil {
			fail(err)
		}

		state := *getOutput.QueryExecution.Status.State
		reason := getOutput.QueryExecution.Status.StateChangeReason // nil if the execution succeeds
		switch state {
		case succeeded:
			fmt.Println(".")
			return getOutput, nil
		case failed:
			fmt.Println(".")
			return getOutput, fmt.Errorf("query failed: %s", *reason)
		case cancelled:
			fmt.Println(".")
			return getOutput, fmt.Errorf("query cancelled: %s", *reason)
		}

		fmt.Print(".")
		time.Sleep(sleepDuration * time.Second)
	}
}
