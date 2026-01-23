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
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeForwardingRuleLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeForwardingRule)

type computeForwardingRuleWrapper struct {
	client gcpshared.ComputeForwardingRuleClient
	*gcpshared.RegionBase
}

// NewComputeForwardingRule creates a new computeForwardingRuleWrapper.
func NewComputeForwardingRule(client gcpshared.ComputeForwardingRuleClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeForwardingRuleWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			gcpshared.ComputeForwardingRule,
		),
	}
}

func (c computeForwardingRuleWrapper) IAMPermissions() []string {
	return []string{
		"compute.forwardingRules.get",
		"compute.forwardingRules.list",
	}
}

func (c computeForwardingRuleWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeForwardingRuleWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		stdlib.NetworkIP,
		gcpshared.ComputeSubnetwork,
		gcpshared.ComputeNetwork,
		gcpshared.ComputeBackendService,
		gcpshared.ComputeTargetHttpProxy,
		gcpshared.ComputeTargetHttpsProxy,
		gcpshared.ComputeTargetTcpProxy,
		gcpshared.ComputeTargetSslProxy,
		gcpshared.ComputeTargetPool,
		gcpshared.ComputeTargetVpnGateway,
		gcpshared.ComputeTargetInstance,
		gcpshared.ComputeServiceAttachment,
		gcpshared.ComputeForwardingRule,
		gcpshared.ComputePublicDelegatedPrefix,
		gcpshared.ServiceDirectoryNamespace,
		gcpshared.ServiceDirectoryService,
	)
}

func (c computeForwardingRuleWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_forwarding_rule.name",
		},
	}
}

