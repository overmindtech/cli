package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/pterm"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// terraformApplyCmd represents the `terraform apply` command
var terraformApplyCmd = &cobra.Command{
	Use:    "apply [overmind options...] -- [terraform options...]",
	Short:  "Runs `terraform apply` between two full system configuration snapshots for tracking. This will be automatically connected with the Change created by the `plan` command.",
	PreRun: PreRunSetup,
	RunE:   TerraformApply, //   CmdWrapper("apply", []string{"explore:read", "changes:write", "config:write", "request:receive"}, NewTfApplyModel),
}

func TerraformApply(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	// span := trace.SpanFromContext(ctx)

	PTermSetup()

	hasPlanSet := false
	autoApprove := false
	planFile := "overmind.plan"
	if len(args) >= 1 {
		f, err := os.Stat(args[len(args)-1])
		if err == nil && !f.IsDir() {
			// the last argument is a file, check that the previous arg is not
			// one that would eat this as argument
			hasPlanSet = true
			if len(args) >= 2 {
				prev := args[len(args)-2]
				for _, a := range []string{"-backup", "--backup", "-state", "--state", "-state-out", "--state-out"} {
					if prev == a || strings.HasPrefix(prev, a+"=") {
						hasPlanSet = false
						break
					}
				}
			}
		}
		if hasPlanSet {
			planFile = args[len(args)-1]
			autoApprove = true
		}
	}

	planArgs := append([]string{"plan"}, planArgsFromApplyArgs(args)...)

	if !hasPlanSet {
		// if the user has not set a plan, we need to set a temporary file to
		// capture the output for all calculations and to run apply afterwards

		f, err := os.CreateTemp("", "overmind-plan")
		if err != nil {
			log.WithError(err).Fatal("failed to create temporary plan file")
		}

		planFile = f.Name()

		planArgs = append(planArgs, "-out", planFile)
		args = append(args, planFile)

		// check for auto-approval setting on the command line. note that
		// terraform will ignore -auto-approve if a plan file is supplied,
		// therefore we only check for the flag when no plan file is supplied
		for _, a := range args {
			if a == "-auto-approve" || a == "-auto-approve=true" || a == "-auto-approve=TRUE" || a == "--auto-approve" || a == "--auto-approve=true" || a == "--auto-approve=TRUE" {
				autoApprove = true
			}
			if a == "-auto-approve=false" || a == "-auto-approve=FALSE" || a == "--auto-approve=false" || a == "--auto-approve=FALSE" {
				autoApprove = false
			}
		}
	}

	args = append([]string{"apply"}, args...)

	needPlan := !hasPlanSet
	needApproval := !autoApprove

	ctx, oi, _, cleanup, err := StartSources(ctx, cmd, args)
	if err != nil {
		return err
	}
	defer cleanup()

	if needPlan {
		err := TerraformPlanImpl(ctx, cmd, oi, planArgs, planFile)
		if err != nil {
			return err
		}
	}

	if needApproval {
		pterm.Println("")
		pterm.Println("Do you want to perform these actions?")
		pterm.Println("")
		pterm.Println("Terraform will perform the actions described above.")
		result, _ := pterm.DefaultInteractiveTextInput.WithDefaultText("Only 'yes' will be accepted to approve").Show()
		if result != "yes" {
			return errors.New("aborted by user")
		}
	}

	return TerraformApplyImpl(ctx, cmd, oi, args, planFile)
}

