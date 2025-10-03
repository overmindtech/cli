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

// NewComputeInstanceGroup creates a new computeInstanceGroupWrapper instance
func NewComputeInstanceGroup(client gcpshared.ComputeInstanceGroupsClient, projectID, zone string) sources.ListableWrapper {
	return &computeInstanceGroupWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			zone,
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

// PotentialLinks returns the potential links for the compute instance wrapper
func (c computeInstanceGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeSubnetwork,
		gcpshared.ComputeNetwork,
	)
}

// TerraformMappings returns the Terraform mappings for the compute instance group wrapper
func (c computeInstanceGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance#argument-reference
			TerraformQueryMap: "google_compute_instance_group.name",
		},
	}
}

// GetLookups returns the lookups for the compute instance group wrapper
func (c computeInstanceGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstanceGroupLookupByName,
	}
}

// Get retrieves a compute instance group by its name
func (c computeInstanceGroupWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetInstanceGroupRequest{
		Project:       c.ProjectID(),
		Zone:          c.Zone(),
		InstanceGroup: queryParts[0],
	}

	instanceGroup, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	item, sdpErr := c.gcpComputeInstanceGroupToSDPItem(instanceGroup)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute instance groups and converts them to sdp.Items.
func (c computeInstanceGroupWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListInstanceGroupsRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	var items []*sdp.Item
	for {
		instanceGroup, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		item, sdpErr := c.gcpComputeInstanceGroupToSDPItem(instanceGroup)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream lists compute instance groups and sends them as stream items.
func (c computeInstanceGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListInstanceGroupsRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	for {
		instanceGroup, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeInstanceGroupToSDPItem(instanceGroup)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// gcpComputeInstanceGroupToSDPItem converts a GCP InstanceGroup to an SDP Item, linking GCP resource fields.
func (c computeInstanceGroupWrapper) gcpComputeInstanceGroupToSDPItem(instanceGroup *computepb.InstanceGroup) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           c.DefaultScope(),
	}

	if network := instanceGroup.GetNetwork(); network != "" {
		networkName := gcpshared.LastPathComponent(network)
		if networkName != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  networkName,
					Scope:  c.ProjectID(),
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
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
					Scope:  c.ProjectID(),
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		}
	}

	return item, nil
}
