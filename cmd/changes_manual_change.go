package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/tracing"
	"github.com/overmindtech/sdp-go"
	"github.com/overmindtech/sdp-go/sdpws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// manualChangeCmd is the equivalent to submit-plan for manual changes
var manualChangeCmd = &cobra.Command{
	Use:   "manual-change [--title TITLE] [--description DESCRIPTION] [--ticket-link URL] --query-scope SCOPE --query-type TYPE --query QUERY",
	Short: "Creates a new Change from a given query",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.Flags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `manual-change` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a goroutine to watch for cancellation signals
		go func() {
			select {
			case <-sigs:
				cancel()
			case <-ctx.Done():
			}
		}()

		exitcode := ManualChange(ctx, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func ManualChange(ctx context.Context, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx, span := tracing.Tracer().Start(ctx, "CLI ManualChange", trace.WithAttributes(
		attribute.String("ovm.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	lf := log.Fields{
		"app": viper.GetString("app"),
	}

	oi, err := NewOvermindInstance(ctx, viper.GetString("app"))
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to get instance data from app")
		return 1
	}

	ctx, _, err = ensureToken(ctx, oi, []string{"changes:write"})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := AuthenticatedChangesClient(ctx, oi)
	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, viper.GetString("ticket-link"),false)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to searching for existing changes")
		return 1
	}

	if changeUuid == uuid.Nil {
		title := changeTitle(viper.GetString("title"))
		createResponse, err := client.CreateChange(ctx, &connect.Request[sdp.CreateChangeRequest]{
			Msg: &sdp.CreateChangeRequest{
				Properties: &sdp.ChangeProperties{
					Title:       title,
					Description: viper.GetString("description"),
					TicketLink:  viper.GetString("ticket-link"),
					Owner:       viper.GetString("owner"),
					// CcEmails:                  viper.GetString("cc-emails"),
				},
			},
		})
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to create change")
			return 1
		}

		maybeChangeUuid := createResponse.Msg.GetChange().GetMetadata().GetUUIDParsed()
		if maybeChangeUuid == nil {
			log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to read change id")
			return 1
		}

		changeUuid = *maybeChangeUuid
		lf["change"] = changeUuid
		log.WithContext(ctx).WithFields(lf).Info("created a new change")
	} else {
		lf["change"] = changeUuid
		log.WithContext(ctx).WithFields(lf).Info("re-using change")
	}

	method, err := methodFromString(viper.GetString("query-method"))
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("can't parse --query-method")
		return 1
	}

	ws, err := sdpws.DialBatch(ctx, oi.GatewayUrl(), otelhttp.DefaultClient, nil)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to connect to gateway")
		return 1
	}

	u := uuid.New()
	q := &sdp.Query{
		UUID:        u[:],
		Method:      method,
		Scope:       viper.GetString("query-scope"),
		Type:        viper.GetString("query-type"),
		Query:       viper.GetString("query"),
		IgnoreCache: true,
	}

	log.WithContext(ctx).WithFields(lf).WithField("item_count", 1).Info("identifying items")
	receivedItems, err := ws.Query(ctx, q)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to send query")
		return 1
	}

	if len(receivedItems) > 0 {
		log.WithContext(ctx).WithFields(lf).WithField("received_items", len(receivedItems)).Info("updating changing items on the change record")
	} else {
		log.WithContext(ctx).WithFields(lf).WithField("received_items", len(receivedItems)).Info("updating change record with no items")
	}

	references := make([]*sdp.Reference, len(receivedItems))
	for i, item := range receivedItems {
		references[i] = item.Reference()
	}
	resultStream, err := client.UpdateChangingItems(ctx, &connect.Request[sdp.UpdateChangingItemsRequest]{
		Msg: &sdp.UpdateChangingItemsRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: references,
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to update changing items")
		return 1
	}

	last_log := time.Now()
	first_log := true
	for resultStream.Receive() {
		msg := resultStream.Msg()

		// log the first message and at most every 250ms during discovery
		// to avoid spanning the cli output
		time_since_last_log := time.Since(last_log)
		if first_log || msg.GetState() != sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING || time_since_last_log > 250*time.Millisecond {
			log.WithContext(ctx).WithFields(lf).WithField("msg", msg).Info("status update")
			last_log = time.Now()
			first_log = false
		}
	}
	if resultStream.Err() != nil {
		log.WithContext(ctx).WithFields(lf).WithError(resultStream.Err()).Error("error streaming results")
		return 1
	}

	frontend, _ := strings.CutSuffix(viper.GetString("frontend"), "/")
	changeUrl := fmt.Sprintf("%v/changes/%v/blast-radius", frontend, changeUuid)
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl).Info("change ready")
	fmt.Println(changeUrl)

	fetchResponse, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("failed to get updated change")
		return 1
	}

	for _, a := range fetchResponse.Msg.GetChange().GetProperties().GetAffectedAppsUUID() {
		appUuid, err := uuid.FromBytes(a)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).WithField("app", a).Error("received invalid app uuid")
			continue
		}
		log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
			"change-url": changeUrl,
			"app":        appUuid,
			"app-url":    fmt.Sprintf("%v/apps/%v", frontend, appUuid),
		}).Info("affected app")
	}

	return 0
}

func init() {
	changesCmd.AddCommand(manualChangeCmd)

	manualChangeCmd.PersistentFlags().String("frontend", "https://app.overmind.tech", "The frontend base URL")

	manualChangeCmd.PersistentFlags().String("title", "", "Short title for this change.")
	manualChangeCmd.PersistentFlags().String("description", "", "Quick description of the change.")
	manualChangeCmd.PersistentFlags().String("ticket-link", "*", "Link to the ticket for this change.")
	manualChangeCmd.PersistentFlags().String("owner", "", "The owner of this change.")
	// manualChangeCmd.PersistentFlags().String("cc-emails", "", "A comma-separated list of emails to keep updated with the status of this change.")

	manualChangeCmd.PersistentFlags().String("query-method", "get", "The method to use (get, list, search)")
	manualChangeCmd.PersistentFlags().String("query-scope", "*", "The scope to query")
	manualChangeCmd.PersistentFlags().String("query-type", "*", "The type to query")
	manualChangeCmd.PersistentFlags().String("query", "", "The actual query to send")
}
