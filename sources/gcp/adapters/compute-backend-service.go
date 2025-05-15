package adapters

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeBackendService = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.BackendService)

	ComputeBackendServiceLookupByName = shared.NewItemTypeLookup("name", ComputeBackendService)
)

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
			ComputeBackendService,
		),
	}
}

func (computeBackendServiceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		ComputeNetwork,
		ComputeSecurityPolicy,
		NetworkSecurityClientTlsPolicy,
		NetworkServicesServiceLbPolicy,
		NetworkServicesServiceBinding,
	)
}

// TerraformMappings returns the Terraform mappings for the compute backend service wrapper
func (c computeBackendServiceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
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

	bs, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	item, sdpErr := c.gcpComputeBackendServiceToSDPItem(bs)
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
			return nil, gcpshared.QueryError(err)
		}

		item, sdpErr := c.gcpComputeBackendServiceToSDPItem(bs)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeBackendServiceWrapper) gcpComputeBackendServiceToSDPItem(bs *computepb.BackendService) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(bs)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            ComputeBackendService.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
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
					Type:   ComputeNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  networkName,
					// This is a global resource
					Scope: c.ProjectID(),
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
						Type:   ComputeSecurityPolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  securityPolicyName,
						// This is a global resource
						Scope: c.ProjectID(),
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
					Type:   ComputeSecurityPolicy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  edgeSecurityPolicyName,
					// This is a global resource
					Scope: c.ProjectID(),
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
				policyName, location := extractNameAndLocation(clientTlsPolicy)
				if location != "" && policyName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							// The resource name will be: "gcp-network-security-client-tls-policy"
							Type:   NetworkSecurityClientTlsPolicy.String(),
							Method: sdp.QueryMethod_GET,
							// This is a global resource but it will require a location dynamically.
							Query: shared.CompositeLookupKey(location, policyName),
							Scope: c.ProjectID(),
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
			policyName, location := extractNameAndLocation(serviceLbPolicy)
			if location != "" && policyName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   NetworkServicesServiceLbPolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, policyName),
						Scope:  c.ProjectID(),
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
				bindingName, location := extractNameAndLocation(serviceBinding)
				if location != "" && bindingName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   NetworkServicesServiceBinding.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(location, bindingName),
							Scope:  c.ProjectID(),
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

// extractNameAndLocation extracts the name and location from the URL
func extractNameAndLocation(url string) (string, string) {
	// The URL should look like this:
	// GET https://networksecurity.googleapis.com/v1/{name=projects/*/locations/*/clientTlsPolicies/*}
	// So we need to extract the location and the policy name from the URL.
	url = strings.TrimSuffix(url, "/") // Remove trailing slash
	parts := strings.Split(url, "/")
	length := len(parts)
	if length < 8 {
		return "", ""
	}

	// It is not in the format we expect
	if parts[length-4] != "locations" {
		return "", ""
	}

	return parts[length-1], parts[length-3]
}
