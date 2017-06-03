package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
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
	Short: "Run the SQL statements.",
	// TODO: fix description
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: runRun,
}

func init() {
	RootCmd.AddCommand(runCmd)

	// Define flags
	f := runCmd.Flags()
	f.StringVarP(&config.Database, "database", "d", "", "The name of the database")
	f.StringVarP(&config.Output, "output", "o", "", "The location in S3 where query results are stored. For example, s3://bucket_name/prefix/")
	f.BoolVarP(&config.Silent, "silent", "s", false, "Do not show progress messages")
}

func runRun(cmd *cobra.Command, args []string) {
	l := len(args)
	if l != 1 { // TODO: run interactive mode if no argument is given
		cmd.Help()
		return
	}

	// Create a service configuration
	cfg := aws.NewConfig().WithRegion(config.Region)

	// Set log level
	if config.Debug {
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
	resultCh := make(chan *exec.Result)
	errCh := make(chan error)
	doneCh := make(chan struct{})
	var wg sync.WaitGroup

	// Print running messages
	if !config.Silent {
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
		query := stmt // capture locally
		if strings.TrimSpace(query) == "" {
			continue // Skip empty statements
		}
		wg.Add(1)
		go runQuery(client, query, resultCh, errCh, &wg)
	}

	// Monitoring goroutine to notify that all the query executions have finished
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	for {
		select {
		case r := <-resultCh:
			fmt.Print("\n")
			print.NewTable(os.Stdout).Print(r)
		case e := <-errCh:
			fmt.Print("\n")
			fmt.Fprintln(os.Stderr, e)
		case <-doneCh:
			return
		}
	}
}

func runQuery(client athenaiface.AthenaAPI, query string, resultCh chan *exec.Result, errCh chan error, wg *sync.WaitGroup) {
	// Run a query, and send results or an error
	r, err := exec.NewQuery(client, query, &config.QueryConfig).Run()
	if err != nil {
		errCh <- err
	} else {
		resultCh <- r
	}
	wg.Done()
}
