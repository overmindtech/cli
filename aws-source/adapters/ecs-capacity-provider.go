package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

var CapacityProviderIncludeFields = []types.CapacityProviderField{
	types.CapacityProviderFieldTags,
}

func capacityProviderOutputMapper(_ context.Context, _ ECSClient, scope string, _ *ecs.DescribeCapacityProvidersInput, output *ecs.DescribeCapacityProvidersOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, provider := range output.CapacityProviders {
		attributes, err := adapterhelpers.ToAttributesWithExclude(provider, "tags")

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "ecs-capacity-provider",
			UniqueAttribute: "Name",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            ecsTagsToMap(provider.Tags),
		}

		if provider.AutoScalingGroupProvider != nil {
			if provider.AutoScalingGroupProvider.AutoScalingGroupArn != nil {
				if a, err := adapterhelpers.ParseARN(*provider.AutoScalingGroupProvider.AutoScalingGroupArn); err == nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "autoscaling-auto-scaling-group",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *provider.AutoScalingGroupProvider.AutoScalingGroupArn,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// These are tightly linked
							In:  true,
							Out: true,
						},
					})
				}
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewECSCapacityProviderAdapter(client ECSClient, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ecs.DescribeCapacityProvidersInput, *ecs.DescribeCapacityProvidersOutput, ECSClient, *ecs.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ecs.DescribeCapacityProvidersInput, *ecs.DescribeCapacityProvidersOutput, ECSClient, *ecs.Options]{
		ItemType:        "ecs-capacity-provider",
		Region:          region,
		AccountID:       accountID,
		Client:          client,
		AdapterMetadata: capacityProviderAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client ECSClient, input *ecs.DescribeCapacityProvidersInput) (*ecs.DescribeCapacityProvidersOutput, error) {
			return client.DescribeCapacityProviders(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*ecs.DescribeCapacityProvidersInput, error) {
			return &ecs.DescribeCapacityProvidersInput{
				CapacityProviders: []string{
					query,
				},
				Include: CapacityProviderIncludeFields,
			}, nil
		},
		InputMapperList: func(scope string) (*ecs.DescribeCapacityProvidersInput, error) {
			return &ecs.DescribeCapacityProvidersInput{
				Include: CapacityProviderIncludeFields,
			}, nil
		},
		PaginatorBuilder: func(client ECSClient, params *ecs.DescribeCapacityProvidersInput) adapterhelpers.Paginator[*ecs.DescribeCapacityProvidersOutput, *ecs.Options] {
			return NewDescribeCapacityProvidersPaginator(client, params)
		},
		OutputMapper: capacityProviderOutputMapper,
	}
}

var capacityProviderAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ecs-capacity-provider",
	DescriptiveName: "Capacity Provider",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		GetDescription:    "Get a capacity provider by its short name or full Amazon Resource Name (ARN).",
		List:              true,
		ListDescription:   "List capacity providers.",
		Search:            true,
		SearchDescription: "Search capacity providers by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_ecs_capacity_provider.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"autoscaling-auto-scaling-group"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})

// Incredibly annoyingly the go package adapters't provide a paginator builder for
// DescribeCapacityProviders despite the fact that it's paginated, so I'm going
// to create one myself below

// DescribeCapacityProvidersPaginator is a paginator for DescribeCapacityProviders
type DescribeCapacityProvidersPaginator struct {
	client    ECSClient
	params    *ecs.DescribeCapacityProvidersInput
	nextToken *string
	firstPage bool
}

// NewDescribeCapacityProvidersPaginator returns a new DescribeCapacityProvidersPaginator
func NewDescribeCapacityProvidersPaginator(client ECSClient, params *ecs.DescribeCapacityProvidersInput) *DescribeCapacityProvidersPaginator {
	if params == nil {
		params = &ecs.DescribeCapacityProvidersInput{}
	}

	return &DescribeCapacityProvidersPaginator{
		client:    client,
		params:    params,
		firstPage: true,
		nextToken: params.NextToken,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *DescribeCapacityProvidersPaginator) HasMorePages() bool {
	return p.firstPage || (p.nextToken != nil && len(*p.nextToken) != 0)
}

// NextPage retrieves the next DescribeCapacityProviders page.
func (p *DescribeCapacityProvidersPaginator) NextPage(ctx context.Context, optFns ...func(*ecs.Options)) (*ecs.DescribeCapacityProvidersOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.NextToken = p.nextToken

	result, err := p.client.DescribeCapacityProviders(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.nextToken
	p.nextToken = result.NextToken

	if prevToken != nil &&
		p.nextToken != nil &&
		*prevToken == *p.nextToken {
		p.nextToken = nil
	}

	return result, nil
}
