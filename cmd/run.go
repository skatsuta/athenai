package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the SQL query statements.",
	// TODO
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("run called")
	},
}

func init() {
	RootCmd.AddCommand(runCmd)

	// Define flags
	runCmd.Flags().String("database", "d", "", "The name of the database")
	runCmd.Flags().String("output", "o", "", "The location in S3 where query results are stored. For example, s3://bucket_name/prefix/")
}
