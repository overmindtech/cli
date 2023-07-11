package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/overmindtech/ovm-cli/cmd/datamaps"
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

// changeFromTfplanCmd represents the change-from-tfplan command
var changeFromTfplanCmd = &cobra.Command{
	Use:   "change-from-tfplan [--title TITLE] [--description DESCRIPTION] [--ticket-link URL] [--tfplan FILE]",
	Short: "Creates a new Change from a given terraform plan file",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind these to viper
		err := viper.BindPFlags(cmd.PersistentFlags())
		if err != nil {
			log.WithError(err).Fatal("could not bind `change-from-tfplan` flags")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		exitcode := ChangeFromTfplan(sigs, nil)
		tracing.ShutdownTracer()
		os.Exit(exitcode)
	},
}

// test data
var (
	affecting_uuid     uuid.UUID  = uuid.New()
	affecting_resource *sdp.Query = &sdp.Query{
		Type:   "elbv2-load-balancer",
		Method: sdp.QueryMethod_GET,
		Query:  "ingress",
		RecursionBehaviour: &sdp.Query_RecursionBehaviour{
			LinkDepth: 0,
		},
		Scope: "944651592624.eu-west-2",
		UUID:  affecting_uuid[:],
	}

	safe_uuid     uuid.UUID  = uuid.New()
	safe_resource *sdp.Query = &sdp.Query{
		Type:   "ec2-security-group",
		Method: sdp.QueryMethod_GET,
		Query:  "sg-09533c300cd1a41c1",
		RecursionBehaviour: &sdp.Query_RecursionBehaviour{
			LinkDepth: 0,
		},
		Scope: "944651592624.eu-west-2",
		UUID:  safe_uuid[:],
	}
)

type TfData struct {
	Address string
	Type    string
	Values  map[string]any
}

func changingItemQueriesFromTfplan(ctx context.Context, lf log.Fields) ([]*sdp.Query, error) {
	// read results from `terraform show -json ${tfplan file}`
	contents, err := os.ReadFile(viper.GetString("tfplan-json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read %v: %w", viper.GetString("tfplan-json"), err)
	}

	changing_items_tf := map[string]TfData{}

	var parsed map[string]any
	json.Unmarshal(contents, &parsed)
	root_module := parsed["planned_values"].(map[string]any)["root_module"].(map[string]any)
	resourceValues := map[string]map[string]any{}
	resourceValuesFromModule(root_module, &resourceValues)

	resource_changes := parsed["resource_changes"].([]any)
	for _, changed_item_data := range resource_changes {
		changed_item_map := changed_item_data.(map[string]any)
		change := changed_item_map["change"].(map[string]any)
		actions := change["actions"].([]any)
		if len(actions) == 0 || actions[0] == "no-op" {
			// skip resources with no changes
			continue
		}
		log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
			"actions": actions,
			"address": changed_item_map["address"],
		}).Debugf("mapping item")

		changed_item := TfData{
			Address: changed_item_map["address"].(string),
			Type:    changed_item_map["type"].(string),
			Values:  resourceValues[changed_item_map["address"].(string)],
		}

		changing_items_tf[changed_item.Address] = changed_item
	}

	var changing_items []*sdp.Query
	// for all managed resources:
	for _, r := range changing_items_tf {
		mappings, ok := datamaps.AwssourceData[r.Type]
		if !ok {
			log.WithContext(ctx).WithFields(lf).WithField("terraform-address", r.Address).Warn("skipping unmapped resource")
			break
		}

		for _, mapData := range mappings {
			queryStr, ok := r.Values[mapData.QueryField]
			if !ok {
				log.WithContext(ctx).
					WithFields(lf).
					WithField("terraform-address", r.Address).
					WithField("terraform-query-field", mapData.QueryField).Warn("skipping resource without query field")
				break
			}

			u := uuid.New()
			changing_items = append(changing_items, &sdp.Query{
				Type:               mapData.Type,
				Method:             mapData.Method,
				Query:              queryStr.(string),
				Scope:              mapData.Scope,
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
				UUID:               u[:],
			})
		}
	}

	return changing_items, nil
}

