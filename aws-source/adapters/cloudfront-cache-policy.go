package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func cachePolicyListFunc(ctx context.Context, client CloudFrontClient, scope string) ([]*types.CachePolicy, error) {
	var policyType types.CachePolicyType

	switch scope {
	case "aws":
		policyType = types.CachePolicyTypeManaged
	default:
		policyType = types.CachePolicyTypeCustom
	}

	out, err := client.ListCachePolicies(ctx, &cloudfront.ListCachePoliciesInput{
		Type: policyType,
	})

	if err != nil {
		return nil, err
	}

	policies := make([]*types.CachePolicy, 0, len(out.CachePolicyList.Items))

	for i := range out.CachePolicyList.Items {
		policies = append(policies, out.CachePolicyList.Items[i].CachePolicy)
	}

	return policies, nil
}

func NewCloudfrontCachePolicyAdapter(client CloudFrontClient, accountID string) *adapterhelpers.GetListAdapter[*types.CachePolicy, CloudFrontClient, *cloudfront.Options] {
	return &adapterhelpers.GetListAdapter[*types.CachePolicy, CloudFrontClient, *cloudfront.Options]{
		ItemType:               "cloudfront-cache-policy",
		Client:                 client,
		AccountID:              accountID,
		Region:                 "", // Cloudfront resources aren't tied to a region
		AdapterMetadata:        cachePolicyAdapterMetadata,
		SupportGlobalResources: true, // Some policies are global
		GetFunc: func(ctx context.Context, client CloudFrontClient, scope, query string) (*types.CachePolicy, error) {
			out, err := client.GetCachePolicy(ctx, &cloudfront.GetCachePolicyInput{
				Id: &query,
			})

			if err != nil {
				return nil, err
			}

			return out.CachePolicy, nil
		},
		ListFunc: cachePolicyListFunc,
		ItemMapper: func(_, scope string, awsItem *types.CachePolicy) (*sdp.Item, error) {
			attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

			if err != nil {
				return nil, err
			}

			item := sdp.Item{
				Type:            "cloudfront-cache-policy",
				UniqueAttribute: "Id",
				Attributes:      attributes,
				Scope:           scope,
			}

			return &item, nil
		},
	}
}

var cachePolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "cloudfront-cache-policy",
	DescriptiveName: "CloudFront Cache Policy",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a CloudFront Cache Policy",
		ListDescription:   "List CloudFront Cache Policies",
		SearchDescription: "Search CloudFront Cache Policies by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_cloudfront_cache_policy.id"},
	},
})
