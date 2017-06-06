package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
	Run: runRun,
}

func init() {
	RootCmd.AddCommand(runCmd)

	// Define flags
	f := runCmd.Flags()
	f.StringVarP(&config.Database, "database", "d", "", "The name of the database")
	f.StringVarP(&config.Location, "location", "l", "", `The location in S3 where query results are stored. For example, "s3://bucket_name/prefix/"`)
	f.BoolVarP(&config.Silent, "silent", "s", false, "Do not show progress messages")
}

// hasDataOnStdin returns true if there is something to read on stdin, otherwise false.
func hasDataOnStdin() bool {
	// Based on https://stackoverflow.com/a/26567513
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func readStdin() (string, error) {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return "", errors.Wrap(err, "failed to read data from stdin")
	}
	data := string(b)
	log.Printf(`Data found on stdin:
--------------------
%s
--------------------
`, data)
	return data, nil
}

func runRun(cmd *cobra.Command, args []string) {
	out := os.Stdout
	a := athenai.New(out, config)

	// Read data on stdin and add it to args
	if hasDataOnStdin() {
		data, err := readStdin()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		args = append(args, data)
	}

	// Run the given queries
	if len(args) > 0 {
		a.RunQuery(args)
		return
	}

	// Run REPL mode
	if err := a.RunREPL(); err != nil {
		fatal(err)
	}
}
