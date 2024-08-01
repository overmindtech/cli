package cmd

import (
	"github.com/spf13/cobra"
)

// integrationsCmd represents the integrations command
var integrationsCmd = &cobra.Command{
	Use:     "integrations",
	GroupID: "api",
	Short:   "Manage integrations with Overmind",
	Long: `Manage integrations with Overmind. These integrations allow you to
integrate Overmind with other tools and services.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(integrationsCmd)

	addAPIFlags(integrationsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// integrationsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// integrationsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
