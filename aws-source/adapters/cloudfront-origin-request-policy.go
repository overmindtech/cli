package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func originRequestPolicyItemMapper(_, scope string, awsItem *types.OriginRequestPolicy) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "cloudfront-origin-request-policy",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
	}

	return &item, nil
}

func NewCloudfrontOriginRequestPolicyAdapter(client *cloudfront.Client, accountID string) *adapterhelpers.GetListAdapter[*types.OriginRequestPolicy, *cloudfront.Client, *cloudfront.Options] {
	return &adapterhelpers.GetListAdapter[*types.OriginRequestPolicy, *cloudfront.Client, *cloudfront.Options]{
		ItemType:        "cloudfront-origin-request-policy",
		Client:          client,
		AccountID:       accountID,
		Region:          "", // Cloudfront resources aren't tied to a region
		AdapterMetadata: originRequestPolicyAdapterMetadata,
		GetFunc: func(ctx context.Context, client *cloudfront.Client, scope, query string) (*types.OriginRequestPolicy, error) {
			out, err := client.GetOriginRequestPolicy(ctx, &cloudfront.GetOriginRequestPolicyInput{
				Id: &query,
			})

			if err != nil {
				return nil, err
			}

			return out.OriginRequestPolicy, nil
		},
		ListFunc: func(ctx context.Context, client *cloudfront.Client, scope string) ([]*types.OriginRequestPolicy, error) {
			out, err := client.ListOriginRequestPolicies(ctx, &cloudfront.ListOriginRequestPoliciesInput{})

			if err != nil {
				return nil, err
			}

			policies := make([]*types.OriginRequestPolicy, 0, len(out.OriginRequestPolicyList.Items))

			for _, policy := range out.OriginRequestPolicyList.Items {
				policies = append(policies, policy.OriginRequestPolicy)
			}

			return policies, nil
		},
		ItemMapper: originRequestPolicyItemMapper,
	}
}

var originRequestPolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "cloudfront-origin-request-policy",
	DescriptiveName: "CloudFront Origin Request Policy",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get Origin Request Policy by ID",
		ListDescription:   "List Origin Request Policies",
		SearchDescription: "Origin Request Policy by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_cloudfront_origin_request_policy.id"},
	},
})
