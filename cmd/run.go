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
	"github.com/skatsuta/athenai/athenai"
	"github.com/spf13/cobra"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRun(cmd, args, newClient(config), config, os.Stdin, os.Stdout)
	},
	// TODO: add examples
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

	// Define flags
	f := runCmd.Flags()
	f.StringVarP(&config.Database, "database", "d", "", "The name of the database")
	f.StringVarP(&config.Location, "location", "l", "", `The location in S3 where query results are stored. For example, "s3://bucket_name/prefix/"`)
}

func validateConfigForRun(cfg *athenai.Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	// For `run` command location config is required
	if !strings.HasPrefix(cfg.Location, "s3://") {
		return &ValidationError{
			Cmd:  "run",
			Name: "location",
			Msg: "valid `location` config starting with 's3://' is required for the `run` command. " +
				"Please specify it by using --location/-l flag or adding `location` key into your config file.",
		}
	}

	return nil
}

// hasDataOn returns true if there is something to read on s, otherwise false.
func hasDataOn(s stater) bool {
	// Based on https://stackoverflow.com/a/26567513
	stat, err := s.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func appendStdinData(args []string, stdin io.Reader) []string {
	b, err := ioutil.ReadAll(stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Ignoring data on stdin since having failed to read:", err)
		return args
	}

	data := string(b)
	log.Printf(`Data read from stdin:
--------------------
%s
--------------------
`, data)
	return append(args, data)
}

func runRun(cmd *cobra.Command, args []string, client athenaiface.AthenaAPI, cfg *athenai.Config, stdin statReader, out io.Writer) error {
	if err := initConfig(cfg, cmd, os.Args[1:]); err != nil {
		return errors.Wrap(err, "error initializing config")
	}

	if err := validateConfigForRun(cfg); err != nil {
		return err
	}

	a := athenai.New(client, out, cfg)

	// Read data on stdin and add it to args
	if hasDataOn(stdin) {
		args = appendStdinData(args, stdin)
	}

	// Run the given queries
	if len(args) > 0 {
		a.RunQuery(args)
		return nil
	}

	// Run REPL mode
	return a.RunREPL()
}
