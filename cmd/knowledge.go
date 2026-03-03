package cmd

import (
	"github.com/spf13/cobra"
)

// knowledgeCmd represents the knowledge command
var knowledgeCmd = &cobra.Command{
	Use:     "knowledge",
	GroupID: "iac",
	Short:   "Manage tribal knowledge files used for change analysis",
	Long: `Knowledge files in .overmind/knowledge/ help Overmind understand your infrastructure
context, giving better change analysis and risk assessment.

The 'list' subcommand shows which knowledge files Overmind would discover from your
current location, using the same logic as 'overmind terraform plan'.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(knowledgeCmd)
}
