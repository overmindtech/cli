package adapters

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

func (c TestCloudFrontClient) ListTagsForResource(ctx context.Context, params *cloudfront.ListTagsForResourceInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListTagsForResourceOutput, error) {
	return &cloudfront.ListTagsForResourceOutput{
		Tags: &types.Tags{
			Items: []types.Tag{
				{
					Key:   PtrString("foo"),
					Value: PtrString("bar"),
				},
			},
		},
	}, nil
}

type TestCloudFrontClient struct{}

func CloudfrontGetAutoConfig(t *testing.T) (*cloudfront.Client, string, string) {
	config, account, region := GetAutoConfig(t)
	client := cloudfront.NewFromConfig(config)

	return client, account, region
}
