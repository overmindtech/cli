package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func (c TestCloudFrontClient) ListTagsForResource(ctx context.Context, params *cloudfront.ListTagsForResourceInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListTagsForResourceOutput, error) {
	return &cloudfront.ListTagsForResourceOutput{
		Tags: &types.Tags{
			Items: []types.Tag{
				{
					Key:   adapterhelpers.PtrString("foo"),
					Value: adapterhelpers.PtrString("bar"),
				},
			},
		},
	}, nil
}

type TestCloudFrontClient struct{}

func CloudfrontGetAutoConfig(t *testing.T) (*cloudfront.Client, string, string) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := cloudfront.NewFromConfig(config)

	return client, account, region
}
