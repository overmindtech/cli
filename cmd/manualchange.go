package cmd

import (
	"context"
	"errors"
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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wspb"
)

// manualChangeCmd represents the submit-plan command
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

		exitcode := ManualChange(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func ManualChange(signals chan os.Signal, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx := context.Background()
	ctx, span := tracing.Tracer().Start(ctx, "CLI ManualChange", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	gatewayUrl := viper.GetString("gateway-url")
	if gatewayUrl == "" {
		gatewayUrl = fmt.Sprintf("%v/api/gateway", viper.GetString("url"))
		viper.Set("gateway-url", gatewayUrl)
	}

	lf := log.Fields{}

	ctx, err = ensureToken(ctx, []string{"changes:write"}, signals)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithField("api-key-url", viper.GetString("api-key-url")).WithError(err).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := AuthenticatedChangesClient(ctx)
	changeUuid, err := getChangeUuid(ctx, sdp.ChangeStatus_CHANGE_STATUS_DEFINING, false)
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

		maybeChangeUuid := createResponse.Msg.Change.Metadata.GetUUIDParsed()
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

	receivedItems := []*sdp.Reference{}

	method, err := methodFromString(viper.GetString("query-method"))
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("can't parse --query-method")
		return 1
	}

	u := uuid.New()

	queries := []*sdp.Query{
		{
			UUID:   u[:],
			Method: method,
			Scope:  viper.GetString("query-scope"),
			Type:   viper.GetString("query-type"),
			Query:  viper.GetString("query"),
		},
	}
	options := &websocket.DialOptions{
		HTTPClient: NewAuthenticatedClient(ctx, otelhttp.DefaultClient),
	}

	log.WithContext(ctx).WithFields(lf).WithField("item_count", len(queries)).Info("identifying items")
	// nolint: bodyclose // nhooyr.io/websocket reads the body internally
	c, _, err := websocket.Dial(ctx, viper.GetString("gateway-url"), options)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to connect to overmind API")
		return 1
	}
	defer c.Close(websocket.StatusGoingAway, "")

	// the default, 32kB is too small for cert bundles and rds-db-cluster-parameter-groups
	c.SetReadLimit(2 * 1024 * 1024)

	queriesSentChan := make(chan struct{})
	go func() {
		for _, q := range queries {
			req := sdp.GatewayRequest{
				MinStatusInterval: minStatusInterval,
				RequestType: &sdp.GatewayRequest_Query{
					Query: q,
				},
			}
			err = wspb.Write(ctx, c, &req)

			if err == nil {
				log.WithContext(ctx).WithFields(log.Fields{
					"scope":  q.Scope,
					"type":   q.Type,
					"query":  q.Query,
					"method": q.Method.String(),
					"uuid":   q.ParseUuid().String(),
				}).Trace("Started query")
			}
			if err != nil {
				log.WithContext(ctx).WithFields(lf).WithError(err).WithField("req", &req).Error("Failed to send request")
				continue
			}
		}
		queriesSentChan <- struct{}{}
	}()

	responses := make(chan *sdp.GatewayResponse)

	// Start a goroutine that reads responses
	go func() {
		for {
			res := new(sdp.GatewayResponse)

			err = wspb.Read(ctx, c, res)

			if err != nil {
				var e websocket.CloseError
				if errors.As(err, &e) {
					log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
						"code":   e.Code.String(),
						"reason": e.Reason,
					}).Info("Websocket closing")
					return
				}
				log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to read response")
				return
			}

			responses <- res
		}
	}()

	activeQueries := make(map[uuid.UUID]bool)
	queriesSent := false

	// Read the responses
