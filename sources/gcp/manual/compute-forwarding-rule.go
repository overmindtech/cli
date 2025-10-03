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
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeForwardingRuleLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeForwardingRule)

type computeForwardingRuleWrapper struct {
	client gcpshared.ComputeForwardingRuleClient

	*gcpshared.RegionBase
}

// NewComputeForwardingRule creates a new computeForwardingRuleWrapper
func NewComputeForwardingRule(client gcpshared.ComputeForwardingRuleClient, projectID, region string) sources.ListableWrapper {
	return &computeForwardingRuleWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			projectID,
			region,
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

// PotentialLinks returns the potential links for the compute forwarding rule wrapper
func (c computeForwardingRuleWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		stdlib.NetworkIP,
		gcpshared.ComputeSubnetwork,
		gcpshared.ComputeNetwork,
		gcpshared.ComputeBackendService,
	)
}

// TerraformMappings returns the Terraform mappings for the compute forwarding rule wrapper
func (c computeForwardingRuleWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_disk#argument-reference
			TerraformQueryMap: "google_compute_forwarding_rule.name",
		},
	}
}

// GetLookups returns the lookups for the compute forwarding rule wrapper
func (c computeForwardingRuleWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeForwardingRuleLookupByName,
	}
}

// Get retrieves a compute forwarding rule by its name
func (c computeForwardingRuleWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetForwardingRuleRequest{
		Project:        c.ProjectID(),
		Region:         c.Region(),
		ForwardingRule: queryParts[0],
	}

	rule, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	item, sdpErr := c.gcpComputeForwardingRuleToSDPItem(rule)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute forwarding rules and converts them to sdp.Items.
func (c computeForwardingRuleWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListForwardingRulesRequest{
		Project: c.ProjectID(),
		Region:  c.Region(),
	})

	var items []*sdp.Item
	for {
		rule, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		item, sdpErr := c.gcpComputeForwardingRuleToSDPItem(rule)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeForwardingRuleWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListForwardingRulesRequest{
		Project: c.ProjectID(),
		Region:  c.Region(),
	})

	for {
		rule, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeForwardingRuleToSDPItem(rule)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeForwardingRuleWrapper) gcpComputeForwardingRuleToSDPItem(rule *computepb.ForwardingRule) (*sdp.Item, *sdp.QueryError) {
	// Convert the forwarding rule to attributes
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
		Scope:           c.DefaultScope(),
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
		// The URL for backend service is in the format:
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/backendServices/{backendService}
		// We need to extract the backend service name and region
		// from the URL and create a linked item query for it
		if strings.Contains(rule.GetBackendService(), "/") {
			backendServiceNameParts := strings.Split(rule.GetBackendService(), "/")
			backendServiceName := backendServiceNameParts[len(backendServiceNameParts)-1]
			region := gcpshared.ExtractPathParam("regions", rule.GetBackendService())
			if region != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeBackendService.String(),
						Method: sdp.QueryMethod_GET,
						Query:  backendServiceName,
						// This is a regional resource
						Scope: gcpshared.RegionalScope(c.ProjectID(), region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// They are tightly coupled
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	// TODO: Further investigate if we can link an item via rule.GetPscConnectionId()

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
			networkNameParts := strings.Split(rule.GetNetwork(), "/")
			networkName := networkNameParts[len(networkNameParts)-1]
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeNetwork.String(),
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

	if subnetwork := rule.GetSubnetwork(); subnetwork != "" {
		if strings.Contains(subnetwork, "/") {
			subnetworkNameParts := strings.Split(subnetwork, "/")
			subnetworkName := subnetworkNameParts[len(subnetworkNameParts)-1]
			region := gcpshared.ExtractPathParam("regions", subnetwork)
			if region != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeSubnetwork.String(),
						Method: sdp.QueryMethod_GET,
						Query:  subnetworkName,
						// This is a regional resource
						Scope: gcpshared.RegionalScope(c.ProjectID(), region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}

	}

	return sdpItem, nil
}
