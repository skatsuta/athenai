package cmd

import (
	"os"

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
	f.StringVarP(&config.Output, "output", "o", "", "The location in S3 where query results are stored. For example, s3://bucket_name/prefix/")
	f.BoolVarP(&config.Silent, "silent", "s", false, "Do not show progress messages")
}

func runRun(cmd *cobra.Command, args []string) {
	l := len(args)
	if l != 1 { // TODO: run interactive mode if no argument is given
		cmd.Help()
		return
	}

	athenai.New(os.Stdout, config).RunQuery(args[0])
}
