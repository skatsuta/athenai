package cmd

import (
	"os"

	"github.com/skatsuta/athenai/athenai"
	"github.com/spf13/cobra"
)

// showCmd represents the show command.
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show query results",
	// TODO: fix description
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		athenai.New(newClient(config), config, os.Stdout).ShowResults()
	},
	// TODO: add examples
}

func init() {
	RootCmd.AddCommand(showCmd)

	// Define flags
	f := showCmd.Flags()
	f.StringVarP(&config.Format, "format", "f", "table", "The formatting style for command output. Valid values: table, csv")
	f.UintVarP(&config.Count, "count", "c", 50, "The maximum possible number of SUCCEEDED query executions to be listed")
}
