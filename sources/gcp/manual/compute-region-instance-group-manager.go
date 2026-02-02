package manual

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeRegionInstanceGroupManagerLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeRegionInstanceGroupManager)

type computeRegionInstanceGroupManagerWrapper struct {
	client gcpshared.RegionInstanceGroupManagerClient
	*gcpshared.RegionBase
}

// NewComputeRegionInstanceGroupManager creates a new computeRegionInstanceGroupManagerWrapper.
func NewComputeRegionInstanceGroupManager(client gcpshared.RegionInstanceGroupManagerClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeRegionInstanceGroupManagerWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeRegionInstanceGroupManager,
		),
	}
}

func (c computeRegionInstanceGroupManagerWrapper) IAMPermissions() []string {
	return []string{
		"compute.regionInstanceGroupManagers.get",
		"compute.regionInstanceGroupManagers.list",
	}
}

func (c computeRegionInstanceGroupManagerWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

// PotentialLinks returns the potential links for the regional compute instance group manager wrapper
func (c computeRegionInstanceGroupManagerWrapper) PotentialLinks() map[shared.ItemType]bool {
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

// TerraformMappings returns the Terraform mappings for the regional compute instance group manager wrapper
func (c computeRegionInstanceGroupManagerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_instance_group_manager#argument-reference
			TerraformQueryMap: "google_compute_region_instance_group_manager.name",
		},
	}
}

// GetLookups returns the lookups for the regional compute instance group manager wrapper
func (c computeRegionInstanceGroupManagerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeRegionInstanceGroupManagerLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Returns true for regional compute instance group managers since they can list across all regions
func (c computeRegionInstanceGroupManagerWrapper) SupportsWildcardScope() bool {
	return true
}

// Get retrieves a regional compute instance group manager by its name
func (c computeRegionInstanceGroupManagerWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetRegionInstanceGroupManagerRequest{
		Project:              location.ProjectID,
		Region:               location.Region,
		InstanceGroupManager: queryParts[0],
	}

	igm, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpRegionInstanceGroupManagerToSDPItem(ctx, igm, location)
}

func (c computeRegionInstanceGroupManagerWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeRegionInstanceGroupManagerWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	// Handle wildcard scope by listing across all configured regions
	if scope == "*" {
		c.listAllRegionsStream(ctx, stream, cache, cacheKey)
		return
	}

	// Handle specific regional scope with per-region List
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListRegionInstanceGroupManagersRequest{
		Project: location.ProjectID,
		Region:  location.Region,
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

		item, sdpErr := c.gcpRegionInstanceGroupManagerToSDPItem(ctx, igm, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeRegionInstanceGroupManagerWrapper) listAllRegionsStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Use a pool to list across all regions in parallel
	p := pool.New().WithContext(ctx).WithMaxGoroutines(10)

	for _, location := range c.Locations() {
		p.Go(func(ctx context.Context) error {
			it := c.client.List(ctx, &computepb.ListRegionInstanceGroupManagersRequest{
				Project: location.ProjectID,
				Region:  location.Region,
			})

			for {
				igm, iterErr := it.Next()
				if errors.Is(iterErr, iterator.Done) {
					break
				}
				if iterErr != nil {
					stream.SendError(gcpshared.QueryError(iterErr, location.ToScope(), c.Type()))
					return iterErr
				}

				item, sdpErr := c.gcpRegionInstanceGroupManagerToSDPItem(ctx, igm, location)
				if sdpErr != nil {
					stream.SendError(sdpErr)
					continue
				}

				cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
				stream.SendItem(item)
			}

			return nil
		})
	}

	// Wait for all goroutines to complete
	_ = p.Wait()
}

func (c computeRegionInstanceGroupManagerWrapper) gcpRegionInstanceGroupManagerToSDPItem(ctx context.Context, instanceGroupManager *computepb.InstanceGroupManager, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	return InstanceGroupManagerToSDPItem(ctx, instanceGroupManager, location, gcpshared.ComputeRegionInstanceGroupManager)
}