func (c computeForwardingRuleWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeForwardingRuleLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute forwarding rules since they use aggregatedList
func (c computeForwardingRuleWrapper) SupportsWildcardScope() bool {
	return true
}

func (c computeForwardingRuleWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetForwardingRuleRequest{
		Project:        location.ProjectID,
		Region:         location.Region,
		ForwardingRule: queryParts[0],
	}

	rule, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeForwardingRuleToSDPItem(ctx, rule, location)
}

func (c computeForwardingRuleWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeForwardingRuleWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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

	it := c.client.List(ctx, &computepb.ListForwardingRulesRequest{
		Project: location.ProjectID,
		Region:  location.Region,
	})

	for {
		rule, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeForwardingRuleToSDPItem(ctx, rule, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// listAggregatedStream uses AggregatedList to stream all forwarding rules across all regions
func (c computeForwardingRuleWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := c.GetProjectIDs()

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListForwardingRulesRequest{
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

				// Parse scope from pair.Key (e.g., "regions/us-central1")
				scopeLocation, err := gcpshared.ParseAggregatedListScope(projectID, pair.Key)
				if err != nil {
					continue // Skip unparseable scopes
				}

				// Only process if this scope is in our adapter's configured locations
				if !c.HasLocation(scopeLocation) {
					continue
				}

				// Process forwarding rules in this scope
				if pair.Value != nil && pair.Value.GetForwardingRules() != nil {
					for _, rule := range pair.Value.GetForwardingRules() {
						item, sdpErr := c.gcpComputeForwardingRuleToSDPItem(ctx, rule, scopeLocation)
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

func (c computeForwardingRuleWrapper) gcpComputeForwardingRuleToSDPItem(ctx context.Context, rule *computepb.ForwardingRule, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(rule, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeForwardingRule.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            rule.GetLabels(),
	}

	if rule.GetIPAddress() != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkIP.String(),
				Method: sdp.QueryMethod_GET,
				Query:  rule.GetIPAddress(),
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	if rule.GetBackendService() != "" {
		if strings.Contains(rule.GetBackendService(), "/") {
			backendServiceName := gcpshared.LastPathComponent(rule.GetBackendService())
			scope, err := gcpshared.ExtractScopeFromURI(ctx, rule.GetBackendService())
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeBackendService.String(),
						Method: sdp.QueryMethod_GET,
						Query:  backendServiceName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	if rule.GetPscConnectionStatus() != "" {
		switch rule.GetPscConnectionStatus() {
		case computepb.ForwardingRule_UNDEFINED_PSC_CONNECTION_STATUS.String(),
			computepb.ForwardingRule_STATUS_UNSPECIFIED.String():
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		case computepb.ForwardingRule_ACCEPTED.String():
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case computepb.ForwardingRule_PENDING.String():
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case computepb.ForwardingRule_REJECTED.String(), computepb.ForwardingRule_CLOSED.String():
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		case computepb.ForwardingRule_NEEDS_ATTENTION.String():
			sdpItem.Health = sdp.Health_HEALTH_WARNING.Enum()
		}
	}

	if rule.GetNetwork() != "" {
		if strings.Contains(rule.GetNetwork(), "/") {
			networkName := gcpshared.LastPathComponent(rule.GetNetwork())
			scope, err := gcpshared.ExtractScopeFromURI(ctx, rule.GetNetwork())
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeNetwork.String(),
						Method: sdp.QueryMethod_GET,
						Query:  networkName,
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

	if subnetwork := rule.GetSubnetwork(); subnetwork != "" {
		if strings.Contains(subnetwork, "/") {
			subnetworkName := gcpshared.LastPathComponent(subnetwork)
			scope, err := gcpshared.ExtractScopeFromURI(ctx, subnetwork)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeSubnetwork.String(),
						Method: sdp.QueryMethod_GET,
						Query:  subnetworkName,
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

	// Link to target resource (polymorphic)
	if target := rule.GetTarget(); target != "" {
		linkedQuery := gcpshared.ForwardingRuleTargetLinker(location.ProjectID, location.ToScope(), target, &sdp.BlastPropagation{
			In:  true,
			Out: true,
		})
		if linkedQuery != nil {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, linkedQuery)
		}
	}

	// Link to base forwarding rule
	if baseForwardingRule := rule.GetBaseForwardingRule(); baseForwardingRule != "" {
		if strings.Contains(baseForwardingRule, "/") {
			forwardingRuleName := gcpshared.LastPathComponent(baseForwardingRule)
			scope, err := gcpshared.ExtractScopeFromURI(ctx, baseForwardingRule)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeForwardingRule.String(),
						Method: sdp.QueryMethod_GET,
						Query:  forwardingRuleName,
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

	// Link to Public Delegated Prefix
	if ipCollection := rule.GetIpCollection(); ipCollection != "" {
		if strings.Contains(ipCollection, "/") {
			prefixName := gcpshared.LastPathComponent(ipCollection)
			scope, err := gcpshared.ExtractScopeFromURI(ctx, ipCollection)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputePublicDelegatedPrefix.String(),
						Method: sdp.QueryMethod_GET,
						Query:  prefixName,
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

	// Link to Service Directory
	for _, reg := range rule.GetServiceDirectoryRegistrations() {
		if namespace := reg.GetNamespace(); namespace != "" {
			loc := gcpshared.ExtractPathParam("locations", namespace)
			namespaceName := gcpshared.ExtractPathParam("namespaces", namespace)
			if loc != "" && namespaceName != "" {
				query := shared.CompositeLookupKey(loc, namespaceName)
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ServiceDirectoryNamespace.String(),
						Method: sdp.QueryMethod_GET,
						Query:  query,
						Scope:  location.ProjectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}

		if service := reg.GetService(); service != "" {
			namespace := reg.GetNamespace()
			if namespace != "" && service != "" {
				loc := gcpshared.ExtractPathParam("locations", namespace)
				namespaceName := gcpshared.ExtractPathParam("namespaces", namespace)
				if loc != "" && namespaceName != "" {
					query := shared.CompositeLookupKey(loc, namespaceName, service)
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ServiceDirectoryService.String(),
							Method: sdp.QueryMethod_GET,
							Query:  query,
							Scope:  location.ProjectID,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}
