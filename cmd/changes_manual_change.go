package cmd

import (
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/sdp-go"
	"github.com/overmindtech/sdp-go/sdpws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// manualChangeCmd is the equivalent to submit-plan for manual changes
var manualChangeCmd = &cobra.Command{
	Use:    "manual-change [--title TITLE] [--description DESCRIPTION] [--ticket-link URL] --query-scope SCOPE --query-type TYPE --query QUERY",
	Short:  "Creates a new Change from a given query",
	PreRun: PreRunSetup,
	RunE:   ManualChange,
}

func ManualChange(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	app, err := getAppUrl(viper.GetString("frontend"), viper.GetString("app"))
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	method, err := methodFromString(viper.GetString("query-method"))
	if err != nil {
		return flagError{fmt.Sprintf("can't parse --query-method: %v\n\n%v", err, cmd.UsageString())}
	}

	ctx, oi, _, err := login(ctx, cmd, []string{"changes:write"})
	if err != nil {
		return err
	}

	client := AuthenticatedChangesClient(ctx, oi)
	changeUuid, err := getChangeUuid(ctx, oi, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, viper.GetString("ticket-link"), false)
	if err != nil {
		return loggedError{
			err:     err,
			message: "failed to identify change",
		}
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
			return loggedError{
				err:     err,
				message: "failed to create change",
			}
		}

		maybeChangeUuid := createResponse.Msg.GetChange().GetMetadata().GetUUIDParsed()
		if maybeChangeUuid == nil {
			return loggedError{
				err:     err,
				message: "failed to read change id",
			}
		}

		changeUuid = *maybeChangeUuid
		log.WithContext(ctx).WithField("change", changeUuid).Info("created a new change")
	} else {
		log.WithContext(ctx).WithField("change", changeUuid).Info("re-using change")
	}

	lf := log.Fields{"change": changeUuid}

	ws, err := sdpws.DialBatch(ctx, oi.GatewayUrl(), otelhttp.DefaultClient, nil)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to connect to gateway",
		}
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
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to send query",
		}
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
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to update changing items",
		}
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
		return loggedError{
			err:     resultStream.Err(),
			fields:  lf,
			message: "error streaming results",
		}
	}

	app, _ = strings.CutSuffix(app, "/")
	changeUrl := fmt.Sprintf("%v/changes/%v/blast-radius", app, changeUuid)
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl).Info("change ready")
	fmt.Println(changeUrl)

	fetchResponse, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: changeUuid[:],
		},
	})
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "failed to get updated change",
		}
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
			"app-url":    fmt.Sprintf("%v/apps/%v", app, appUuid),
		}).Info("affected app")
	}

	return nil
}

func init() {
	changesCmd.AddCommand(manualChangeCmd)
	addAPIFlags(manualChangeCmd)
	manualChangeCmd.PersistentFlags().String("frontend", "", "The frontend base URL")
	_ = submitPlanCmd.PersistentFlags().MarkDeprecated("frontend", "This flag is no longer used and will be removed in a future release. Use the '--app' flag instead.")
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
