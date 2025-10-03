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

var ComputeRegionBackendServiceLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeRegionBackendService)

type computeRegionBackendServiceWrapper struct {
	client gcpshared.ComputeRegionBackendServiceClient

	*gcpshared.RegionBase
}

// NewComputeRegionBackendService creates a new computeRegionBackendServiceWrapper instance
func NewComputeRegionBackendService(client gcpshared.ComputeRegionBackendServiceClient, projectID string, region string) sources.ListableWrapper {
	return &computeRegionBackendServiceWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			projectID,
			region,
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

func (c computeRegionBackendServiceWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
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
		ComputeRegionBackendServiceLookupByName,
	}
}

// Get retrieves a compute region backend service by its region and name
func (c computeRegionBackendServiceWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	name := queryParts[0]

	req := &computepb.GetRegionBackendServiceRequest{
		Project:        c.ProjectID(),
		Region:         c.Region(),
		BackendService: name,
	}

	service, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), c.DefaultScope(), service)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists all compute region backend services in the specified region
func (c computeRegionBackendServiceWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListRegionBackendServicesRequest{
		Project: c.ProjectID(),
		Region:  c.Region(),
	})

	var items []*sdp.Item
	for {
		bs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.Region(), c.Type())
		}

		item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), c.Region(), bs)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream lists all compute region backend services in the specified region and streams them to the provided stream
func (c computeRegionBackendServiceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListRegionBackendServicesRequest{
		Project: c.ProjectID(),
		Region:  c.Region(),
	})

	for {
		backendService, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.Region(), c.Type()))
			return
		}

		item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), c.Region(), backendService)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}
