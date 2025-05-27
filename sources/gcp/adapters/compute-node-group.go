package adapters

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeNodeGroup = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.NodeGroup)

	ComputeNodeGroupLookupByName = shared.NewItemTypeLookup("name", ComputeNodeGroup)
)

type computeNodeGroupWrapper struct {
	client gcpshared.ComputeNodeGroupClient
	*gcpshared.ZoneBase
}

// NewComputeNodeGroup creates a new computeNodeGroupWrapper instance
func NewComputeNodeGroup(client gcpshared.ComputeNodeGroupClient, projectID, zone string) sources.ListableWrapper {
	return &computeNodeGroupWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			zone,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			ComputeNodeGroup,
		),
	}
}

// PotentialLinks returns the potential links for the compute instance wrapper
func (c computeNodeGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		ComputeNodeTemplate,
	)
}

// TerraformMappings returns the Terraform mappings for the compute instance wrapper
func (c computeNodeGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_node_group.name",
		},
	}
}

// GetLookups defines how the source can be queried for specific items.
func (c computeNodeGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeNodeGroupLookupByName,
	}
}

// Get retrieves a compute node group by its name
func (c computeNodeGroupWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetNodeGroupRequest{
		Project:   c.ProjectID(),
		Zone:      c.Zone(),
		NodeGroup: queryParts[0],
	}

	nodegroup, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeNodeGroupToSDPItem(nodegroup)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute node groups and converts them to sdp.Items.
func (c computeNodeGroupWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListNodeGroupsRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	var items []*sdp.Item
	for {
		nodegroup, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeNodeGroupToSDPItem(nodegroup)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeNodeGroupWrapper) gcpComputeNodeGroupToSDPItem(nodegroup *computepb.NodeGroup) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(nodegroup)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            ComputeNodeGroup.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),

		// No labels for node groups.
	}

	templateUrl := nodegroup.GetNodeTemplate()
	if templateUrl != "" {
		// https://www.googleapis.com/compute/v1/projects/{project}/regions/{region}/nodeTemplates/{name}

		region := gcpshared.ExtractRegion(templateUrl)
		name := gcpshared.LastPathComponent(templateUrl)

		if region != "" && name != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeNodeTemplate.String(),
					Method: sdp.QueryMethod_GET,
					Query:  name,
					Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Translate nodegroup status to common sdp status.
	switch nodegroup.GetStatus() {
	case computepb.NodeGroup_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	case computepb.NodeGroup_INVALID.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.NodeGroup_CREATING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.NodeGroup_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	}

	return sdpItem, nil
}
