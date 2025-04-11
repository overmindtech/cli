package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// terraformCmd represents the terraform command
var terraformCmd = &cobra.Command{
	Use:     "terraform",
	GroupID: "iac",
	Short:   "Run Terraform with Overmind's risk analysis and change tracking",
	Long: `By using 'overmind terraform plan/apply' in place of your normal
'terraform plan/apply' commands, you can get a risk analysis and change
tracking for your Terraform changes with no extra effort.

Plan: Overmind will run a normal plan, then determine the potential blast
radius using real-time data from AWS and Kubernetes. It will then analyse the
risks that the changes pose to your infrastructure and return them at the
command line.

Apply: Overmind will do all the same steps as a plan, plus it will take a
snapshot before and after the actual apply, meaning that you get a diff of
everything that happened, including any unexpected repercussions.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(terraformCmd)

	addChangeCreationFlags(terraformCmd)
}

var applyOnlyArgs = []string{
	"auto-approve",
}

var planOnlyArgs = []string{
	"var",
	"var-file",
}

// planArgsFromApplyArgs filters out all apply-specific arguments from arguments
// to `terraform apply`, so that we can run the corresponding `terraform plan`
// command
func planArgsFromApplyArgs(args []string) []string {
	planArgs := []string{}
appendLoop:
	for _, arg := range args {
		for _, applyOnlyArg := range applyOnlyArgs {
			if strings.HasPrefix(arg, "-"+applyOnlyArg) {
				continue appendLoop
			}
			if strings.HasPrefix(arg, "--"+applyOnlyArg) {
				continue appendLoop
			}
		}
		planArgs = append(planArgs, arg)
	}
	return planArgs
}

// applyArgsFromApplyArgs filters out all plan-specific arguments from arguments to `terraform apply`, so that we can run the corresponding `terraform apply` command
func applyArgsFromApplyArgs(args []string) []string {
	applyArgs := []string{}
appendLoop:
	for _, arg := range args {
		for _, planOnlyArg := range planOnlyArgs {
			if strings.HasPrefix(arg, "-"+planOnlyArg) {
				continue appendLoop
			}
			if strings.HasPrefix(arg, "--"+planOnlyArg) {
				continue appendLoop
			}
		}
		applyArgs = append(applyArgs, arg)
	}
	return applyArgs
}