responses:
	for {
		select {
		case <-queriesSentChan:
			queriesSent = true

		case <-signals:
			log.WithContext(ctx).WithFields(lf).Info("Received interrupt, exiting")
			return 1

		case <-ctx.Done():
			log.WithContext(ctx).WithFields(lf).Info("Context cancelled, exiting")
			return 1

		case resp := <-responses:
			switch resp.ResponseType.(type) {

			case *sdp.GatewayResponse_Status:
				status := resp.GetStatus()
				statusFields := log.Fields{
					"summary":                  status.Summary,
					"responders":               status.Summary.Responders,
					"queriesSent":              queriesSent,
					"post_processing_complete": status.PostProcessingComplete,
				}

				if status.Done() {
					// fall through from all "final" query states, check if there's still queries in progress;
					// only break from the loop if all queries have already been sent
					// TODO: see above, still needs DefaultStartTimeout implemented to account for slow sources
					allDone := allDone(ctx, activeQueries, lf)
					statusFields["allDone"] = allDone
					if allDone && queriesSent {
						log.WithContext(ctx).WithFields(lf).WithFields(statusFields).Info("all responders and queries done")
						break responses
					} else {
						log.WithContext(ctx).WithFields(lf).WithFields(statusFields).Info("all responders done, with unfinished queries")
					}
				} else {
					log.WithContext(ctx).WithFields(lf).WithFields(statusFields).Info("still waiting for responders")
				}

			case *sdp.GatewayResponse_QueryStatus:
				status := resp.GetQueryStatus()
				statusFields := log.Fields{
					"status": status.Status.String(),
				}
				queryUuid := status.GetUUIDParsed()
				if queryUuid == nil {
					log.WithContext(ctx).WithFields(lf).WithFields(statusFields).Debugf("Received QueryStatus with nil UUID")
					continue responses
				}
				statusFields["query"] = queryUuid

				switch status.Status {
				case sdp.QueryStatus_UNSPECIFIED:
					statusFields["unexpected_status"] = true
				case sdp.QueryStatus_STARTED:
					activeQueries[*queryUuid] = true
				case sdp.QueryStatus_FINISHED:
					activeQueries[*queryUuid] = false
				case sdp.QueryStatus_ERRORED:
					activeQueries[*queryUuid] = false
				case sdp.QueryStatus_CANCELLED:
					activeQueries[*queryUuid] = false
				default:
					statusFields["unexpected_status"] = true
				}

				log.WithContext(ctx).WithFields(lf).WithFields(statusFields).Debugf("query status update")

			case *sdp.GatewayResponse_NewItem:
				item := resp.GetNewItem()
				log.WithContext(ctx).WithFields(lf).WithField("item", item.GloballyUniqueName()).Infof("new item")

				receivedItems = append(receivedItems, item.Reference())

			case *sdp.GatewayResponse_NewEdge:
				log.WithContext(ctx).WithFields(lf).Debug("ignored edge")

			case *sdp.GatewayResponse_QueryError:
				err := resp.GetQueryError()
				log.WithContext(ctx).WithFields(lf).WithError(err).Errorf("Error from %v(%v)", err.ResponderName, err.SourceName)

			case *sdp.GatewayResponse_Error:
				err := resp.GetError()
				log.WithContext(ctx).WithFields(lf).WithField(log.ErrorKey, err).Errorf("generic error")

			default:
				j := protojson.Format(resp)
				log.WithContext(ctx).WithFields(lf).Infof("Unknown %T Response:\n%v", resp.ResponseType, j)
			}
		}
	}

	if len(receivedItems) > 0 {
		log.WithContext(ctx).WithFields(lf).WithField("received_items", len(receivedItems)).Info("updating changing items on the change record")
	} else {
		log.WithContext(ctx).WithFields(lf).WithField("received_items", len(receivedItems)).Info("updating change record with no items")
	}
	resultStream, err := client.UpdateChangingItems(ctx, &connect.Request[sdp.UpdateChangingItemsRequest]{
		Msg: &sdp.UpdateChangingItemsRequest{
			ChangeUUID:    changeUuid[:],
			ChangingItems: receivedItems,
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
		if first_log || msg.State != sdp.CalculateBlastRadiusResponse_STATE_DISCOVERING || time_since_last_log > 250*time.Millisecond {
			log.WithContext(ctx).WithFields(lf).WithField("msg", msg).Info("status update")
			last_log = time.Now()
			first_log = false
		}
	}
	if resultStream.Err() != nil {
		log.WithContext(ctx).WithFields(lf).WithError(resultStream.Err()).Error("error streaming results")
		return 1
	}

	changeUrl := fmt.Sprintf("%v/changes/%v", viper.GetString("frontend"), changeUuid)
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

	for _, a := range fetchResponse.Msg.Change.Properties.AffectedAppsUUID {
		appUuid, err := uuid.FromBytes(a)
		if err != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).WithField("app", a).Error("received invalid app uuid")
			continue
		}
		log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
			"change-url": changeUrl,
			"app":        appUuid,
			"app-url":    fmt.Sprintf("%v/apps/%v", viper.GetString("frontend"), appUuid),
		}).Info("affected app")
	}

	return 0
}

func init() {
	rootCmd.AddCommand(manualChangeCmd)

	manualChangeCmd.PersistentFlags().String("changes-url", "", "The changes service API endpoint (defaults to --url)")
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

	manualChangeCmd.PersistentFlags().String("timeout", "3m", "How long to wait for responses")
}
