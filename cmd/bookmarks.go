/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// bookmarksCmd represents the bookmarks command
var bookmarksCmd = &cobra.Command{
	Use:     "bookmarks",
	GroupID: "api",
	Short:   "Interact with the bookmarks that were created in the Explore view",
	Long: `A bookmark in Overmind is a set of queries that are stored together and can be
executed as a single block.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(bookmarksCmd)

	addAPIFlags(bookmarksCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bookmarksCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bookmarksCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
