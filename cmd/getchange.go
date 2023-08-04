package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/tracing"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// getChangeCmd represents the get-change command
var getChangeCmd = &cobra.Command{
	Use:   "get-change {--uuid ID | --change https://app.overmind.tech/changes/c772d072-6b0b-4763-b7c5-ff5069beed4c}",
	Short: "Displays the contents of a change.",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `get-change` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		exitcode := GetChange(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func GetChange(signals chan os.Signal, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}

	ctx := context.Background()
	ctx, span := tracing.Tracer().Start(ctx, "CLI GetChange", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, err = ensureToken(ctx, signals)
	if err != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	lf := log.Fields{}
	changeUuid, err := getChangeUuid(ctx, sdp.ChangeStatus(sdp.ChangeStatus_value[viper.GetString("status")]))
	if err != nil {
		log.WithError(err).WithFields(lf).Error("failed to identify change")
		return 1
	}

	lf["uuid"] = changeUuid.String()

	client := AuthenticatedChangesClient(ctx)
	response, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"change-url": viper.GetString("change-url"),
		}).Error("failed to get change")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"change-uuid":        uuid.UUID(response.Msg.Change.Metadata.UUID),
		"change-created":     response.Msg.Change.Metadata.CreatedAt.AsTime(),
		"change-name":        response.Msg.Change.Properties.Title,
		"change-description": response.Msg.Change.Properties.Description,
	}).Info("found change")

	switch viper.GetString("format") {
	case "json":
		b, _ := json.MarshalIndent(response.Msg.Change, "", "  ")
		fmt.Println(string(b))
	case "markdown":
		changeUrl := fmt.Sprintf("%v/changes/%v", viper.GetString("frontend"), changeUuid.String())
		if response.Msg.Change.Metadata.NumAffectedApps != 0 || response.Msg.Change.Metadata.NumAffectedItems != 0 {
			// we have affected stuff
			fmt.Printf(`## Blast Radius  &nbsp; ·  &nbsp; [View in Overmind](%v) <img align="center" width="16" src="https://raw.githubusercontent.com/overmindtech/ovm-cli/main/assets/chainLink.png" alt="chain link icon" />

> **Warning**
> Overmind identified potentially affected apps and items as a result of this pull request.

<br>

| <img align="center" width="16" src="https://raw.githubusercontent.com/overmindtech/ovm-cli/main/assets/blastRadiusItems.png" alt="icon for blast radius items" /> &nbsp;Affected items |
| ------------- |
| [%v items](%v) |
`, changeUrl, response.Msg.Change.Metadata.NumAffectedItems, changeUrl)
		} else {
			fmt.Printf(`## Blast Radius  &nbsp; ·  &nbsp; [View in Overmind](%v) <img align="center" width="16" src="https://raw.githubusercontent.com/overmindtech/ovm-cli/main/assets/chainLink.png" alt="chain link icon" />

> **✅ Checks complete**
> Overmind didn't identify any potentially affected apps and items as a result of this pull request.

`, changeUrl)
		}
	}

	return 0
}

func init() {
	rootCmd.AddCommand(getChangeCmd)

	withChangeUuidFlags(getChangeCmd)
	getChangeCmd.PersistentFlags().String("status", "", "The expected status of the change. Use this with --ticket-link. Allowed values: CHANGE_STATUS_UNSPECIFIED, CHANGE_STATUS_DEFINING, CHANGE_STATUS_HAPPENING, CHANGE_STATUS_PROCESSING, CHANGE_STATUS_DONE")

	getChangeCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")
	getChangeCmd.PersistentFlags().String("format", "json", "How to render the change. Possible values: json, markdown")

	getChangeCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
}
