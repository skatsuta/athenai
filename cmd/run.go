package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/skatsuta/athenai/exec"
	"github.com/skatsuta/athenai/print"
	"github.com/spf13/cobra"
)

const (
	tickInterval = 1000 * time.Millisecond
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the SQL query statements.",
	// TODO: fix description
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: runRun,
}

var (
	queryConfig exec.QueryConfig
	silent      bool
)

func init() {
	RootCmd.AddCommand(runCmd)

	// Define flags
	runCmd.Flags().StringVarP(&queryConfig.Database, "database", "d", "", "The name of the database")
	runCmd.Flags().StringVarP(&queryConfig.Output, "output", "o", "", "The location in S3 where query results are stored. For example, s3://bucket_name/prefix/")
	runCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Do not show progress messages")

	// Override usage
	// TODO: more friendly usage examples
	runCmd.Use = `athenai run [flags] ["QUERY"]`
}

func runRun(cmd *cobra.Command, args []string) {
	l := len(args)
	if l != 1 { // TODO: run interactive mode if no argument is given
		cmd.Help()
		return
	}

	// Create a service configuration
	cfg := aws.NewConfig().WithRegion(region)

	// Set log level
	if debug {
		cfg = cfg.WithLogLevel(aws.LogDebug | aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestErrors)
	} else {
		// Surpress log outputs
		log.SetOutput(ioutil.Discard)
	}

	// Create an Athena client
	client := athena.New(session.Must(session.NewSession(cfg)))

	// Split SQL statements by semicolons
	stmts := strings.Split(args[0], ";")

	// Create channels
	l = len(stmts)
	resultChs := make([]chan *exec.Result, 0, l)
	errChs := make([]chan error, 0, l)

	// Print running messages
	if !silent {
		go func() {
			tick := time.Tick(tickInterval)
			fmt.Print("Running query")
			for {
				select {
				case <-tick:
					fmt.Print(".")
				}
			}
		}()
	}

	// Run each statement concurrently using goroutine
	for _, stmt := range stmts {
		if strings.TrimSpace(stmt) == "" {
			continue // Skip empty statements
		}

		resultCh := make(chan *exec.Result)
		errCh := make(chan error)
		go runQuery(client, stmt, resultCh, errCh)

		resultChs = append(resultChs, resultCh)
		errChs = append(errChs, errCh)
	}

	l = len(resultChs)
	for i := 0; i < l; i++ {
	loop:
		for {
			select {
			case r := <-resultChs[i]:
				fmt.Print("\n")
				print.NewTable(os.Stdout).Print(r)
				break loop
			case e := <-errChs[i]:
				fmt.Print("\n")
				fmt.Fprintln(os.Stderr, e)
				break loop
			}
		}
	}
}

func runQuery(client athenaiface.AthenaAPI, query string, resultCh chan *exec.Result, errCh chan error) {
	// Run a query, and send results or an error
	r, err := exec.NewQuery(client, query, &queryConfig).Run()
	if err != nil {
		errCh <- err
	} else {
		resultCh <- r
	}
}
