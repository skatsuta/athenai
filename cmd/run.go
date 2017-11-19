package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/pkg/errors"
	"github.com/skatsuta/athenai/core"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs (executes) the SQL query statements",
	Long: `Runs (executes) the SQL query statements. You can run queries either on interactive (REPL) mode,
from command line arguments or from an SQL file. Athenai waits for the query executions and shows
the query results in table or CSV format once the executions have finished.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRun(cmd, args, newClient(config), config, os.Stdin, stdout)
	},
	Example: `  # Start interactive (REPL) mode
  $ atheani run
  # Then provide queries to execute
  athenai> SELECT date, time, requestip, method, status FROM cloudfront_logs LIMIT 5;

  # Run queries from command line arguments
  $ athenai run "SELECT date, time, requestip, method, status FROM cloudfront_logs LIMIT 5;"

  # Run queries from an SQL file
  $ athenai run file://sample.sql
  # Or provide queries via stdin
  $ athenai run < sample.sql

  # Specify the database and S3 location to use
  $ athenai run --database sampledb --location s3://sample-bucket/ "SELECT date, time, requestip FROM cloudfront_logs LIMIT 5;"

  # Encrypt the query results in Amazon S3 (e.g. using SSE_KMS)
  $ athenai run --encrypt SSE_KMS --kms $KMS_KEY_ARN "SELECT date, time, requestip FROM cloudfront_logs LIMIT 5;"

  # Print results in CSV format
  $ athenai run --format csv "SELECT date, time, requestip FROM cloudfront_logs LIMIT 5;"

  # Output (save) results to a file
  $ athenai run --output /path/to/file "SELECT date, time, requestip FROM cloudfront_logs LIMIT 5;"`,
}

type stater interface {
	Stat() (os.FileInfo, error)
}

type statReader interface {
	io.Reader
	stater
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.Use = `run [flags] [queries...]`

	// Define flags
	f := runCmd.Flags()
	f.StringVarP(&config.Database, "database", "d", "", "The name of the database")
	f.StringVarP(&config.Location, "location", "l", "", `The location in S3 where query results are stored. For example, "s3://bucket_name/prefix/"`)
	f.StringVarP(&config.Encrypt, "encrypt", "e", "", "The encryption type for encrypting query results in Amazon S3. Valid values: SSE_S3, SSE_KMS, CSE_KMS")
	f.StringVarP(&config.KMS, "kms", "k", "", `The KMS key ARN or ID used when "SSE_KMS" or "CSE_KMS" is specified in the encryption type`)
	f.StringVarP(&config.Format, "format", "f", "table", "The formatting style for command output. Valid values: table, csv")
	f.UintVar(&config.Concurrent, "concurrent", 5, "The maximum number of concurrent query executions at a time. Usually no need to configure this value")
}

func validateConfigForRun(cfg *core.Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	// For `run` command location config is required
	log.Println("Validating output location:", cfg.Location)
	if !strings.HasPrefix(cfg.Location, "s3://") {
		return errors.New("valid `location` setting starting with 's3://' is required for the `run` command.\n" +
			"Please specify it using --location/-l flag or adding `location = s3://...` entry into your config file.")
	}

	// SSE_KMS or CSE_KMS encryption type requires an KMS key ARN or ID
	if cfg.Encrypt == "SSE_KMS" || cfg.Encrypt == "CSE_KMS" {
		log.Printf(`Encryption type "%s" is specified; validating KMS key: %s\n`, cfg.Encrypt, cfg.KMS)
		// We cannot check the validity of KMS ARN or ID locally, so just check whether the KMS option is provided or not
		if cfg.KMS == "" {
			return errors.New(`KMS key ARN or ID is required when you use "SSE_KMS" or "CSE_KMS" encryption type.` +
				"\nPlease specify it using --kms/-k flag or adding `kms = ...` entry into your config file.")
		}
	}

	return nil
}

// hasDataOn returns true if there is something to read on s, otherwise false.
func hasDataOn(s stater) bool {
	// Based on https://stackoverflow.com/a/26567513
	stat, err := s.Stat()
	if err != nil {
		log.Println("Error getting stat of file:", err)
		return false
	}
	log.Printf("File mode: %s (%o)\n", stat.Mode(), stat.Mode())
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func appendStdinData(args []string, stdin io.Reader) []string {
	log.Printf("Args before appending data on stdin: %#v\n", args)
	b, err := ioutil.ReadAll(stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ignoring data on stdin since having failed to read:", err)
		return args
	}

	data := string(b)
	log.Printf(`Read data from stdin:
--------------------
%s
--------------------
`, data)
	return append(args, data)
}

func runRun(cmd *cobra.Command, args []string, client athenaiface.AthenaAPI, cfg *core.Config, stdin statReader, out io.Writer) (err error) {
	if e := validateConfigForRun(cfg); e != nil {
		return errors.Wrap(e, "validation for run command failed")
	}

	a := core.New(client, cfg, out)

	// Read data on stdin and add it to args
	if hasDataOn(stdin) {
		log.Println("Stdin seems to have some data. Reading and appending it to args")
		args = appendStdinData(args, stdin)
	}

	// Run the given queries
	l := len(args)
	if l > 0 {
		log.Printf("%d args provided: %#v\n", l, args)
		a.RunQuery(args...)
		return nil
	}

	// Run REPL mode
	log.Printf("No args provided. Starting REPL mode")
	return a.RunREPL()
}
