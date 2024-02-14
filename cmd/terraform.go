package cmd

import (
	"github.com/spf13/cobra"
)

// terraformCmd represents the terraform command
var terraformCmd = &cobra.Command{
	Use:     "terraform",
	GroupID: "iac",
	Short:   "Run Terrafrom with Overmind's change tracking - COMING SOON",
	Long:    `COMING SOON`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(terraformCmd)

	// Hide this flag from the Terraform help as we don't want it to be messy
	rootCmd.PersistentFlags().MarkHidden("url")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// terraformCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// terraformCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
