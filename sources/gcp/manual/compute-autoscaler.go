package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/proto"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeAutoscalerLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeAutoscaler)

type computeAutoscalerWrapper struct {
	client gcpshared.ComputeAutoscalerClient
	*gcpshared.ZoneBase
}

// NewComputeAutoscaler creates a new computeAutoscalerWrapper instance.
func NewComputeAutoscaler(client gcpshared.ComputeAutoscalerClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeAutoscalerWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
			gcpshared.ComputeAutoscaler,
		),
	}
}

func (c computeAutoscalerWrapper) IAMPermissions() []string {
	return []string{
		"compute.autoscalers.get",
		"compute.autoscalers.list",
	}
}

func (c computeAutoscalerWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeAutoscalerWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeInstanceGroupManager,
	)
}

func (c computeAutoscalerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_address#argument-reference
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_autoscaler.name",
		},
	}
}

func (c computeAutoscalerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeAutoscalerLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute autoscalers since they use aggregatedList
func (c computeAutoscalerWrapper) SupportsWildcardScope() bool {
	return true
}

// Get retrieves an autoscaler by its name for a specific scope.
func (c computeAutoscalerWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetAutoscalerRequest{
		Project:    location.ProjectID,
		Zone:       location.Zone,
		Autoscaler: queryParts[0],
	}

	autoscaler, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeAutoscalerToSDPItem(ctx, autoscaler, location)
}

// List lists autoscalers for a specific scope.
func (c computeAutoscalerWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

// ListStream lists autoscalers for a specific scope and sends them to a stream.
func (c computeAutoscalerWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	// Handle wildcard scope with AggregatedList
	if scope == "*" {
		c.listAggregatedStream(ctx, stream, cache, cacheKey)
		return
	}

	// Handle specific scope with per-zone List
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	results := c.client.List(ctx, &computepb.ListAutoscalersRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		autoscaler, iterErr := results.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeAutoscalerToSDPItem(ctx, autoscaler, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// listAggregatedStream uses AggregatedList to stream all autoscalers across all zones
func (c computeAutoscalerWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := c.GetProjectIDs()

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListAutoscalersRequest{
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
					return iterErr
				}

				// Parse scope from pair.Key (e.g., "zones/us-central1-a")
				scopeLocation, err := gcpshared.ParseAggregatedListScope(projectID, pair.Key)
				if err != nil {
					continue // Skip unparseable scopes
				}

				// Only process if this scope is in our adapter's configured locations
				if !c.HasLocation(scopeLocation) {
					continue
				}

				// Process autoscalers in this scope
				if pair.Value != nil && pair.Value.GetAutoscalers() != nil {
					for _, autoscaler := range pair.Value.GetAutoscalers() {
						item, sdpErr := c.gcpComputeAutoscalerToSDPItem(ctx, autoscaler, scopeLocation)
						if sdpErr != nil {
							stream.SendError(sdpErr)
							continue
						}

						cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
						stream.SendItem(item)
					}
				}
			}

			return nil
		})
	}

	// Wait for all goroutines to complete
	_ = p.Wait()
}

func (c computeAutoscalerWrapper) gcpComputeAutoscalerToSDPItem(ctx context.Context, autoscaler *computepb.Autoscaler, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(autoscaler)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeAutoscaler.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		// Autoscalers don't have labels.
	}

	instanceGroupManagerName := autoscaler.GetTarget()
	if instanceGroupManagerName != "" {
		igmNameParts := strings.Split(instanceGroupManagerName, "/")
		igmName := igmNameParts[len(igmNameParts)-1]
		scope, err := gcpshared.ExtractScopeFromURI(ctx, instanceGroupManagerName)
		if err == nil {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeInstanceGroupManager.String(),
					Method: sdp.QueryMethod_GET,
					Query:  igmName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
	}

	return sdpItem, nil
}
