package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"google.golang.org/protobuf/types/known/durationpb"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wspb"
)

// requestCmd represents the start command
var requestCmd = &cobra.Command{
	Use:   "request",
	Short: "Runs a request against the overmind API",
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		exitcode := Request(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

func Request(signals chan os.Signal, ready chan bool) int {
	timeout, err := time.ParseDuration(viper.GetString("timeout"))
	if err != nil {
		log.Errorf("invalid --timeout value '%v', error: %v", viper.GetString("timeout"), err)
		return 1
	}
	ctx := context.Background()
	ctx, span := tracing.Tracer().Start(ctx, "CLI Request", trace.WithAttributes(
		attribute.String("om.config", fmt.Sprintf("%v", viper.AllSettings())),
	))
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Construct the request
	req, err := createInitialRequest()
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to create initial request")
		return 1
	}

	// Connect to the websocket
	log.WithContext(ctx).Debugf("Connecting to overmind API: %v", viper.GetString("url"))

	ctx, err = ensureToken(ctx, signals)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("failed to authenticate")
		return 1
	}

	options := &websocket.DialOptions{
		HTTPClient: otelhttp.DefaultClient,
	}

	if viper.GetString("token") != "" {
		log.Info("Setting authorization token")
		options.HTTPHeader = make(http.Header)
		options.HTTPHeader.Set("Authorization", fmt.Sprintf("Bearer %v", viper.GetString("token")))
	}

	c, _, err := websocket.Dial(ctx, viper.GetString("url"), options)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(log.Fields{
			"url": viper.GetString("url"),
		}).Error("Failed to connect to overmind API")
		return 1
	}
	defer c.Close(websocket.StatusGoingAway, "")

	// Log the request in JSON
	b, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		log.WithContext(ctx).WithError(err).Error("Failed to marshal request")
		return 1
	}

	log.WithContext(ctx).Infof("Request:\n%v", string(b))

	err = wspb.Write(ctx, c, req)
	if err != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"error": err,
		}).Error("Failed to send request")
		return 1
	}

	responses := make(chan *sdp.GatewayResponse)

	// Start a goroutine that reads responses
	go func() {
		for {
			res := new(sdp.GatewayResponse)

			err = wspb.Read(ctx, c, res)

			if err != nil {
				var e websocket.CloseError
				if errors.As(err, &e) {
					log.WithContext(ctx).WithFields(log.Fields{
						"code":   e.Code.String(),
						"reason": e.Reason,
					}).Info("Websocket closing")
					return
				}
				log.WithContext(ctx).WithFields(log.Fields{
					"error": err,
				}).Error("Failed to read response")
				return
			}

			responses <- res
		}
	}()

	activeQueries := make(map[uuid.UUID]bool)

	// Read the responses
responses:
	for {
		select {
		case <-signals:
			log.WithContext(ctx).Info("Received interrupt, exiting")
			return 1
		case <-ctx.Done():
			log.WithContext(ctx).Info("Context cancelled, exiting")
			return 1
		case resp := <-responses:
			switch resp.ResponseType.(type) {
			case *sdp.GatewayResponse_QueryStatus:
				status := resp.GetQueryStatus()
				queryUuid := status.GetUUIDParsed()
				if queryUuid == nil {
					log.WithContext(ctx).Debugf("Received QueryStatus with nil UUID: %v", status.Status.String())
					continue responses
				}

				log.WithContext(ctx).Debugf("Status for %v: %v", queryUuid, status.Status.String())

				switch status.Status {
				case sdp.QueryStatus_STARTED:
					activeQueries[*queryUuid] = true
					continue responses
				case sdp.QueryStatus_FINISHED:
					activeQueries[*queryUuid] = false
				case sdp.QueryStatus_ERRORED:
					activeQueries[*queryUuid] = false
				case sdp.QueryStatus_CANCELLED:
					activeQueries[*queryUuid] = false
				default:
					log.WithContext(ctx).Debugf("unexpected status %v: %v", queryUuid, status.Status.String())
					continue responses
				}

				// fall through from all "final" query states, check if there's still queries in progress
				// TODO: needs DefaultStartTimeout implemented to account for slow sources
				allDone := true
			active:
				for q := range activeQueries {
					if activeQueries[q] {
						log.WithContext(ctx).Debugf("%v still active", q)
						allDone = false
						break active
					}
				}

				if allDone {
					break responses
				}
			case *sdp.GatewayResponse_NewItem:
				item := resp.GetNewItem()

				log.WithContext(ctx).Infof("New item: %v", item.GloballyUniqueName())
			case *sdp.GatewayResponse_NewEdge:
				edge := resp.GetNewEdge()

				log.WithContext(ctx).Infof("New edge: %v->%v", edge.From.GloballyUniqueName(), edge.To.GloballyUniqueName())
			case *sdp.GatewayResponse_QueryError:
				err := resp.GetQueryError()

				log.WithContext(ctx).Errorf("Error from %v(%v): %v", err.ResponderName, err.SourceName, err)
			case *sdp.GatewayResponse_Error:
				err := resp.GetError()
				log.WithContext(ctx).Errorf("generic error: %v", err)
			default:
				j := protojson.Format(resp)

				log.WithContext(ctx).Infof("Unknown %T Response:\n%v", resp.ResponseType, j)
			}
		}
	}

	if viper.GetBool("snapshot-after") {
		log.WithContext(ctx).Info("Starting snapshot")
		msgId := uuid.New()
		snapReq := &sdp.GatewayRequest{
			RequestType: &sdp.GatewayRequest_StoreSnapshot{
				StoreSnapshot: &sdp.StoreSnapshot{
					Name:        viper.GetString("snapshot-name"),
					Description: viper.GetString("snapshot-description"),
					MsgID:       msgId[:],
				},
			},
		}
		err = wspb.Write(ctx, c, snapReq)
		if err != nil {
			log.WithContext(ctx).WithFields(log.Fields{
				"error": err,
			}).Error("Failed to send snapshot request")
			return 1
		}

		for {
			select {
			case <-signals:
				log.WithContext(ctx).Info("Received interrupt, exiting")
				return 1
			case <-ctx.Done():
				log.WithContext(ctx).Info("Context cancelled, exiting")
				return 1
			case resp := <-responses:
				switch resp.ResponseType.(type) {
				case *sdp.GatewayResponse_SnapshotStoreResult:
					result := resp.GetSnapshotStoreResult()
					if result.Success {
						log.WithContext(ctx).Infof("Snapshot stored successfully: %v", uuid.UUID(result.SnapshotID))
						return 0
					}

					log.WithContext(ctx).Errorf("Snapshot store failed: %v", result.ErrorMessage)
					return 1
				default:
					j := protojson.Format(resp)

					log.WithContext(ctx).Infof("Unknown %T Response:\n%v", resp.ResponseType, j)
				}
			}
		}
	}
	return 0
}

