package cmd

import (
	"context"
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

// changeFromTfplanCmd represents the change-from-tfplan command
var changeFromTfplanCmd = &cobra.Command{
	Use:   "change-from-tfplan [--title TITLE] [--description DESCRIPTION] [--ticket-link URL] [--tfplan FILE]",
	Short: "Creates a new Change from a given terraform plan (in JSON format)",
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		exitcode := ChangeFromTfplan(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

var (
	affecting_resource *sdp.Reference = &sdp.Reference{
		Type:                 "elbv2-load-balancer",
		UniqueAttributeValue: "ingress",
		Scope:                "944651592624.eu-west-2",
	}
	safe_resource *sdp.Reference = &sdp.Reference{
		Type:                 "ec2-security-group",
		UniqueAttributeValue: "sg-09533c300cd1a41c1",
		Scope:                "944651592624.eu-west-2",
	}
)

func ChangeFromTfplan(signals chan os.Signal, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx := context.Background()
	ctx, span := tracing.Tracer().Start(ctx, "CLI ChangeFromTfplan", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	// Connect to the websocket
	log.WithContext(ctx).Debugf("Connecting to overmind API: %v", viper.GetString("url"))

	ctx, err = ensureToken(ctx, signals)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := AuthenticatedChangesClient(ctx)
	createResponse, err := client.CreateChange(ctx, &connect.Request[sdp.CreateChangeRequest]{
		Msg: &sdp.CreateChangeRequest{
			Properties: &sdp.ChangeProperties{
				Title:       viper.GetString("title"),
				Description: viper.GetString("description"),
				TicketLink:  viper.GetString("ticket-link"),
				Owner:       viper.GetString("owner"),
				// CcEmails:                  viper.GetString("cc-emails"),
			},
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("failed to create change")
		return 1
	}
	log.WithContext(ctx).WithFields(log.Fields{
		"url":    viper.GetString("url"),
		"change": createResponse.Msg.Change.Metadata.GetUUIDParsed(),
	}).Info("created a new change")

	var changing_items []*sdp.Reference
	if viper.GetBool("test-affecting") {
		changing_items = []*sdp.Reference{affecting_resource}
	} else {
		changing_items = []*sdp.Reference{safe_resource}
	}

	resultStream, err := client.UpdateChangingItems(ctx, &connect.Request[sdp.UpdateChangingItemsRequest]{
		Msg: &sdp.UpdateChangingItemsRequest{
			ChangeUUID:    createResponse.Msg.Change.Metadata.UUID,
			ChangingItems: changing_items,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url":    viper.GetString("url"),
			"change": createResponse.Msg.Change.Metadata.GetUUIDParsed(),
		}).Error("failed to update changing items")
		return 1
	}

	last_log := time.Now()
	first_log := true
	for resultStream.Receive() {
		if resultStream.Err() != nil {
			log.WithContext(ctx).WithError(err).WithFields(log.Fields{
				"url":    viper.GetString("url"),
				"change": createResponse.Msg.Change.Metadata.GetUUIDParsed(),
			}).Error("error streaming results")
			return 1
		}

		msg := resultStream.Msg()

		// log the first message and at most every 500ms during discovery
		// to avoid spanning the cli output
		time_since_last_log := time.Since(last_log)
		if first_log || msg.State != sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING || time_since_last_log > 250*time.Millisecond {
			log.WithContext(ctx).WithFields(log.Fields{
				"url":    viper.GetString("url"),
				"change": createResponse.Msg.Change.Metadata.GetUUIDParsed(),
				"msg":    msg,
			}).Info("status update")
			last_log = time.Now()
			first_log = false
		}
	}

	log.WithContext(ctx).WithFields(log.Fields{
		"change":     createResponse.Msg.Change.Metadata.GetUUIDParsed(),
		"change-url": fmt.Sprintf("%v/changes/%v", viper.GetString("frontend"), createResponse.Msg.Change.Metadata.GetUUIDParsed()),
	}).Info("change ready")

	fetchResponse, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: createResponse.Msg.Change.Metadata.UUID,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("failed to get updated change")
		return 1
	}
	for _, a := range fetchResponse.Msg.Change.Properties.AffectedAppsUUID {
		appUuid, err := uuid.FromBytes(a)
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(log.Fields{
				"url":   viper.GetString("url"),
				"value": a,
			}).Error("received invalid app uuid")
			continue
		}
		log.WithContext(ctx).WithFields(log.Fields{
			"change":     createResponse.Msg.Change.Metadata.GetUUIDParsed(),
			"change-url": fmt.Sprintf("%v/changes/%v", viper.GetString("frontend"), createResponse.Msg.Change.Metadata.GetUUIDParsed()),
			"app":        appUuid,
			"app-url":    fmt.Sprintf("%v/apps/%v", viper.GetString("frontend"), appUuid),
		}).Info("affected app")
	}

	return 0
}

func init() {
	rootCmd.AddCommand(changeFromTfplanCmd)

	changeFromTfplanCmd.PersistentFlags().String("changes-url", "https://api.prod.overmind.tech/", "The changes service API endpoint")
	changeFromTfplanCmd.PersistentFlags().String("frontend", "https://app.overmind.tech/", "The frontend base URL")

	changeFromTfplanCmd.PersistentFlags().String("terraform", "terraform", "The binary to use for calling terraform. Will be looked up in the system PATH.")
	changeFromTfplanCmd.PersistentFlags().String("tfplan", "./tfplan", "Parse changing items from this terraform plan file.")

	changeFromTfplanCmd.PersistentFlags().String("title", "", "Short title for this change.")
	changeFromTfplanCmd.PersistentFlags().String("description", "", "Quick description of the change.")
	changeFromTfplanCmd.PersistentFlags().String("ticket-link", "*", "Link to the ticket for this change.")
	changeFromTfplanCmd.PersistentFlags().String("owner", "", "The owner of this change.")
	// changeFromTfplanCmd.PersistentFlags().String("cc-emails", "", "A comma-separated list of emails to keep updated with the status of this change.")

	changeFromTfplanCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
	changeFromTfplanCmd.PersistentFlags().Bool("test-affecting", true, "Choose from the hardcoded test data whether to use a resource that is affecting the test app or not.")

	// Bind these to viper
	err := viper.BindPFlags(changeFromTfplanCmd.PersistentFlags())
	if err != nil {
		log.WithError(err).Fatal("could not bind `change-from-tfplan` flags")
	}
}
