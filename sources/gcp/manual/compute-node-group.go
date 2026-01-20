package manual

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeNodeGroupLookupByName             = shared.NewItemTypeLookup("name", gcpshared.ComputeNodeGroup)
	ComputeNodeGroupLookupByNodeTemplateName = shared.NewItemTypeLookup("nodeTemplateName", gcpshared.ComputeNodeGroup)
)

type computeNodeGroupWrapper struct {
	client gcpshared.ComputeNodeGroupClient
	*gcpshared.ZoneBase
}

// NewComputeNodeGroup creates a new computeNodeGroupWrapper instance.
func NewComputeNodeGroup(client gcpshared.ComputeNodeGroupClient, locations []gcpshared.LocationInfo) sources.SearchableListableWrapper {
	return &computeNodeGroupWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeNodeGroup,
		),
	}
}

func (c computeNodeGroupWrapper) IAMPermissions() []string {
	return []string{
		"compute.nodeGroups.get",
		"compute.nodeGroups.list",
	}
}

func (c computeNodeGroupWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeNodeGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeNodeTemplate,
	)
}

func (c computeNodeGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_node_group.name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "google_compute_node_template.name",
		},
	}
}

func (c computeNodeGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeNodeGroupLookupByName,
	}
}

func (c computeNodeGroupWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeNodeGroupLookupByNodeTemplateName,
		},
	}
}

func (c computeNodeGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetNodeGroupRequest{
		Project:   location.ProjectID,
		Zone:      location.Zone,
		NodeGroup: queryParts[0],
	}

	nodeGroup, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeNodeGroupToSDPItem(ctx, nodeGroup, location)
}

func (c computeNodeGroupWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListNodeGroupsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	var items []*sdp.Item
	for {
		nodegroup, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeNodeGroupToSDPItem(ctx, nodegroup, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeNodeGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListNodeGroupsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		nodeGroup, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeNodeGroupToSDPItem(ctx, nodeGroup, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeNodeGroupWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	nodeTemplate := queryParts[0]

	req := &computepb.ListNodeGroupsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
		Filter:  ptr.To("nodeTemplate = " + nodeTemplate),
	}

	it := c.client.List(ctx, req)

	var items []*sdp.Item
	for {
		nodegroup, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeNodeGroupToSDPItem(ctx, nodegroup, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeNodeGroupWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	nodeTemplate := queryParts[0]

	req := &computepb.ListNodeGroupsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
		Filter:  ptr.To("nodeTemplate = " + nodeTemplate),
	}

	it := c.client.List(ctx, req)

	for {
		nodeGroup, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeNodeGroupToSDPItem(ctx, nodeGroup, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeNodeGroupWrapper) gcpComputeNodeGroupToSDPItem(ctx context.Context, nodegroup *computepb.NodeGroup, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(nodegroup)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeNodeGroup.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		// No labels for node groups.
	}

	templateUrl := nodegroup.GetNodeTemplate()
	if templateUrl != "" {
		name := gcpshared.LastPathComponent(templateUrl)
		if name != "" {
			scope, err := gcpshared.ExtractScopeFromURI(ctx, templateUrl)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeNodeTemplate.String(),
						Method: sdp.QueryMethod_GET,
						Query:  name,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	switch nodegroup.GetStatus() {
	case computepb.NodeGroup_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	case computepb.NodeGroup_INVALID.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.NodeGroup_CREATING.String(),
		computepb.NodeGroup_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	}

	return sdpItem, nil
}
