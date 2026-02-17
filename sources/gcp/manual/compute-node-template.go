package manual

import (
	"context"
	"errors"
	"sync/atomic"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/proto"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
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
func NewComputeNodeTemplate(client gcpshared.ComputeNodeTemplateClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeNodeTemplateWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			locations,
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

func (c computeNodeTemplateWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeNodeTemplateWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeNodeGroup,
	)
}

func (c computeNodeTemplateWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_node_template.name",
		},
	}
}

func (c computeNodeTemplateWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeNodeTemplateLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute node templates since they use aggregatedList
func (c computeNodeTemplateWrapper) SupportsWildcardScope() bool {
	return true
}

func (c computeNodeTemplateWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetNodeTemplateRequest{
		Project:      location.ProjectID,
		Region:       location.Region,
		NodeTemplate: queryParts[0],
	}

	nodeTemplate, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeNodeTemplateToSDPItem(nodeTemplate, location)
}

func (c computeNodeTemplateWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeNodeTemplateWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	// Handle wildcard scope with AggregatedList
	if scope == "*" {
		c.listAggregatedStream(ctx, stream, cache, cacheKey)
		return
	}

	// Handle specific scope with per-region List
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListNodeTemplatesRequest{
		Project: location.ProjectID,
		Region:  location.Region,
	})

	var itemsSent int
	var hadError bool
	for {
		nodeTemplate, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeNodeTemplateToSDPItem(nodeTemplate, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			hadError = true
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
		itemsSent++
	}
	if itemsSent == 0 && !hadError {
		notFoundErr := &sdp.QueryError{
			ErrorType:     sdp.QueryError_NOTFOUND,
			ErrorString:   "no compute node templates found in scope " + scope,
			Scope:         scope,
			SourceName:    c.Name(),
			ItemType:      c.Type(),
			ResponderName: c.Name(),
		}
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, cacheKey)
	}
}

// listAggregatedStream uses AggregatedList to stream all node templates across all regions
func (c computeNodeTemplateWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := gcpshared.GetProjectIDsFromLocations(c.Locations())

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)
	var itemsSent atomic.Int32
	var hadError atomic.Bool

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListNodeTemplatesRequest{
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
					hadError.Store(true)
					return iterErr
				}

				// Parse scope from pair.Key (e.g., "regions/us-central1")
				scopeLocation, err := gcpshared.ParseAggregatedListScope(projectID, pair.Key)
				if err != nil {
					continue // Skip unparseable scopes
				}

				// Only process if this scope is in our adapter's configured locations
				if !gcpshared.HasLocationInSlices(scopeLocation, c.Locations()) {
					continue
				}

				// Process node templates in this scope
				if pair.Value != nil && pair.Value.GetNodeTemplates() != nil {
					for _, nodeTemplate := range pair.Value.GetNodeTemplates() {
						item, sdpErr := c.gcpComputeNodeTemplateToSDPItem(nodeTemplate, scopeLocation)
						if sdpErr != nil {
							stream.SendError(sdpErr)
							hadError.Store(true)
							continue
						}

						cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
						stream.SendItem(item)
						itemsSent.Add(1)
					}
				}
			}

			return nil
		})
	}

	// Wait for all goroutines to complete
	_ = p.Wait()
	if itemsSent.Load() == 0 && !hadError.Load() {
		notFoundErr := &sdp.QueryError{
			ErrorType:     sdp.QueryError_NOTFOUND,
			ErrorString:   "no compute node templates found in scope *",
			Scope:         "*",
			SourceName:    c.Name(),
			ItemType:      c.Type(),
			ResponderName: c.Name(),
		}
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, cacheKey)
	}
}

func (c computeNodeTemplateWrapper) gcpComputeNodeTemplateToSDPItem(nodeTemplate *computepb.NodeTemplate, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           location.ToScope(),
	}

	// Backlink to any node group using this template.
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.ComputeNodeGroup.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  nodeTemplate.GetName(),
			Scope:  "*",
		},
	})

	return sdpItem, nil
}
