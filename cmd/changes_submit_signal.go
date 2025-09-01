package cmd

import (
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// submitSignalCmd represents the submit-signal command
var submitSignalCmd = &cobra.Command{
	Use:     "submit-signal --title TITLE --description DESCRIPTION [--value VALUE] [--category CATEGORY]",
	Short:   "Creates a custom signal for a change",
	Example: `overmind changes submit-signal --title "Automated testing results" --description "All automated tests passed" --value 5.0 --category Testing`,
	PreRun:  PreRunSetup,
	RunE:    SubmitSignal,
}

func SubmitSignal(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write", "api:write"}, nil)
	if err != nil {
		return err
	}
	// Validate required flags
	if viper.GetString("title") == "" {
		return flagError{"--title is required"}
	}
	value, err := validateValue(viper.GetFloat64("value"))
	if err != nil {
		return flagError{"--value is invalid: " + err.Error()}
	}
	if viper.GetString("description") == "" {
		return flagError{"--description is required"}
	}
	changeUUID, err := getChangeUuid(ctx, oi, sdp.ChangeStatus(sdp.ChangeStatus_value[viper.GetString("status")]), viper.GetString("ticket-link"), true)
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to identify change",
		}
	}

	lf := log.Fields{
		"uuid":       changeUUID.String(),
		"change-url": viper.GetString("change-url"),
	}
	client := AuthenticatedSignalsClient(ctx, oi)
	returnedSignal, err := client.AddSignal(ctx, connect.NewRequest(&sdp.AddSignalRequest{
		Properties: &sdp.SignalProperties{
			Name:        viper.GetString("title"),
			Description: viper.GetString("description"),
			Value:       value,
			Category:    viper.GetString("category"),
		},
		ChangeUUID: changeUUID[:],
	}))
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to create signal",
		}
	}
	if returnedSignal.Msg == nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "signal creation returned no data",
		}
	}

	b, err := json.MarshalIndent(returnedSignal.Msg, "", "  ")
	if err != nil {
		fmt.Printf("Successfully created signal for change %s\n", changeUUID.String())
		log.Infof("Error rendering Signal: %v", err)
	} else {
		fmt.Printf("Successfully created signal for change %s\n", changeUUID.String())
		fmt.Println(string(b))
	}
	return nil
}

func validateValue(value float64) (float64, error) {
	if value < -5.0 || value > 5.0 {
		return 0, fmt.Errorf("must be between -5.0 and 5.0, got %f", value)
	}
	return value, nil
}

func init() {
	changesCmd.AddCommand(submitSignalCmd)

	addAPIFlags(submitSignalCmd)
	addChangeUuidFlags(submitSignalCmd)

	submitSignalCmd.PersistentFlags().String("title", "", "Title of the signal")
	submitSignalCmd.PersistentFlags().String("description", "", "Description of the signal")
	submitSignalCmd.PersistentFlags().Float64("value", 0, "Value of the signal (eg from -5.0 to 5.0, where -5.0 is very bad and 5.0 is very good)")
	submitSignalCmd.PersistentFlags().String("category", string(sdp.SignalCategoryNameCustom), "Category of the signal (eg Custom, etc.)")
}
