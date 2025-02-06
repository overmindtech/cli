package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

// Converts a CloudFront Tags object to a map
func cloudfrontTagsToMap(tags *types.Tags) map[string]string {
	if tags == nil {
		return nil
	}

	tagMap := make(map[string]string)

	for _, tag := range tags.Items {
		if tag.Key != nil && tag.Value != nil {
			tagMap[*tag.Key] = *tag.Value
		}
	}

	return tagMap
}

type CloudFrontClient interface {
	GetCachePolicy(ctx context.Context, params *cloudfront.GetCachePolicyInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetCachePolicyOutput, error)
	ListCachePolicies(ctx context.Context, params *cloudfront.ListCachePoliciesInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListCachePoliciesOutput, error)

	GetDistribution(ctx context.Context, params *cloudfront.GetDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionOutput, error)
	ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error)

	GetStreamingDistribution(ctx context.Context, params *cloudfront.GetStreamingDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetStreamingDistributionOutput, error)
	ListStreamingDistributions(ctx context.Context, params *cloudfront.ListStreamingDistributionsInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListStreamingDistributionsOutput, error)

	ListTagsForResource(ctx context.Context, params *cloudfront.ListTagsForResourceInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListTagsForResourceOutput, error)
}
