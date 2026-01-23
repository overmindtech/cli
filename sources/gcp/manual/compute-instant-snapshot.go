package manual

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/proto"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeInstantSnapshotLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeInstantSnapshot)

type computeInstantSnapshotWrapper struct {
	client gcpshared.ComputeInstantSnapshotsClient
	*gcpshared.ZoneBase
}

// NewComputeInstantSnapshot creates a new computeInstantSnapshotWrapper instance.
func NewComputeInstantSnapshot(client gcpshared.ComputeInstantSnapshotsClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeInstantSnapshotWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			gcpshared.ComputeInstantSnapshot,
		),
	}
}

func (c computeInstantSnapshotWrapper) IAMPermissions() []string {
	return []string{
		"compute.instantSnapshots.get",
		"compute.instantSnapshots.list",
	}
}

func (c computeInstantSnapshotWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeInstantSnapshotWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeDisk,
	)
}

func (c computeInstantSnapshotWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_instant_snapshot.name",
		},
	}
}

func (c computeInstantSnapshotWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstantSnapshotLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute instant snapshots since they use aggregatedList
func (c computeInstantSnapshotWrapper) SupportsWildcardScope() bool {
	return true
}

func (c computeInstantSnapshotWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetInstantSnapshotRequest{
		Project:         location.ProjectID,
		Zone:            location.Zone,
		InstantSnapshot: queryParts[0],
	}

	instantSnapshot, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeInstantSnapshotToSDPItem(ctx, instantSnapshot, location)
}

func (c computeInstantSnapshotWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeInstantSnapshotWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	// Handle wildcard scope with AggregatedList
	if scope == "*" {
		c.listAggregatedStream(ctx, stream, cache, cacheKey)
		return
	}

	// Handle specific scope with per-zone List
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListInstantSnapshotsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		instantSnapshot, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeInstantSnapshotToSDPItem(ctx, instantSnapshot, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// listAggregatedStream uses AggregatedList to stream all instant snapshots across all zones
func (c computeInstantSnapshotWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := gcpshared.GetProjectIDsFromLocations(c.Locations())

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListInstantSnapshotsRequest{
				Project:              projectID,
				ReturnPartialSuccess: proto.Bool(true), // Handle partial failures gracefully
			})

			for {
				pair, iterErr := it.Next()
				if errors.Is(iterErr, iterator.Done) {
					break
				}
				if iterErr != nil {
					stream.SendError(gcpshared.QueryError(iterErr, projectID, c.Type()))
					return iterErr
				}

				// Parse scope from pair.Key (e.g., "zones/us-central1-a")
				scopeLocation, err := gcpshared.ParseAggregatedListScope(projectID, pair.Key)
				if err != nil {
					continue // Skip unparseable scopes
				}

				// Only process if this scope is in our adapter's configured locations
				if !gcpshared.HasLocationInSlices(scopeLocation, c.Locations()) {
					continue
				}

				// Process instant snapshots in this scope
				if pair.Value != nil && pair.Value.GetInstantSnapshots() != nil {
					for _, instantSnapshot := range pair.Value.GetInstantSnapshots() {
						item, sdpErr := c.gcpComputeInstantSnapshotToSDPItem(ctx, instantSnapshot, scopeLocation)
						if sdpErr != nil {
							stream.SendError(sdpErr)
							continue
						}

						cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
						stream.SendItem(item)
					}
				}
			}

			return nil
		})
	}

	// Wait for all goroutines to complete
	_ = p.Wait()
}

func (c computeInstantSnapshotWrapper) gcpComputeInstantSnapshotToSDPItem(ctx context.Context, instantSnapshot *computepb.InstantSnapshot, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(instantSnapshot, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeInstantSnapshot.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            instantSnapshot.GetLabels(),
	}

	// Link source disk
	if disk := instantSnapshot.GetSourceDisk(); disk != "" {
		diskName := gcpshared.LastPathComponent(disk)
		if diskName != "" {
			scope, err := gcpshared.ExtractScopeFromURI(ctx, disk)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeDisk.String(),
						Method: sdp.QueryMethod_GET,
						Query:  diskName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				})
			}
		}
	}

	switch instantSnapshot.GetStatus() {
	case computepb.InstantSnapshot_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.InstantSnapshot_CREATING.String(),
		computepb.InstantSnapshot_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.InstantSnapshot_FAILED.String(),
		computepb.InstantSnapshot_UNAVAILABLE.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.InstantSnapshot_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	default:
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	}

	return sdpItem, nil
}
