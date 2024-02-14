package cmd

import (
	"github.com/spf13/cobra"
)

// invitesCmd represents the invites command
var invitesCmd = &cobra.Command{
	Use:     "invites",
	GroupID: "api",
	Short:   "Manage invites for your team to Overmind",
	Long:    `Create and revoke Overmind invitations`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(invitesCmd)

	addAPIFlags(invitesCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// invitesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// invitesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