func createInitialRequest() (*sdp.GatewayRequest, error) {
	req := new(sdp.GatewayRequest)
	u := uuid.New()

	switch viper.GetString("request-type") {
	case "query":
		var method sdp.QueryMethod

		switch viper.GetString("query-method") {
		case "get":
			method = sdp.QueryMethod_GET
		case "list":
			method = sdp.QueryMethod_LIST
		case "search":
			method = sdp.QueryMethod_SEARCH
		default:
			return nil, fmt.Errorf("query method %v not supported", viper.GetString("query-method"))
		}

		req.RequestType = &sdp.GatewayRequest_Query{
			Query: &sdp.Query{
				Method:  method,
				Type:    viper.GetString("query-type"),
				Query:   viper.GetString("query"),
				Scope:   viper.GetString("query-scope"),
				Timeout: durationpb.New(10 * time.Hour),
				UUID:    u[:],
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth:                  viper.GetUint32("link-depth"),
					FollowOnlyBlastPropagation: viper.GetBool("blast-radius"),
				},
			},
		}
	case "load-bookmark":
		bookmarkUUID, err := uuid.Parse(viper.GetString("bookmark-uuid"))
		if err != nil {
			return nil, err
		}
		msgID := uuid.New()
		req.RequestType = &sdp.GatewayRequest_LoadBookmark{
			LoadBookmark: &sdp.LoadBookmark{
				UUID:  bookmarkUUID[:],
				MsgID: msgID[:],
			},
		}
	case "load-snapshot":
		snapshotUUID, err := uuid.Parse(viper.GetString("snapshot-uuid"))
		if err != nil {
			return nil, err
		}
		msgID := uuid.New()
		req.RequestType = &sdp.GatewayRequest_LoadSnapshot{
			LoadSnapshot: &sdp.LoadSnapshot{
				UUID:  snapshotUUID[:],
				MsgID: msgID[:],
			},
		}
	default:
		return nil, fmt.Errorf("request type %v not supported", viper.GetString("request-type"))
	}

	return req, nil
}

func init() {
	rootCmd.AddCommand(requestCmd)

	requestCmd.PersistentFlags().String("request-type", "query", "The type of request to send (query, load-bookmark, load-snapshot)")

	requestCmd.PersistentFlags().String("query-method", "get", "The method to use (get, list, search)")
	requestCmd.PersistentFlags().String("query-type", "*", "The type to query")
	requestCmd.PersistentFlags().String("query", "", "The actual query to send")
	requestCmd.PersistentFlags().String("query-scope", "*", "The scope to query")

	requestCmd.PersistentFlags().String("bookmark-uuid", "", "The UUID of the bookmark to load")
	requestCmd.PersistentFlags().String("snapshot-uuid", "", "The UUID of the snapshot to load")

	requestCmd.PersistentFlags().Bool("snapshot-after", false, "Set this to create a snapshot of the query results")
	requestCmd.PersistentFlags().String("snapshot-name", "CLI", "The snapshot name of the query results")
	requestCmd.PersistentFlags().String("snapshot-description", "none", "The snapshot description of the query results")

	requestCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
	requestCmd.PersistentFlags().Uint32("link-depth", 0, "How deeply to link")
	requestCmd.PersistentFlags().Bool("blast-radius", false, "Whether to query using blast radius, note that if using this option, link-depth should be set to > 0")

	// Bind these to viper
	err := viper.BindPFlags(requestCmd.PersistentFlags())
	if err != nil {
		log.WithError(err).Fatal("could not bind `request` flags")
	}
}
