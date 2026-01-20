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

// NewComputeRegionBackendService creates a new computeRegionBackendServiceWrapper instance.
func NewComputeRegionBackendService(client gcpshared.ComputeRegionBackendServiceClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeRegionBackendServiceWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			locations,
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
		gcpshared.ComputeNetworkEndpointGroup,
		gcpshared.ComputeHealthCheck,
		gcpshared.ComputeInstance,
	)
}

func (c computeRegionBackendServiceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_region_backend_service.name",
		},
	}
}

func (c computeRegionBackendServiceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeRegionBackendServiceLookupByName,
	}
}

func (c computeRegionBackendServiceWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetRegionBackendServiceRequest{
		Project:        location.ProjectID,
		Region:         location.Region,
		BackendService: queryParts[0],
	}

	service, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return gcpComputeBackendServiceToSDPItem(ctx, location.ProjectID, location.ToScope(), service)
}

func (c computeRegionBackendServiceWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListRegionBackendServicesRequest{
		Project: location.ProjectID,
		Region:  location.Region,
	})

	var items []*sdp.Item
	for {
		bs, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := gcpComputeBackendServiceToSDPItem(ctx, location.ProjectID, location.ToScope(), bs)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeRegionBackendServiceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListRegionBackendServicesRequest{
		Project: location.ProjectID,
		Region:  location.Region,
	})

	for {
		backendService, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := gcpComputeBackendServiceToSDPItem(ctx, location.ProjectID, location.ToScope(), backendService)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}
