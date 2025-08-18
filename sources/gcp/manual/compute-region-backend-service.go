package manual

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeRegionBackendServiceLookupByRegion = shared.NewItemTypeLookup("region", gcpshared.ComputeRegionBackendService)
	ComputeRegionBackendServiceLookupByName   = shared.NewItemTypeLookup("name", gcpshared.ComputeRegionBackendService)
)

type computeRegionBackendServiceWrapper struct {
	client gcpshared.ComputeRegionBackendServiceClient

	*gcpshared.ProjectBase
}

// NewComputeRegionBackendService creates a new computeRegionBackendServiceWrapper instance
func NewComputeRegionBackendService(client gcpshared.ComputeRegionBackendServiceClient, projectID string) sources.SearchableWrapper {
	return &computeRegionBackendServiceWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeRegionBackendService,
		),
	}
}

func (c computeRegionBackendServiceWrapper) IAMPermissions() []string {
	return []string{
		"compute.regionBackendServices.get",
		"compute.regionBackendServices.list",
	}
}

func (computeRegionBackendServiceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeNetwork,
		gcpshared.ComputeSecurityPolicy,
		gcpshared.NetworkSecurityClientTlsPolicy,
		gcpshared.NetworkServicesServiceLbPolicy,
		gcpshared.NetworkServicesServiceBinding,
		gcpshared.ComputeInstanceGroup,
	)
}

// TerraformMappings returns the Terraform mappings for the compute region backend service wrapper
func (c computeRegionBackendServiceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_region_backend_service
			TerraformQueryMap: "google_compute_region_backend_service.name",
		},
	}
}

// GetLookups returns the lookups for the compute region backend service wrapper
func (c computeRegionBackendServiceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeRegionBackendServiceLookupByRegion,
		ComputeRegionBackendServiceLookupByName,
	}
}

// Get retrieves a compute region backend service by its region and name
func (c computeRegionBackendServiceWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	region := queryParts[0]
	name := queryParts[1]

	req := &computepb.GetRegionBackendServiceRequest{
		Project:        c.ProjectID(),
		Region:         region,
		BackendService: name,
	}

	service, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), service)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (c computeRegionBackendServiceWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeRegionBackendServiceLookupByName,
		},
	}
}

// Search searches for compute region backend services based withing the given region.
func (c computeRegionBackendServiceWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListRegionBackendServicesRequest{
		Project: c.ProjectID(),
		Region:  queryParts[0],
	})

	var items []*sdp.Item
	for {
		bs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), bs)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// SearchStream streams the search results for compute region backend services.
func (c computeRegionBackendServiceWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey, queryParts ...string) {
	it := c.client.List(ctx, &computepb.ListRegionBackendServicesRequest{
		Project: c.ProjectID(),
		Region:  queryParts[0],
	})

	for {
		backendService, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), backendService)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}