func TerraformApplyImpl(ctx context.Context, cmd *cobra.Command, oi sdp.OvermindInstance, args []string, planFile string) error {
	client := AuthenticatedChangesClient(ctx, oi)

	changeUuid, err := func() (uuid.UUID, error) {
		multi := pterm.DefaultMultiPrinter
		_, _ = multi.Start()
		defer func() {
			_, _ = multi.Stop()
		}()

		var err error
		ticketLink := viper.GetString("ticket-link")
		if ticketLink == "" {
			ticketLink, err = getTicketLinkFromPlan(planFile)
			if err != nil {
				return uuid.Nil, err
			}
		}

		changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, ticketLink, true)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to identify change: %w", err)
		}

		startingChangeSnapshotSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Starting Change")

		startStream, err := client.StartChange(ctx, &connect.Request[sdp.StartChangeRequest]{
			Msg: &sdp.StartChangeRequest{
				ChangeUUID: changeUuid[:],
			},
		})
		if err != nil {
			startingChangeSnapshotSpinner.Fail(fmt.Sprintf("Starting Change: %v", err))
			return uuid.Nil, fmt.Errorf("failed to start change: %w", err)
		}

		var startMsg *sdp.StartChangeResponse
		lastLog := time.Now().Add(-1 * time.Minute)
		for startStream.Receive() {
			startMsg = startStream.Msg()
			// print progress every 2 seconds
			if time.Now().After(lastLog.Add(2 * time.Second)) {
				log.WithFields(log.Fields{
					"state": startMsg.GetState(),
					"items": startMsg.GetNumItems(),
					"edges": startMsg.GetNumEdges(),
				}).Trace("progress")
				lastLog = time.Now()
			}
			stateLabel := "unknown"
			switch startMsg.GetState() {
			case sdp.StartChangeResponse_STATE_UNSPECIFIED:
				stateLabel = "unknown"
			case sdp.StartChangeResponse_STATE_TAKING_SNAPSHOT:
				stateLabel = "capturing current state"
			case sdp.StartChangeResponse_STATE_SAVING_SNAPSHOT:
				stateLabel = "saving state"
			case sdp.StartChangeResponse_STATE_DONE:
				stateLabel = "done"
			}
			startingChangeSnapshotSpinner.UpdateText(fmt.Sprintf("Starting Change: %v", snapshotDetail(stateLabel, startMsg.GetNumItems(), startMsg.GetNumEdges())))
		}
		if startStream.Err() != nil {
			startingChangeSnapshotSpinner.Fail(fmt.Sprintf("Starting Change: %v", startStream.Err()))
			return uuid.Nil, startStream.Err()
		}

		startingChangeSnapshotSpinner.Success()
		return changeUuid, nil
	}()

	if err != nil {
		return err
	}

	// apply the args filtering here, after providers have been configured above
	// (which might still need --var and --var-file information)
	err = RunApply(ctx, applyArgsFromApplyArgs(args))
	if err != nil {
		return err
	}

	multi := pterm.DefaultMultiPrinter
	_, _ = multi.Start()
	defer func() {
		_, _ = multi.Stop()
	}()

	endingChangeSnapshotSpinner, _ := pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Ending Change")

	endStream, err := client.EndChange(ctx, &connect.Request[sdp.EndChangeRequest]{
		Msg: &sdp.EndChangeRequest{
			ChangeUUID: changeUuid[:],
		},
	})
	if err != nil {
		endingChangeSnapshotSpinner.Fail(fmt.Sprintf("Ending Change: %v", err))
		return fmt.Errorf("failed to end change: %w", err)
	}

	var endMsg *sdp.EndChangeResponse
	lastLog := time.Now().Add(-1 * time.Minute)
	for endStream.Receive() {
		endMsg = endStream.Msg()
		// print progress every 2 seconds
		if time.Now().After(lastLog.Add(2 * time.Second)) {
			log.WithFields(log.Fields{
				"state": endMsg.GetState(),
				"items": endMsg.GetNumItems(),
				"edges": endMsg.GetNumEdges(),
			}).Trace("progress")
			lastLog = time.Now()
		}
		stateLabel := "unknown"
		switch endMsg.GetState() {
		case sdp.EndChangeResponse_STATE_UNSPECIFIED:
			stateLabel = "unknown"
		case sdp.EndChangeResponse_STATE_TAKING_SNAPSHOT:
			stateLabel = "capturing current state"
		case sdp.EndChangeResponse_STATE_SAVING_SNAPSHOT:
			stateLabel = "saving state"
		case sdp.EndChangeResponse_STATE_DONE:
			stateLabel = "done"
		}
		endingChangeSnapshotSpinner.UpdateText(fmt.Sprintf("Ending Change: %v", snapshotDetail(stateLabel, endMsg.GetNumItems(), endMsg.GetNumEdges())))
	}
	if endStream.Err() != nil {
		endingChangeSnapshotSpinner.Fail(fmt.Sprintf("Ending Change: %v", endStream.Err()))
		return endStream.Err()
	}

	endingChangeSnapshotSpinner.Success()

	return nil
}

func init() {
	terraformCmd.AddCommand(terraformApplyCmd)

	addAPIFlags(terraformApplyCmd)
	addChangeUuidFlags(terraformApplyCmd)
	addTerraformBaseFlags(terraformApplyCmd)
}
