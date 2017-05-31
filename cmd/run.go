package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/skatsuta/athenai/exec"
	"github.com/skatsuta/athenai/print"
	"github.com/spf13/cobra"
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
	queryConfig = &exec.QueryConfig{}
)

func init() {
	RootCmd.AddCommand(runCmd)

	// Define flags
	runCmd.Flags().StringVarP(&queryConfig.Database, "database", "d", "", "The name of the database")
	runCmd.Flags().StringVarP(&queryConfig.Output, "output", "o", "", "The location in S3 where query results are stored. For example, s3://bucket_name/prefix/")
}

func runRun(cmd *cobra.Command, args []string) {
	cfg := aws.NewConfig().WithRegion(region)

	// Set log level
	if debug {
		cfg = cfg.WithLogLevel(aws.LogDebug | aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestErrors)
	} else {
		log.SetOutput(ioutil.Discard)
	}

	sess := session.Must(session.NewSession(cfg))
	client := athena.New(sess)

	var query string
	if len(args) > 0 {
		query = args[0]
	}

	// TODO: validate query
	q, err := exec.NewQuery(client, query, queryConfig)
	if err != nil {
		fatal(err)
	}

	resultCh := make(chan *exec.Result)
	errCh := make(chan error)
	tick := time.Tick(1000 * time.Millisecond)

	go func(q *exec.Query, resultCh chan *exec.Result, errCh chan error) {
		r, err := q.Run()
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- r
	}(q, resultCh, errCh)

	fmt.Print("Running query")
	for {
		select {
		case r := <-resultCh:
			fmt.Print("\n")
			print.NewTable(os.Stdout).Print(r)
			return
		case e := <-errCh:
			fatal(e)
		case <-tick:
			fmt.Print(".")
		}
	}
}
