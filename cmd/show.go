package cmd

import (
	"os"

	"github.com/skatsuta/athenai/athenai"
	"github.com/spf13/cobra"
)

// showCmd represents the show command.
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Shows the results of selected query executions",
	Long: `Shows the results of selected query executions that are complete.
You can filter entries interactively, and select multiple query executions to show at a time.`,
	Run: func(cmd *cobra.Command, args []string) {
		athenai.New(newClient(config), config, os.Stdout).ShowResults()
	},
	Example: `  # Show the results of query executions
  $ athenai show

  # Specify the number of entries to list
  $ athenai show --count 100

  # List all of completed query executions (may be very slow)
  $ athenai show --count 0

  # Print the results in CSV format
  $ athenai show --format csv`,
}

func init() {
	RootCmd.AddCommand(showCmd)

	// Define flags
	f := showCmd.Flags()
	f.StringVarP(&config.Format, "format", "f", "table", "The formatting style for command output. Valid values: table, csv")
	f.UintVarP(&config.Count, "count", "c", 50, "The maximum possible number of SUCCEEDED query executions to list")
}
