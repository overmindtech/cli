package manual

import (
	"context"
	"errors"
	"fmt"
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

var ComputeAddressLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeAddress)

type computeAddressWrapper struct {
	client gcpshared.ComputeAddressClient
	*gcpshared.RegionBase
}

// NewComputeAddress creates a new computeAddressWrapper.
func NewComputeAddress(client gcpshared.ComputeAddressClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeAddressWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			gcpshared.ComputeAddress,
		),
	}
}

func (c computeAddressWrapper) IAMPermissions() []string {
	return []string{
		"compute.addresses.get",
		"compute.addresses.list",
	}
}

func (c computeAddressWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeAddressWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		stdlib.NetworkIP,
		gcpshared.ComputeAddress,
		gcpshared.ComputeSubnetwork,
		gcpshared.ComputeNetwork,
		gcpshared.ComputeForwardingRule,
		gcpshared.ComputeGlobalForwardingRule,
		gcpshared.ComputeInstance,
		gcpshared.ComputeTargetVpnGateway,
		gcpshared.ComputeRouter,
		gcpshared.ComputePublicDelegatedPrefix,
	)
}

func (c computeAddressWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_address.name",
		},
	}
}

func (c computeAddressWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeAddressLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute addresses since they use aggregatedList
func (c computeAddressWrapper) SupportsWildcardScope() bool {
	return true
}

func (c computeAddressWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetAddressRequest{
		Project: location.ProjectID,
		Region:  location.Region,
		Address: queryParts[0],
	}

	address, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeAddressToSDPItem(ctx, address, location)
}

func (c computeAddressWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeAddressWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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

	it := c.client.List(ctx, &computepb.ListAddressesRequest{
		Project: location.ProjectID,
		Region:  location.Region,
	})

	for {
		address, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeAddressToSDPItem(ctx, address, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// listAggregatedStream uses AggregatedList to stream all addresses across all regions
func (c computeAddressWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := c.GetProjectIDs()

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListAddressesRequest{
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

				// Process addresses in this scope
				if pair.Value != nil && pair.Value.GetAddresses() != nil {
					for _, address := range pair.Value.GetAddresses() {
						item, sdpErr := c.gcpComputeAddressToSDPItem(ctx, address, scopeLocation)
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

func (c computeAddressWrapper) gcpComputeAddressToSDPItem(ctx context.Context, address *computepb.Address, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(address, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeAddress.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            address.GetLabels(),
	}

	if network := address.GetNetwork(); network != "" {
		if strings.Contains(network, "/") {
			networkName := gcpshared.LastPathComponent(network)
			scope, err := gcpshared.ExtractScopeFromURI(ctx, network)
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

	if subnetwork := address.GetSubnetwork(); subnetwork != "" {
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

	if ip := address.GetAddress(); ip != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkIP.String(),
				Method: sdp.QueryMethod_GET,
				Query:  ip,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	// Link to resources using this address
	for _, userURI := range address.GetUsers() {
		if userURI != "" {
			linkedQuery := gcpshared.AddressUsersLinker(
				ctx,
				location.ProjectID,
				userURI,
				&sdp.BlastPropagation{In: true, Out: true},
			)
			if linkedQuery != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, linkedQuery)
			}
		}
	}

	// Link to Public Delegated Prefix
	if ipCollection := address.GetIpCollection(); ipCollection != "" {
		if strings.Contains(ipCollection, "/") {
			region := gcpshared.ExtractPathParam("regions", ipCollection)
			prefixName := gcpshared.LastPathComponent(ipCollection)
			if region != "" && prefixName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputePublicDelegatedPrefix.String(),
						Method: sdp.QueryMethod_GET,
						Query:  prefixName,
						Scope:  fmt.Sprintf("%s.%s", location.ProjectID, region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	switch address.GetStatus() {
	case computepb.Address_RESERVING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Address_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.Address_RESERVED.String(),
		computepb.Address_IN_USE.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	}

	return sdpItem, nil
}
