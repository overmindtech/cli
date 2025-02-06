package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func functionItemMapper(_, scope string, awsItem *types.FunctionSummary) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "cloudfront-function",
		UniqueAttribute: "Name",
		Attributes:      attributes,
		Scope:           scope,
	}

	return &item, nil
}

func NewCloudfrontCloudfrontFunctionAdapter(client *cloudfront.Client, accountID string) *adapterhelpers.GetListAdapter[*types.FunctionSummary, *cloudfront.Client, *cloudfront.Options] {
	return &adapterhelpers.GetListAdapter[*types.FunctionSummary, *cloudfront.Client, *cloudfront.Options]{
		ItemType:        "cloudfront-function",
		Client:          client,
		AccountID:       accountID,
		Region:          "", // Cloudfront resources aren't tied to a region
		AdapterMetadata: cloudfrontFunctionAdapterMetadata,
		GetFunc: func(ctx context.Context, client *cloudfront.Client, scope, query string) (*types.FunctionSummary, error) {
			out, err := client.DescribeFunction(ctx, &cloudfront.DescribeFunctionInput{
				Name: &query,
			})

			if err != nil {
				return nil, err
			}

			return out.FunctionSummary, nil
		},
		ListFunc: func(ctx context.Context, client *cloudfront.Client, scope string) ([]*types.FunctionSummary, error) {
			out, err := client.ListFunctions(ctx, &cloudfront.ListFunctionsInput{
				Stage: types.FunctionStageLive,
			})

			if err != nil {
				return nil, err
			}

			summaries := make([]*types.FunctionSummary, 0, len(out.FunctionList.Items))

			for _, item := range out.FunctionList.Items {
				summaries = append(summaries, &item)
			}

			return summaries, nil
		},
		ItemMapper: functionItemMapper,
	}
}

var cloudfrontFunctionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "cloudfront-function",
	DescriptiveName: "CloudFront Function",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a CloudFront Function by name",
		ListDescription:   "List CloudFront Functions",
		SearchDescription: "Search CloudFront Functions by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_cloudfront_function.name"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
