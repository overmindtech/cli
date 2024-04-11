package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"connectrpc.com/connect"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// terraformApplyCmd represents the `terraform apply` command
var terraformApplyCmd = &cobra.Command{
	Use:   "apply [overmind options...] -- [terraform options...]",
	Short: "Runs `terraform apply` between two full system configuration snapshots for tracking. This will be automatically connected with the Change created by the `plan` command.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `terraform apply` flags")
		}
	},
	Run: CmdWrapper("apply", []string{"changes:write", "config:write", "request:receive"}, nil),
}

func TerraformApply(ctx context.Context, args []string, oi OvermindInstance, token *oauth2.Token) error {
	cancel, err := InitializeSources(ctx, oi, viper.GetString("aws-config"), viper.GetString("aws-profile"), token)
	defer cancel()
	if err != nil {
		return err
	}

	ticketLink := viper.GetString("ticket-link")
	if ticketLink == "" {
		ticketLink, err = getTicketLinkFromPlan()
		if err != nil {
			return err
		}
	}

	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, ticketLink, true)
	if err != nil {
		return fmt.Errorf("failed to identify change: %w", err)
	}

	client := AuthenticatedChangesClient(ctx, oi)
	startStream, err := client.StartChange(ctx, &connect.Request[sdp.StartChangeRequest]{
		Msg: &sdp.StartChangeRequest{
			ChangeUUID: changeUuid[:],
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start change: %w", err)
	}
	for startStream.Receive() {
		msg := startStream.Msg()
		log.WithFields(log.Fields{
			"state": msg.GetState(),
			"items": msg.GetNumItems(),
			"edges": msg.GetNumEdges(),
		}).Info("progress")
	}
	if startStream.Err() != nil {
		return fmt.Errorf("failed to process start change: %w", startStream.Err())
	}

	args = append([]string{"apply"}, args...)
	// plan file needs to go last
	args = append(args, "overmind.plan")

	prompt := `
* AWS Source: running
* stdlib Source: running

# Applying Changes

Running ` + "`" + `terraform %v` + "`" + `
`

	r := NewTermRenderer()
	out, err := r.Render(fmt.Sprintf(prompt, strings.Join(args, " ")))
	if err != nil {
		panic(err)
	}
	fmt.Print(out)

	tfApplyCmd := exec.CommandContext(ctx, "terraform", args...)
	tfApplyCmd.Stderr = os.Stderr
	tfApplyCmd.Stdout = os.Stdout
	tfApplyCmd.Stdin = os.Stdin

	err = tfApplyCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run terraform apply: %w", err)
	}

	endStream, err := client.EndChange(ctx, &connect.Request[sdp.EndChangeRequest]{
		Msg: &sdp.EndChangeRequest{
			ChangeUUID: changeUuid[:],
		},
	})
	if err != nil {
		return fmt.Errorf("failed to end change: %w", err)
	}
	for endStream.Receive() {
		msg := endStream.Msg()
		log.WithFields(log.Fields{
			"state": msg.GetState(),
			"items": msg.GetNumItems(),
			"edges": msg.GetNumEdges(),
		}).Info("progress")
	}
	if endStream.Err() != nil {
		return fmt.Errorf("failed to process end change: %w", endStream.Err())
	}

	return nil
}

func init() {
	terraformCmd.AddCommand(terraformApplyCmd)

	addAPIFlags(terraformApplyCmd)
	addChangeUuidFlags(terraformApplyCmd)
	addTerraformBaseFlags(terraformApplyCmd)
}
