package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func streamingDistributionGetFunc(ctx context.Context, client CloudFrontClient, scope string, input *cloudfront.GetStreamingDistributionInput) (*sdp.Item, error) {
	out, err := client.GetStreamingDistribution(ctx, input)

	if err != nil {
		return nil, err
	}

	d := out.StreamingDistribution

	if d == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "streaming distribution was nil",
		}
	}

	var tags map[string]string

	// Get the tags
	tagsOut, err := client.ListTagsForResource(ctx, &cloudfront.ListTagsForResourceInput{
		Resource: d.ARN,
	})

	if err == nil {
		tags = cloudfrontTagsToMap(tagsOut.Tags)
	} else {
		tags = adapterhelpers.HandleTagsError(ctx, err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tags for streaming distribution %v: %w", *d.Id, err)
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(d)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "cloudfront-streaming-distribution",
		UniqueAttribute: "Id",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            tags,
	}

	if d.Status != nil {
		switch *d.Status {
		case "InProgress":
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Deployed":
			item.Health = sdp.Health_HEALTH_OK.Enum()
		}
	}

	if d.DomainName != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *d.DomainName,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// DNS is always linked
				In:  true,
				Out: true,
			},
		})
	}

	if dc := d.StreamingDistributionConfig; dc != nil {
		if dc.S3Origin != nil {
			if dc.S3Origin.DomainName != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *dc.S3Origin.DomainName,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly linked
						In:  true,
						Out: true,
					},
				})
			}

			if dc.S3Origin.OriginAccessIdentity != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "cloudfront-cloud-front-origin-access-identity",
						Method: sdp.QueryMethod_GET,
						Query:  *dc.S3Origin.OriginAccessIdentity,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the access identity will affect the distribution
						In: true,
						// The distribution won't affect the access identity
						Out: false,
					},
				})
			}
		}

		if dc.Aliases != nil {
			for _, alias := range dc.Aliases.Items {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  alias,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly linked
						In:  true,
						Out: true,
					},
				})
			}
		}

		if dc.Logging != nil && dc.Logging.Bucket != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *dc.Logging.Bucket,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Tightly linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	return &item, nil
}

func NewCloudfrontStreamingDistributionAdapter(client CloudFrontClient, accountID string) *adapterhelpers.AlwaysGetAdapter[*cloudfront.ListStreamingDistributionsInput, *cloudfront.ListStreamingDistributionsOutput, *cloudfront.GetStreamingDistributionInput, *cloudfront.GetStreamingDistributionOutput, CloudFrontClient, *cloudfront.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*cloudfront.ListStreamingDistributionsInput, *cloudfront.ListStreamingDistributionsOutput, *cloudfront.GetStreamingDistributionInput, *cloudfront.GetStreamingDistributionOutput, CloudFrontClient, *cloudfront.Options]{
		ItemType:        "cloudfront-streaming-distribution",
		Client:          client,
		AccountID:       accountID,
		Region:          "", // Cloudfront resources aren't tied to a region
		AdapterMetadata: streamingDistributionAdapterMetadata,
		ListInput:       &cloudfront.ListStreamingDistributionsInput{},
		ListFuncPaginatorBuilder: func(client CloudFrontClient, input *cloudfront.ListStreamingDistributionsInput) adapterhelpers.Paginator[*cloudfront.ListStreamingDistributionsOutput, *cloudfront.Options] {
			return cloudfront.NewListStreamingDistributionsPaginator(client, input)
		},
		GetInputMapper: func(scope, query string) *cloudfront.GetStreamingDistributionInput {
			return &cloudfront.GetStreamingDistributionInput{
				Id: &query,
			}
		},
		ListFuncOutputMapper: func(output *cloudfront.ListStreamingDistributionsOutput, input *cloudfront.ListStreamingDistributionsInput) ([]*cloudfront.GetStreamingDistributionInput, error) {
			var inputs []*cloudfront.GetStreamingDistributionInput

			for _, sd := range output.StreamingDistributionList.Items {
				inputs = append(inputs, &cloudfront.GetStreamingDistributionInput{
					Id: sd.Id,
				})
			}

			return inputs, nil
		},
		GetFunc: streamingDistributionGetFunc,
	}
}

var streamingDistributionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "CloudFront Streaming Distribution",
	Type:            "cloudfront-streaming-distribution",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Search:         true,
		Get:            true,
		List:           true,
		GetDescription: "Get a Streaming Distribution by ID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "aws_cloudfront_distribution.arn",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "aws_cloudfront_distribution.id",
		},
	},
	PotentialLinks: []string{"dns"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
