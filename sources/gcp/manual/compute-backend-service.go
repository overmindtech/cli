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
)

var ComputeBackendServiceLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeBackendService)

type computeBackendServiceWrapper struct {
	globalClient     gcpshared.ComputeBackendServiceClient
	regionalClient   gcpshared.ComputeRegionBackendServiceClient
	projectLocations []gcpshared.LocationInfo // For global backend services
	regionLocations  []gcpshared.LocationInfo // For regional backend services
	*shared.Base
}

// NewComputeBackendService creates a new computeBackendServiceWrapper instance that handles both global and regional backend services.
func NewComputeBackendService(globalClient gcpshared.ComputeBackendServiceClient, regionalClient gcpshared.ComputeRegionBackendServiceClient, projectLocations []gcpshared.LocationInfo, regionLocations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	// Combine all locations for scope generation
	allLocations := make([]gcpshared.LocationInfo, 0, len(projectLocations)+len(regionLocations))
	allLocations = append(allLocations, projectLocations...)
	allLocations = append(allLocations, regionLocations...)

	scopes := make([]string, 0, len(allLocations))
	for _, location := range allLocations {
		scopes = append(scopes, location.ToScope())
	}

	return &computeBackendServiceWrapper{
		globalClient:     globalClient,
		regionalClient:   regionalClient,
		projectLocations: projectLocations,
		regionLocations:  regionLocations,
		Base:             shared.NewBase(sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION, gcpshared.ComputeBackendService, scopes),
	}
}

// validateAndParseScope parses the scope and validates it against configured locations.
// Returns the LocationInfo if valid, or a QueryError if the scope is invalid or not configured.
func (c computeBackendServiceWrapper) validateAndParseScope(scope string) (gcpshared.LocationInfo, *sdp.QueryError) {
	location, err := gcpshared.LocationFromScope(scope)
	if err != nil {
		return gcpshared.LocationInfo{}, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	// Check if the location is in the adapter's configured locations
	allLocations := append([]gcpshared.LocationInfo{}, c.projectLocations...)
	allLocations = append(allLocations, c.regionLocations...)

	for _, configuredLoc := range allLocations {
		if location.Equals(configuredLoc) {
			return location, nil
		}
	}

	return gcpshared.LocationInfo{}, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOSCOPE,
		ErrorString: fmt.Sprintf("scope %s not found in adapter's configured locations", scope),
	}
}

func (c computeBackendServiceWrapper) IAMPermissions() []string {
	return []string{
		"compute.backendServices.get",
		"compute.backendServices.list",
		"compute.regionBackendServices.get",
		"compute.regionBackendServices.list",
	}
}

func (c computeBackendServiceWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (computeBackendServiceWrapper) PotentialLinks() map[shared.ItemType]bool {
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
		gcpshared.ComputeRegion,
	)
}

func (c computeBackendServiceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_backend_service.name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_region_backend_service.name",
		},
	}
}

