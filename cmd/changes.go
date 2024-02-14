package cmd

import (
	"github.com/spf13/cobra"
)

// changesCmd represents the changes command
var changesCmd = &cobra.Command{
	Use:     "changes",
	GroupID: "api",
	Short:   "Create, update and delete changes in Overmind",
	Long: `Manage changes that are being tracked using Overmind. NOTE: It is probably
easier to use our IaC wrappers such as 'overmind terraform plan' rather than
using these commands directly, but they are provided for flexibility.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(changesCmd)

	addAPIFlags(changesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// changesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// changesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
