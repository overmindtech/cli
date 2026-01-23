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

// NewComputeSecurityPolicy creates a new computeSecurityPolicyWrapper instance.
func NewComputeSecurityPolicy(client gcpshared.ComputeSecurityPolicyClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeSecurityPolicyWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
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

func (c computeSecurityPolicyWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeSecurityPolicyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeRule,
	)
}

func (c computeSecurityPolicyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_security_policy.name",
		},
	}
}

func (c computeSecurityPolicyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeSecurityPolicyLookupByName,
	}
}

func (c computeSecurityPolicyWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetSecurityPolicyRequest{
		Project:        location.ProjectID,
		SecurityPolicy: queryParts[0],
	}

	policy, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeSecurityPolicyToSDPItem(policy, location)
}

func (c computeSecurityPolicyWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeSecurityPolicyWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListSecurityPoliciesRequest{
		Project: location.ProjectID,
	})

	for {
		securityPolicy, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeSecurityPolicyToSDPItem(securityPolicy, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		stream.SendItem(item)
	}
}

func (c computeSecurityPolicyWrapper) gcpComputeSecurityPolicyToSDPItem(securityPolicy *computepb.SecurityPolicy, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           location.ToScope(),
		Tags:            securityPolicy.GetLabels(),
	}

	// Link to associated rules
	for _, rule := range securityPolicy.GetRules() {
		policyName := securityPolicy.GetName()
		rulePriority := strconv.Itoa(int(rule.GetPriority()))
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.ComputeRule.String(),
				Method: sdp.QueryMethod_GET,
				Query:  shared.CompositeLookupKey(policyName, rulePriority),
				Scope:  location.ProjectID,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  false,
				Out: true,
			},
		})
	}

	return sdpItem, nil
}
