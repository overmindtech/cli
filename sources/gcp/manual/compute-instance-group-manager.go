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

var ComputeInstanceGroupManagerLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeInstanceGroupManager)

type computeInstanceGroupManagerWrapper struct {
	client gcpshared.ComputeInstanceGroupManagerClient
	*gcpshared.ZoneBase
}

// NewComputeInstanceGroupManager creates a new computeInstanceGroupManagerWrapper.
func NewComputeInstanceGroupManager(client gcpshared.ComputeInstanceGroupManagerClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeInstanceGroupManagerWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeInstanceGroupManager,
		),
	}
}

func (c computeInstanceGroupManagerWrapper) IAMPermissions() []string {
	return []string{
		"compute.instanceGroupManagers.get",
		"compute.instanceGroupManagers.list",
	}
}

func (c computeInstanceGroupManagerWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

// PotentialLinks returns the potential links for the compute instance group manager wrapper
func (c computeInstanceGroupManagerWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeInstanceTemplate,
		gcpshared.ComputeRegionInstanceTemplate,
		gcpshared.ComputeInstanceGroup,
		gcpshared.ComputeTargetPool,
		gcpshared.ComputeResourcePolicy,
		gcpshared.ComputeAutoscaler,
		gcpshared.ComputeHealthCheck,
		gcpshared.ComputeZone,
		gcpshared.ComputeRegion,
	)
}

// TerraformMappings returns the Terraform mappings for the compute instance group manager wrapper
func (c computeInstanceGroupManagerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance_group_manager#argument-reference
			TerraformQueryMap: "google_compute_instance_group_manager.name",
		},
	}
}

// GetLookups returns the lookups for the compute instance group manager wrapper
func (c computeInstanceGroupManagerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstanceGroupManagerLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute instance group managers since they use aggregatedList
func (c computeInstanceGroupManagerWrapper) SupportsWildcardScope() bool {
	return true
}

// Get retrieves a compute instance group manager by its name
func (c computeInstanceGroupManagerWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetInstanceGroupManagerRequest{
		Project:              location.ProjectID,
		Zone:                 location.Zone,
		InstanceGroupManager: queryParts[0],
	}

	igm, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpInstanceGroupManagerToSDPItem(ctx, igm, location)
}

func (c computeInstanceGroupManagerWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeInstanceGroupManagerWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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

	it := c.client.List(ctx, &computepb.ListInstanceGroupManagersRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		igm, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpInstanceGroupManagerToSDPItem(ctx, igm, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// listAggregatedStream uses AggregatedList to stream all instance group managers across all zones
func (c computeInstanceGroupManagerWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := gcpshared.GetProjectIDsFromLocations(c.Locations())

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListInstanceGroupManagersRequest{
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

				// Process instance group managers in this scope
				if pair.Value != nil && pair.Value.GetInstanceGroupManagers() != nil {
					for _, igm := range pair.Value.GetInstanceGroupManagers() {
						item, sdpErr := c.gcpInstanceGroupManagerToSDPItem(ctx, igm, scopeLocation)
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

func (c computeInstanceGroupManagerWrapper) gcpInstanceGroupManagerToSDPItem(ctx context.Context, instanceGroupManager *computepb.InstanceGroupManager, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	return InstanceGroupManagerToSDPItem(ctx, instanceGroupManager, location, gcpshared.ComputeInstanceGroupManager)
}
