package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeBackendServiceLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeBackendService)

type computeBackendServiceWrapper struct {
	client gcpshared.ComputeBackendServiceClient

	*gcpshared.ProjectBase
}

// NewComputeBackendService creates a new computeBackendServiceWrapper instance
func NewComputeBackendService(client gcpshared.ComputeBackendServiceClient, projectID string) sources.ListableWrapper {
	return &computeBackendServiceWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeBackendService,
		),
	}
}

func (c computeBackendServiceWrapper) IAMPermissions() []string {
	return []string{
		"compute.backendServices.get",
		"compute.backendServices.list",
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
	)
}

// TerraformMappings returns the Terraform mappings for the compute backend service wrapper
func (c computeBackendServiceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_backend_service#argument-reference
			TerraformQueryMap: "google_compute_backend_service.name",
		},
	}
}

// GetLookups returns the lookups for the compute backend service wrapper
func (c computeBackendServiceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeBackendServiceLookupByName,
	}
}

// Get retrieves a compute backend service by its name
func (c computeBackendServiceWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetBackendServiceRequest{
		Project:        c.ProjectID(),
		BackendService: queryParts[0],
	}

	service, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), c.DefaultScope(), service)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute backend services and converts them to sdp.Items.
func (c computeBackendServiceWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListBackendServicesRequest{
		Project: c.ProjectID(),
	})

	var items []*sdp.Item
	for {
		bs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), c.DefaultScope(), bs)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream lists compute backend services and sends them to the stream.
func (c computeBackendServiceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListBackendServicesRequest{
		Project: c.ProjectID(),
	})

	for {
		backendService, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := gcpComputeBackendServiceToSDPItem(c.ProjectID(), c.DefaultScope(), backendService)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func gcpComputeBackendServiceToSDPItem(projectID string, scope string, bs *computepb.BackendService) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(bs)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeBackendService.String(),
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
			networkNameParts := strings.Split(network, "/")
			networkName := networkNameParts[len(networkNameParts)-1]
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
			securityPolicyNameParts := strings.Split(securityPolicy, "/")
			if len(securityPolicyNameParts) >= 2 {
				securityPolicyName := securityPolicyNameParts[len(securityPolicyNameParts)-1]
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeSecurityPolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  securityPolicyName,
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
	}

	// The resource URL for the edge security policy associated with this backend service.
	if edgeSecurityPolicy := bs.GetEdgeSecurityPolicy(); edgeSecurityPolicy != "" {
		if strings.Contains(edgeSecurityPolicy, "/") {
			edgeSecurityPolicyNameParts := strings.Split(edgeSecurityPolicy, "/")
			edgeSecurityPolicyName := edgeSecurityPolicyNameParts[len(edgeSecurityPolicyNameParts)-1]
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeSecurityPolicy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  edgeSecurityPolicyName,
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
				// https://cloud.google.com/compute/docs/reference/rest/v1/networkEndpointGroups/get#http-request
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

	return sdpItem, nil
}
