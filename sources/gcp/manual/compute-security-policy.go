package manual

import (
	"context"
	"errors"
	"strconv"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeSecurityPolicyLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeSecurityPolicy)

type computeSecurityPolicyWrapper struct {
	client gcpshared.ComputeSecurityPolicyClient

	*gcpshared.ProjectBase
}

// NewComputeSecurityPolicy creates a new computeSecurityPolicyWrapper instance
func NewComputeSecurityPolicy(client gcpshared.ComputeSecurityPolicyClient, projectID string) sources.ListableWrapper {
	return &computeSecurityPolicyWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.ComputeSecurityPolicy,
		),
	}
}

func (c computeSecurityPolicyWrapper) IAMPermissions() []string {
	return []string{
		"compute.securityPolicies.get",
		"compute.securityPolicies.list",
	}
}

// PotentialLinks returns the potential links for the compute forwarding rule wrapper
func (c computeSecurityPolicyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeRule,
	)
}

// TerraformMappings returns the Terraform mappings for the compute security policy wrapper
func (c computeSecurityPolicyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_security_policy#argument-reference
			TerraformQueryMap: "google_compute_security_policy.name",
		},
	}
}

// GetLookups returns the lookups for the compute security policy wrapper
func (c computeSecurityPolicyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeSecurityPolicyLookupByName,
	}
}

// Get retrieves a compute security policy by its name
func (c computeSecurityPolicyWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetSecurityPolicyRequest{
		Project:        c.ProjectID(),
		SecurityPolicy: queryParts[0],
	}

	policy, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	item, sdpErr := c.gcpComputeSecurityPolicyToSDPItem(policy)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute security policies and converts them to sdp.Items.
func (c computeSecurityPolicyWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListSecurityPoliciesRequest{
		Project: c.ProjectID(),
	})

	var items []*sdp.Item
	for {
		securityPolicy, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		item, sdpErr := c.gcpComputeSecurityPolicyToSDPItem(securityPolicy)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream lists compute security policies and sends them as items to the stream.
func (c computeSecurityPolicyWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListSecurityPoliciesRequest{
		Project: c.ProjectID(),
	})

	for {
		securityPolicy, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeSecurityPolicyToSDPItem(securityPolicy)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		stream.SendItem(item)
	}
}

// gcpComputeSecurityPolicyToSDPItem converts a GCP Security Policy to an SDP Item
func (c computeSecurityPolicyWrapper) gcpComputeSecurityPolicyToSDPItem(securityPolicy *computepb.SecurityPolicy) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(securityPolicy, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeSecurityPolicy.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            securityPolicy.GetLabels(),
	}

	// Link to associated rules
	// API Reference: https://cloud.google.com/compute/docs/reference/rest/v1/securityPolicies/getRule
	// Rules can be inserted into a security policy using the following API call:
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/securityPolicies/{securityPolicy}/getRule
	for _, rule := range securityPolicy.GetRules() {
		policyName := securityPolicy.GetName()
		rulePriority := strconv.Itoa(int(rule.GetPriority()))
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.ComputeRule.String(),
				Method: sdp.QueryMethod_GET,
				Query:  shared.CompositeLookupKey(policyName, rulePriority),
				Scope:  c.ProjectID(),
			},
			BlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
		})

	}
	return sdpItem, nil
}
