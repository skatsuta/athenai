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
	runCmd.Use = `run [flags] [queries...]`

	// Define flags
	f := runCmd.Flags()
	f.StringVarP(&config.Database, "database", "d", "", "The name of the database")
	f.StringVarP(&config.Location, "location", "l", "", `The location in S3 where query results are stored. For example, "s3://bucket_name/prefix/"`)
	f.BoolVar(&config.Order, "order", false, "Show query results in the same order as queries")
}

func validateConfigForRun(cfg *athenai.Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	// For `run` command location config is required
	log.Println("Validating output location:", cfg.Location)
	if !strings.HasPrefix(cfg.Location, "s3://") {
		return &ValidationError{
			Cmd:  "run",
			Name: "location",
			Msg: "valid `location` setting starting with 's3://' is required for the `run` command. " +
				"Please specify it by using --location/-l flag or adding `location` setting into your config file.",
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

func runRun(cmd *cobra.Command, args []string, client athenaiface.AthenaAPI, cfg *athenai.Config, stdin statReader, out io.Writer) error {
	if err := validateConfigForRun(cfg); err != nil {
		return err
	}

	a := athenai.New(client, out, cfg)

	// Read data on stdin and add it to args
	if hasDataOn(stdin) {
		log.Println("Stdin seems to have some data. Reading and appending it to args")
		args = appendStdinData(args, stdin)
	}

	// Run the given queries
	l := len(args)
	if l > 0 {
		log.Printf("%d args provided: %#v\n", l, args)
		a.RunQuery(args)
		return nil
	}

	// Run REPL mode
	log.Printf("No args provided. Starting REPL mode")
	return a.RunREPL()
}