func (c computeBackendServiceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeBackendServiceLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for backend services since they use aggregatedList
func (c computeBackendServiceWrapper) SupportsWildcardScope() bool {
	return true
}

func (c computeBackendServiceWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	// Parse and validate the scope
	location, err := c.validateAndParseScope(scope)
	if err != nil {
		return nil, err
	}

	// Route to the appropriate API based on whether the scope includes a region
	if location.Regional() {
		// Regional backend service
		req := &computepb.GetRegionBackendServiceRequest{
			Project:        location.ProjectID,
			Region:         location.Region,
			BackendService: queryParts[0],
		}

		service, getErr := c.regionalClient.Get(ctx, req)
		if getErr != nil {
			return nil, gcpshared.QueryError(getErr, scope, c.Type())
		}

		return gcpComputeBackendServiceToSDPItem(ctx, location.ProjectID, location.ToScope(), service, gcpshared.ComputeBackendService)
	}

	// Global backend service
	req := &computepb.GetBackendServiceRequest{
		Project:        location.ProjectID,
		BackendService: queryParts[0],
	}

	service, getErr := c.globalClient.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return gcpComputeBackendServiceToSDPItem(ctx, location.ProjectID, location.ToScope(), service, gcpshared.ComputeBackendService)
}

func (c computeBackendServiceWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeBackendServiceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	// Handle wildcard scope with AggregatedList
	if scope == "*" {
		c.listAggregatedStream(ctx, stream, cache, cacheKey)
		return
	}

	// Parse and validate the scope
	location, err := c.validateAndParseScope(scope)
	if err != nil {
		stream.SendError(err)
		return
	}

	// Route to the appropriate API based on whether the scope includes a region
	if location.Regional() {
		// Regional backend services
		it := c.regionalClient.List(ctx, &computepb.ListRegionBackendServicesRequest{
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

			item, sdpErr := gcpComputeBackendServiceToSDPItem(ctx, location.ProjectID, location.ToScope(), backendService, gcpshared.ComputeBackendService)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}

			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	} else {
		// Global backend services
		it := c.globalClient.List(ctx, &computepb.ListBackendServicesRequest{
			Project: location.ProjectID,
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

			item, sdpErr := gcpComputeBackendServiceToSDPItem(ctx, location.ProjectID, location.ToScope(), backendService, gcpshared.ComputeBackendService)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}

			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// listAggregatedStream uses AggregatedList to stream all backend services across all regions (global and regional)
func (c computeBackendServiceWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := gcpshared.GetProjectIDsFromLocations(c.projectLocations, c.regionLocations)

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.globalClient.AggregatedList(ctx, &computepb.AggregatedListBackendServicesRequest{
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

				// Parse scope from pair.Key (e.g., "global" or "regions/us-central1")
				scopeLocation, err := gcpshared.ParseAggregatedListScope(projectID, pair.Key)
				if err != nil {
					continue // Skip unparseable scopes
				}

				// Only process if this scope is in our adapter's configured locations
				if !gcpshared.HasLocationInSlices(scopeLocation, c.projectLocations, c.regionLocations) {
					continue
				}

				// Process backend services in this scope
				if pair.Value != nil && pair.Value.GetBackendServices() != nil {
					for _, backendService := range pair.Value.GetBackendServices() {
						item, sdpErr := gcpComputeBackendServiceToSDPItem(ctx, scopeLocation.ProjectID, scopeLocation.ToScope(), backendService, gcpshared.ComputeBackendService)
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

func gcpComputeBackendServiceToSDPItem(ctx context.Context, projectID string, scope string, bs *computepb.BackendService, itemType shared.ItemType) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(bs)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            itemType.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
	}

	// The URL of the network to which this backend service belongs.
	// This field must be set for Internal Passthrough Network Load Balancers when the haPolicy is enabled,
	// and for External Passthrough Network Load Balancers when the haPolicy fastIpMove is enabled.
	// This field can only be specified when the load balancing scheme is set to INTERNAL.
	if network := bs.GetNetwork(); network != "" {
		if strings.Contains(network, "/") {
			networkName := gcpshared.LastPathComponent(network)
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  networkName,
					// This is a global resource
					Scope: projectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// TODO: We need keyring as well for linking keys.
	// So, at this point, without a proper integration tests, we don't have enough confidence to link this.
	// Names of the keys for signing request URLs.
	// signedURLKeyNames := bs.GetCdnPolicy().GetSignedUrlKeyNames()

	// The resource URL for the security policy associated with this backend service.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/securityPolicies/{securityPolicy}
	// https://cloud.google.com/compute/docs/reference/rest/v1/securityPolicies/get
	if securityPolicy := bs.GetSecurityPolicy(); securityPolicy != "" {
		if strings.Contains(securityPolicy, "/") {
			securityPolicyName := gcpshared.LastPathComponent(securityPolicy)
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeSecurityPolicy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  securityPolicyName,
					Scope:  projectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// The resource URL for the edge security policy associated with this backend service.
	if edgeSecurityPolicy := bs.GetEdgeSecurityPolicy(); edgeSecurityPolicy != "" {
		if strings.Contains(edgeSecurityPolicy, "/") {
			edgeSecurityPolicyName := gcpshared.LastPathComponent(edgeSecurityPolicy)
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeSecurityPolicy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  edgeSecurityPolicyName,
					Scope:  projectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Optional. A URL referring to a networksecurity.ClientTlsPolicy resource that describes how clients should authenticate with this service's backends.
	// clientTlsPolicy only applies to a global BackendService with the loadBalancingScheme set to INTERNAL_SELF_MANAGED.
	// If left blank, communications are not encrypted.
	if bs.GetSecuritySettings() != nil {
		if clientTlsPolicy := bs.GetSecuritySettings().GetClientTlsPolicy(); clientTlsPolicy != "" {
			// The URL should look like this:
			// GET https://networksecurity.googleapis.com/v1/{name=projects/*/locations/*/clientTlsPolicies/*}
			// See: https://cloud.google.com/service-mesh/docs/reference/network-security/rest/v1/projects.locations.clientTlsPolicies/get
			// This will be a global resource but it will require a location dynamically.
			// So, we need to extract the location and the policy name from the URL.
			if strings.Contains(clientTlsPolicy, "/") {
				params := gcpshared.ExtractPathParams(clientTlsPolicy, "locations", "clientTlsPolicies")
				if len(params) == 2 && params[0] != "" && params[1] != "" {
					location := params[0]
					policyName := params[1]
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							// The resource name will be: "gcp-network-security-client-tls-policy"
							Type:   gcpshared.NetworkSecurityClientTlsPolicy.String(),
							Method: sdp.QueryMethod_GET,
							// This is a global resource but it will require a location dynamically.
							Query: shared.CompositeLookupKey(location, policyName),
							Scope: projectID,
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

	// Health checks are used by the backend service to probe the health of its backends.
	// At most one health check can be specified per backend service.
	// For regional backend services, these are typically regional health checks.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/healthChecks/{healthCheck}
	// or GET https://compute.googleapis.com/compute/v1/projects/{project}/global/healthChecks/{healthCheck}
	// https://cloud.google.com/compute/docs/reference/rest/v1/regionHealthChecks/get
	// https://cloud.google.com/compute/docs/reference/rest/v1/healthChecks/get
	if healthChecks := bs.GetHealthChecks(); len(healthChecks) > 0 {
		// At most one health check is allowed, but we iterate in case multiple are present
		for _, healthCheckURL := range healthChecks {
			if healthCheckURL != "" && strings.Contains(healthCheckURL, "/") {
				// Extract scope from the health check URL (could be global or regional)
				healthCheckScope, err := gcpshared.ExtractScopeFromURI(ctx, healthCheckURL)
				if err != nil {
					// If scope extraction fails, skip this health check
					continue
				}

				// Extract health check name from URL
				healthCheckName := gcpshared.LastPathComponent(healthCheckURL)
				if healthCheckName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeHealthCheck.String(),
							Method: sdp.QueryMethod_GET,
							Query:  healthCheckName,
							Scope:  healthCheckScope,
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

	for _, backend := range bs.GetBackends() {
		if backend.GetGroup() != "" {
			// The group field is a URL to a Compute Instance Group or Network Endpoint Group.
			// We can link it to the Compute Instance Group or Network Endpoint Group.
			if strings.Contains(backend.GetGroup(), "/nodeGroups/") {
				// https://cloud.google.com/compute/docs/reference/rest/v1/nodeGroups/get#http-request
				params := gcpshared.ExtractPathParams(backend.GetGroup(), "zones", "nodeGroups")
				if len(params) == 2 && params[0] != "" && params[1] != "" {
					zone := params[0]
					groupName := params[1]
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeInstanceGroup.String(),
							Method: sdp.QueryMethod_GET,
							Query:  groupName,
							Scope:  zone,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					})
				}
			}
			if strings.Contains(backend.GetGroup(), "/networkEndpointGroups/") {
				// Network Endpoint Groups can be zonal, regional, or global
				// https://cloud.google.com/compute/docs/reference/rest/v1/networkEndpointGroups/get
				// https://cloud.google.com/compute/docs/reference/rest/v1/regionNetworkEndpointGroups/get
				// https://cloud.google.com/compute/docs/reference/rest/v1/globalNetworkEndpointGroups/get
				// Extract scope from the NEG URL
				negScope, err := gcpshared.ExtractScopeFromURI(ctx, backend.GetGroup())
				if err != nil {
					// Fallback to zonal extraction for backward compatibility
					params := gcpshared.ExtractPathParams(backend.GetGroup(), "zones", "networkEndpointGroups")
					if len(params) == 2 && params[0] != "" && params[1] != "" {
						zone := params[0]
						negName := params[1]
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeNetworkEndpointGroup.String(),
								Method: sdp.QueryMethod_GET,
								Query:  negName,
								Scope:  zone,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: false,
							},
						})
					}
				} else {
					// Use scope extraction for zonal, regional, or global NEGs
					negName := gcpshared.LastPathComponent(backend.GetGroup())
					if negName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeNetworkEndpointGroup.String(),
								Method: sdp.QueryMethod_GET,
								Query:  negName,
								Scope:  negScope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: false,
							},
						})
					}
				}
			}
			// Also check for instanceGroups (unmanaged instance groups)
			if strings.Contains(backend.GetGroup(), "/instanceGroups/") {
				// https://cloud.google.com/compute/docs/reference/rest/v1/instanceGroups/get
				params := gcpshared.ExtractPathParams(backend.GetGroup(), "zones", "instanceGroups")
				if len(params) == 2 && params[0] != "" && params[1] != "" {
					zone := params[0]
					groupName := params[1]
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeInstanceGroup.String(),
							Method: sdp.QueryMethod_GET,
							Query:  groupName,
							Scope:  zone,
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

	// URL to networkservices.ServiceLbPolicy resource. Can only be set if load balancing scheme is EXTERNAL, EXTERNAL_MANAGED, INTERNAL_MANAGED or INTERNAL_SELF_MANAGED and the scope is global.
	// GET https://networkservices.googleapis.com/v1/{name=projects/*/locations/*/serviceLbPolicies/*}
	// https://cloud.google.com/service-mesh/docs/reference/network-services/rest/v1/projects.locations.serviceLbPolicies/get
	if serviceLbPolicy := bs.GetServiceLbPolicy(); serviceLbPolicy != "" {
		if strings.Contains(serviceLbPolicy, "/") {
			params := gcpshared.ExtractPathParams(serviceLbPolicy, "locations", "serviceLbPolicies")
			if len(params) == 2 && params[0] != "" && params[1] != "" {
				location := params[0]
				policyName := params[1]
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.NetworkServicesServiceLbPolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, policyName),
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	// URLs of networkservices.ServiceBinding resources. Can only be set if load balancing scheme is INTERNAL_SELF_MANAGED. If set, lists of backends and health checks must be both empty.
	// GET https://networkservices.googleapis.com/v1alpha1/{name=projects/*/locations/*/serviceBindings/*}
	// https://cloud.google.com/service-mesh/docs/reference/network-services/rest/v1alpha1/projects.locations.serviceBindings/get
	if serviceBindings := bs.GetServiceBindings(); serviceBindings != nil {
		for _, serviceBinding := range serviceBindings {
			if strings.Contains(serviceBinding, "/") {
				params := gcpshared.ExtractPathParams(serviceBinding, "locations", "serviceBindings")
				if len(params) == 2 && params[0] != "" && params[1] != "" {
					location := params[0]
					bindingName := params[1]
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.NetworkServicesServiceBinding.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(location, bindingName),
							Scope:  projectID,
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

	// HA Policy (High Availability Policy) for External Passthrough and Internal Passthrough Network Load Balancers.
	// Used for self-managed high availability with zonal NEG backends.
	// GET https://cloud.google.com/compute/docs/reference/rest/v1/backendServices#BackendService
	if haPolicy := bs.GetHaPolicy(); haPolicy != nil {
		if leader := haPolicy.GetLeader(); leader != nil {
			// Link to the Network Endpoint Group containing the leader endpoint
			// haPolicy.leader.backendGroup is a fully-qualified URL of the zonal NEG containing the leader endpoint.
			// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/networkEndpointGroups/{networkEndpointGroup}
			// https://cloud.google.com/compute/docs/reference/rest/v1/networkEndpointGroups/get
			if backendGroup := leader.GetBackendGroup(); backendGroup != "" {
				negScope, err := gcpshared.ExtractScopeFromURI(ctx, backendGroup)
				if err == nil {
					negName := gcpshared.LastPathComponent(backendGroup)
					if negName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeNetworkEndpointGroup.String(),
								Method: sdp.QueryMethod_GET,
								Query:  negName,
								Scope:  negScope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: false,
							},
						})
					}
				}
			}

			// Link to the Compute Instance designated as leader
			// haPolicy.leader.networkEndpoint.instance is the name of the VM instance in the NEG to be leader.
			// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{instance}
			// https://cloud.google.com/compute/docs/reference/rest/v1/instances/get
			if networkEndpoint := leader.GetNetworkEndpoint(); networkEndpoint != nil {
				if instanceName := networkEndpoint.GetInstance(); instanceName != "" {
					// The instance name alone is not enough - we need to extract the zone from the backendGroup
					// Since the leader must be in the same NEG as specified in backendGroup, we can extract zone from there
					if backendGroup := leader.GetBackendGroup(); backendGroup != "" {
						// Extract zone from backendGroup URL
						zone := gcpshared.ExtractPathParam("zones", backendGroup)
						if zone != "" {
							instanceScope := gcpshared.ZonalScope(projectID, zone)
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   gcpshared.ComputeInstance.String(),
									Method: sdp.QueryMethod_GET,
									Query:  instanceName,
									Scope:  instanceScope,
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
		}
	}

	// The URL of the region where the regional backend service resides.
	// This field is output-only and is not applicable to global backend services.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}
	// https://cloud.google.com/compute/docs/reference/rest/v1/regions/get
	if region := bs.GetRegion(); region != "" {
		if strings.Contains(region, "/") {
			regionNameParts := strings.Split(region, "/")
			regionName := regionNameParts[len(regionNameParts)-1]
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeRegion.String(),
					Method: sdp.QueryMethod_GET,
					Query:  regionName,
					// Regions are project-scoped resources
					Scope: projectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	return sdpItem, nil
}
