package cmd

import (
	_ "embed"
	"fmt"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/go/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// getSignalsCmd represents the get-signals command
var getSignalsCmd = &cobra.Command{
	Use:   "get-signals {--uuid ID | --change https://app.overmind.tech/changes/c772d072-6b0b-4763-b7c5-ff5069beed4c}",
	Short: "Displays all signals for a change including overview, item, and custom signals.",
	Long: `Displays all signals for a change including:
- Overall signal for the change
- Top level signals for each category
- Routineness signals per item
- Individual custom signals

This provides more detailed signal information than get-change.`,
	PreRun: PreRunSetup,
	RunE:   GetSignals,
}

func GetSignals(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate status flag
	status, err := validateChangeStatus(viper.GetString("status"))
	if err != nil {
		return err
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:read"}, nil)
	if err != nil {
		return err
	}

	changeUuid, err := getChangeUUIDAndCheckStatus(ctx, oi, status, viper.GetString("ticket-link"), true)
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to identify change",
		}
	}

	lf := log.Fields{
		"uuid":       changeUuid.String(),
		"change-url": viper.GetString("change"),
	}

	client := AuthenticatedChangesClient(ctx, oi)
	if err := waitForChangeAnalysis(ctx, client, changeUuid, lf); err != nil {
		return err
	}

	// get the change signals
	var format sdp.ChangeOutputFormat
	switch viper.GetString("format") {
	case "json":
		format = sdp.ChangeOutputFormat_CHANGE_OUTPUT_FORMAT_JSON
	case "markdown":
		format = sdp.ChangeOutputFormat_CHANGE_OUTPUT_FORMAT_MARKDOWN
	default:
		return fmt.Errorf("Unknown output format. Please select 'json' or 'markdown'")
	}
	signalsRes, err := client.GetChangeSignals(ctx, &connect.Request[sdp.GetChangeSignalsRequest]{
		Msg: &sdp.GetChangeSignalsRequest{
			UUID:               changeUuid[:],
			ChangeOutputFormat: format,
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to get change signals",
		}
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"ovm.change.uuid": changeUuid.String(),
	}).Debug("found change signals")

	fmt.Println(signalsRes.Msg.GetSignals())

	return nil
}

func init() {
	changesCmd.AddCommand(getSignalsCmd)
	addAPIFlags(getSignalsCmd)

	addChangeUuidFlags(getSignalsCmd)
	getSignalsCmd.PersistentFlags().String("status", "CHANGE_STATUS_DEFINING", "The expected status of the change. Use this with --ticket-link to get the first change with that status for a given ticket link. Allowed values: CHANGE_STATUS_DEFINING (ready for analysis/analysis in progress), CHANGE_STATUS_HAPPENING (deployment in progress), CHANGE_STATUS_DONE (deployment completed)")

	getSignalsCmd.PersistentFlags().String("frontend", "", "The frontend base URL")
	_ = getSignalsCmd.PersistentFlags().MarkDeprecated("frontend", "This flag is no longer used and will be removed in a future release. Use the '--app' flag instead.") // MarkDeprecated only errors if the flag doesn't exist, we fall back to using app
	getSignalsCmd.PersistentFlags().String("format", "json", "How to render the signals. Possible values: json, markdown")
}