func resourceValuesFromModule(module map[string]any, result *map[string]map[string]any) {
	resource_data, ok := module["resources"]
	if ok {
		for _, r := range resource_data.([]any) {
			resource := r.(map[string]any)
			(*result)[resource["address"].(string)] = resource["values"].(map[string]any)
		}
	}

	child_modules_data, ok := module["child_modules"]
	if ok {
		for _, cm := range child_modules_data.([]any) {
			child_module := cm.(map[string]any)
			resourceValuesFromModule(child_module, result)
		}
	}
}

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

	lf := log.Fields{
		"url": viper.GetString("url"),
	}

	ctx, err = ensureToken(ctx, signals)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to authenticate")
		return 1
	}

	// apply a timeout to the main body of processing
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.WithContext(ctx).WithFields(lf).Info("resolving items from terraform plan")
	queries, err := changingItemQueriesFromTfplan(ctx, lf)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to read terraform plan")
		return 1
	}

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
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("failed to create change")
		return 1
	}

	lf["change"] = createResponse.Msg.Change.Metadata.GetUUIDParsed()
	log.WithContext(ctx).WithFields(lf).Info("created a new change")

	receivedItems := []*sdp.Reference{}

	if len(queries) > 0 {
		options := &websocket.DialOptions{
			HTTPClient: NewAuthenticatedClient(ctx, otelhttp.DefaultClient),
		}

		log.WithContext(ctx).WithFields(lf).WithField("item_count", len(queries)).Info("identifying items")
		c, _, err := websocket.Dial(ctx, viper.GetString("url"), options)
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
	} else {
		log.WithContext(ctx).WithFields(lf).Info("no item queries mapped, skipping changing items")
	}

	if len(receivedItems) > 0 {
		log.WithContext(ctx).WithFields(lf).WithField("received_items", len(receivedItems)).Info("updating changing items on the change record")
	} else {
		log.WithContext(ctx).WithFields(lf).WithField("received_items", len(receivedItems)).Info("updating change record with no items")
	}
	resultStream, err := client.UpdateChangingItems(ctx, &connect.Request[sdp.UpdateChangingItemsRequest]{
		Msg: &sdp.UpdateChangingItemsRequest{
			ChangeUUID:    createResponse.Msg.Change.Metadata.UUID,
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
		if resultStream.Err() != nil {
			log.WithContext(ctx).WithFields(lf).WithError(err).Error("error streaming results")
			return 1
		}

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

	changeUrl := fmt.Sprintf("%v/changes/%v", viper.GetString("frontend"), createResponse.Msg.Change.Metadata.GetUUIDParsed())
	log.WithContext(ctx).WithFields(lf).WithField("change-url", changeUrl).Info("change ready")

	fetchResponse, err := client.GetChange(ctx, &connect.Request[sdp.GetChangeRequest]{
		Msg: &sdp.GetChangeRequest{
			UUID: createResponse.Msg.Change.Metadata.UUID,
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

func allDone(ctx context.Context, activeQueries map[uuid.UUID]bool, lf log.Fields) bool {
	allDone := true
	for q := range activeQueries {
		if activeQueries[q] {
			log.WithContext(ctx).WithFields(lf).WithField("query", q).Debugf("query still active")
			allDone = false
			break
		}
	}
	return allDone
}

func init() {
	rootCmd.AddCommand(changeFromTfplanCmd)

	changeFromTfplanCmd.PersistentFlags().String("changes-url", "https://api.prod.overmind.tech", "The changes service API endpoint")
	changeFromTfplanCmd.PersistentFlags().String("frontend", "https://app.overmind.tech", "The frontend base URL")

	changeFromTfplanCmd.PersistentFlags().String("tfplan-json", "./tfplan.json", "Parse changing items from this terraform plan JSON file. Generate this using `terraform show -json PLAN_FILE`")

	changeFromTfplanCmd.PersistentFlags().String("title", "", "Short title for this change.")
	changeFromTfplanCmd.PersistentFlags().String("description", "", "Quick description of the change.")
	changeFromTfplanCmd.PersistentFlags().String("ticket-link", "*", "Link to the ticket for this change.")
	changeFromTfplanCmd.PersistentFlags().String("owner", "", "The owner of this change.")
	// changeFromTfplanCmd.PersistentFlags().String("cc-emails", "", "A comma-separated list of emails to keep updated with the status of this change.")

	changeFromTfplanCmd.PersistentFlags().String("timeout", "1m", "How long to wait for responses")
}
