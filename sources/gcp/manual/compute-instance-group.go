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

var ComputeInstanceGroupLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeInstanceGroup)

type computeInstanceGroupWrapper struct {
	client gcpshared.ComputeInstanceGroupsClient
	*gcpshared.ZoneBase
}

// NewComputeInstanceGroup creates a new computeInstanceGroupWrapper instance.
func NewComputeInstanceGroup(client gcpshared.ComputeInstanceGroupsClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeInstanceGroupWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeInstanceGroup,
		),
	}
}

func (c computeInstanceGroupWrapper) IAMPermissions() []string {
	return []string{
		"compute.instanceGroups.get",
		"compute.instanceGroups.list",
	}
}

func (c computeInstanceGroupWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeInstanceGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeSubnetwork,
		gcpshared.ComputeNetwork,
		gcpshared.ComputeZone,
		gcpshared.ComputeRegion,
	)
}

func (c computeInstanceGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_instance_group.name",
		},
	}
}

func (c computeInstanceGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstanceGroupLookupByName,
	}
}

func (c computeInstanceGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetInstanceGroupRequest{
		Project:       location.ProjectID,
		Zone:          location.Zone,
		InstanceGroup: queryParts[0],
	}

	instanceGroup, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeInstanceGroupToSDPItem(instanceGroup, location)
}

func (c computeInstanceGroupWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListInstanceGroupsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	var items []*sdp.Item
	for {
		instanceGroup, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeInstanceGroupToSDPItem(instanceGroup, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeInstanceGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListInstanceGroupsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		instanceGroup, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeInstanceGroupToSDPItem(instanceGroup, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeInstanceGroupWrapper) gcpComputeInstanceGroupToSDPItem(instanceGroup *computepb.InstanceGroup, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(instanceGroup, "")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	item := &sdp.Item{
		Type:            gcpshared.ComputeInstanceGroup.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
	}

	if network := instanceGroup.GetNetwork(); network != "" {
		networkName := gcpshared.LastPathComponent(network)
		if networkName != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  networkName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
	}

	if subnetwork := instanceGroup.GetSubnetwork(); subnetwork != "" {
		subnetworkName := gcpshared.LastPathComponent(subnetwork)
		if subnetworkName != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeSubnetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  subnetworkName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
	}

	if zone := instanceGroup.GetZone(); zone != "" {
		zoneName := gcpshared.LastPathComponent(zone)
		if zoneName != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeZone.String(),
					Method: sdp.QueryMethod_GET,
					Query:  zoneName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	if region := instanceGroup.GetRegion(); region != "" {
		regionName := gcpshared.LastPathComponent(region)
		if regionName != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeRegion.String(),
					Method: sdp.QueryMethod_GET,
					Query:  regionName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	return item, nil
}
