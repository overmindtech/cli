package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func ResponseHeadersPolicyItemMapper(_, scope string, awsItem *types.ResponseHeadersPolicy) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "cloudfront-response-headers-policy",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
	}

	return &item, nil
}

func NewCloudfrontResponseHeadersPolicyAdapter(client *cloudfront.Client, accountID string) *adapterhelpers.GetListAdapter[*types.ResponseHeadersPolicy, *cloudfront.Client, *cloudfront.Options] {
	return &adapterhelpers.GetListAdapter[*types.ResponseHeadersPolicy, *cloudfront.Client, *cloudfront.Options]{
		ItemType:        "cloudfront-response-headers-policy",
		Client:          client,
		AccountID:       accountID,
		Region:          "", // Cloudfront resources aren't tied to a region
		AdapterMetadata: responseHeadersPolicyAdapterMetadata,
		GetFunc: func(ctx context.Context, client *cloudfront.Client, scope, query string) (*types.ResponseHeadersPolicy, error) {
			out, err := client.GetResponseHeadersPolicy(ctx, &cloudfront.GetResponseHeadersPolicyInput{
				Id: &query,
			})

			if err != nil {
				return nil, err
			}

			return out.ResponseHeadersPolicy, nil
		},
		ListFunc: func(ctx context.Context, client *cloudfront.Client, scope string) ([]*types.ResponseHeadersPolicy, error) {
			out, err := client.ListResponseHeadersPolicies(ctx, &cloudfront.ListResponseHeadersPoliciesInput{})

			if err != nil {
				return nil, err
			}

			policies := make([]*types.ResponseHeadersPolicy, 0, len(out.ResponseHeadersPolicyList.Items))

			for _, policy := range out.ResponseHeadersPolicyList.Items {
				policies = append(policies, policy.ResponseHeadersPolicy)
			}

			return policies, nil
		},
		ItemMapper: ResponseHeadersPolicyItemMapper,
	}
}

var responseHeadersPolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "cloudfront-response-headers-policy",
	DescriptiveName: "CloudFront Response Headers Policy",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get Response Headers Policy by ID",
		ListDescription:   "List Response Headers Policies",
		SearchDescription: "Search Response Headers Policy by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_cloudfront_response_headers_policy.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
