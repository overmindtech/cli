package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpws"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// createSnapshotCmd represents the create snapshot command
var createSnapshotCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a snapshot by running a query and storing the results",
	Long: `Creates a snapshot by executing a query with the specified parameters and then
storing all discovered items and edges as a named snapshot. This is useful for
capturing the state of your infrastructure at a specific point in time.

The command accepts the same query parameters as the 'query' command, plus
snapshot-specific parameters for naming and describing the snapshot.`,
	PreRun: PreRunSetup,
	RunE:   CreateSnapshot,
}

func CreateSnapshot(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	ctx, oi, _, err := login(ctx, cmd, []string{"explore:read", "changes:write", "reverselink:request"}, nil)
	if err != nil {
		return err
	}

	// Validate required snapshot parameters
	name := viper.GetString("name")
	if name == "" {
		return flagError{usage: fmt.Sprintf("snapshot name is required\n\n%v", cmd.UsageString())}
	}

	lf := log.Fields{
		"snapshot-name": name,
	}
	description := viper.GetString("description")
	if description != "" {
		lf["snapshot-description"] = description
	}

	handler := &createSnapshotHandler{
		lf:                           lf,
		LoggingGatewayMessageHandler: sdpws.LoggingGatewayMessageHandler{Level: log.InfoLevel},
		items:                        []*sdp.Item{},
		edges:                        []*sdp.Edge{},
	}

	gatewayUrl := oi.GatewayUrl()
	lf["gateway-url"] = gatewayUrl
	c, err := sdpws.DialBatch(ctx, gatewayUrl,
		NewAuthenticatedClient(ctx, otelhttp.DefaultClient),
		handler,
	)
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Failed to connect to overmind API")
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to connect to overmind API",
		}
	}
	defer c.Close(ctx)

	// Create and validate the query
	q, err := CreateQuery()
	if err != nil {
		return flagError{usage: fmt.Sprintf("invalid query: %v\n\n%v", err, cmd.UsageString())}
	}

	log.WithContext(ctx).WithFields(lf).WithField("uuid", uuid.UUID(q.GetUUID())).Info("Starting query for snapshot creation")

	// Execute the query
	err = c.SendQuery(ctx, q)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to execute query",
		}
	}

	// Log the query details
	b, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Warn("Failed to marshal query for logging")
	} else {
		log.WithContext(ctx).WithFields(lf).WithField("uuid", uuid.UUID(q.GetUUID())).Debugf("Query executed:\n%v", string(b))
	}

	// Wait for the query to complete
	err = c.Wait(ctx, uuid.UUIDs{uuid.UUID(q.GetUUID())})
	if err != nil {
		log.WithContext(ctx).WithFields(lf).WithError(err).Error("Query failed")
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Query execution failed",
		}
	}

	log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
		"itemsCollected": len(handler.items),
		"edgesCollected": len(handler.edges),
	}).Info("Query completed, creating snapshot")

	// Create the snapshot
	snapshotID, err := c.StoreSnapshot(ctx, name, description)
	if err != nil {
		return loggedError{
			err:     err,
			fields:  lf,
			message: "Failed to create snapshot",
		}
	}

	log.WithContext(ctx).WithFields(lf).WithFields(log.Fields{
		"snapshot-id": snapshotID.String(),
		"itemsStored": len(handler.items),
		"edgesStored": len(handler.edges),
	}).Info("Snapshot created successfully")

	fmt.Printf("✅ Snapshot created successfully\n")
	fmt.Printf("   ID: %s\n", snapshotID.String())
	fmt.Printf("   Name: %s\n", name)
	if description != "" {
		fmt.Printf("   Description: %s\n", description)
	}
	fmt.Printf("   Items: %d\n", len(handler.items))
	fmt.Printf("   Edges: %d\n", len(handler.edges))

	return nil
}

// createSnapshotHandler is a simple implementation of GatewayMessageHandler for snapshot creation
type createSnapshotHandler struct {
	lf log.Fields

	items []*sdp.Item
	edges []*sdp.Edge

	sdpws.LoggingGatewayMessageHandler
}

// assert that createSnapshotHandler implements GatewayMessageHandler
var _ sdpws.GatewayMessageHandler = (*createSnapshotHandler)(nil)

func (h *createSnapshotHandler) NewItem(ctx context.Context, item *sdp.Item) {
	h.LoggingGatewayMessageHandler.NewItem(ctx, item)
	h.items = append(h.items, item)
}

func (h *createSnapshotHandler) NewEdge(ctx context.Context, edge *sdp.Edge) {
	h.LoggingGatewayMessageHandler.NewEdge(ctx, edge)
	h.edges = append(h.edges, edge)
}

func init() {
	snapshotsCmd.AddCommand(createSnapshotCmd)

	addAPIFlags(createSnapshotCmd)

	// Query parameters (reused from query command)
	createSnapshotCmd.PersistentFlags().String("query-method", "get", "The method to use (get, list, search)")
	createSnapshotCmd.PersistentFlags().String("query-type", "*", "The type to query")
	createSnapshotCmd.PersistentFlags().String("query", "", "The actual query to send")
	createSnapshotCmd.PersistentFlags().String("query-scope", "*", "The scope to query")
	createSnapshotCmd.PersistentFlags().Bool("ignore-cache", false, "Set to true to ignore all caches in overmind")
	createSnapshotCmd.PersistentFlags().Uint32("link-depth", 0, "How deeply to link")
	createSnapshotCmd.PersistentFlags().Bool("blast-radius", false, "Whether to query using blast radius, note that if using this option, link-depth should be set to > 0")

	// Snapshot-specific parameters
	createSnapshotCmd.PersistentFlags().String("name", "", "The name for the snapshot (required)")
	createSnapshotCmd.PersistentFlags().String("description", "", "The description for the snapshot")

	// Mark name as required
	_ = createSnapshotCmd.MarkPersistentFlagRequired("name")
}
