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

var ComputeNodeTemplateLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeNodeTemplate)

type computeNodeTemplateWrapper struct {
	client gcpshared.ComputeNodeTemplateClient

	*gcpshared.RegionBase
}

// NewComputeNodeTemplate creates a new computeNodeTemplateWrapper instance.
func NewComputeNodeTemplate(client gcpshared.ComputeNodeTemplateClient, projectID, region string) sources.ListableWrapper {
	return &computeNodeTemplateWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			projectID,
			region,
			sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
			gcpshared.ComputeNodeTemplate,
		),
	}
}

func (c computeNodeTemplateWrapper) IAMPermissions() []string {
	return []string{
		"compute.nodeTemplates.get",
		"compute.nodeTemplates.list",
	}
}

func (c computeNodeTemplateWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeNodeGroup,
	)
}

func (c computeNodeTemplateWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_node_group#argument-reference
			TerraformQueryMap: "google_compute_node_template.name",
		},
	}
}

func (c computeNodeTemplateWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeNodeTemplateLookupByName,
	}
}

func (c computeNodeTemplateWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetNodeTemplateRequest{
		Project:      c.ProjectID(),
		Region:       c.Region(),
		NodeTemplate: queryParts[0],
	}

	nodeTemplate, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeNodeTemplateToSDPItem(nodeTemplate)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (c computeNodeTemplateWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	results := c.client.List(ctx, &computepb.ListNodeTemplatesRequest{
		Project: c.ProjectID(),
		Region:  c.Region(),
	})

	var items []*sdp.Item
	for {
		nodeTemplate, err := results.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeNodeTemplateToSDPItem(nodeTemplate)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeNodeTemplateWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListNodeTemplatesRequest{
		Project: c.ProjectID(),
		Region:  c.Region(),
	})

	for {
		nodeTemplate, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeNodeTemplateToSDPItem(nodeTemplate)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeNodeTemplateWrapper) gcpComputeNodeTemplateToSDPItem(nodeTemplate *computepb.NodeTemplate) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(nodeTemplate)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeNodeTemplate.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		// No labels
	}

	// Backlink to any node group using this template.
	// TODO: Revisit this link when working on this issue:
	// https://linear.app/overmind/issue/ENG-404/investigate-how-to-create-backlinks-without-the-location-information
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.ComputeNodeGroup.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  nodeTemplate.GetName(),
			Scope:  "*",
		},

		BlastPropagation: &sdp.BlastPropagation{
			In:  false,
			Out: true,
		},
	})

	return sdpItem, nil
}
