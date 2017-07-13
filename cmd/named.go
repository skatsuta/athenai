package cmd

import (
	"github.com/spf13/cobra"
)

// namedCmd represents the base command for managing named queries.
var namedCmd = &cobra.Command{
	Use:   "named",
	Short: "Manage named queries",
	// TODO
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	// TODO: add exapmles
}

func init() {
	RootCmd.AddCommand(namedCmd)
}
