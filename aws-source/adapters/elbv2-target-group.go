package adapters

import (
	"context"
	"fmt"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func targetGroupOutputMapper(ctx context.Context, client elbv2Client, scope string, _ *elbv2.DescribeTargetGroupsInput, output *elbv2.DescribeTargetGroupsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	tgArns := make([]string, 0)

	for _, tg := range output.TargetGroups {
		if tg.TargetGroupArn != nil {
			tgArns = append(tgArns, *tg.TargetGroupArn)
		}
	}

	tagsMap := elbv2GetTagsMap(ctx, client, tgArns)

	for _, tg := range output.TargetGroups {
		attrs, err := adapterhelpers.ToAttributesWithExclude(tg)

		if err != nil {
			return nil, err
		}

		var tags map[string]string

		if tg.TargetGroupArn != nil {
			tags = tagsMap[*tg.TargetGroupArn]
		}

		item := sdp.Item{
			Type:            "elbv2-target-group",
			UniqueAttribute: "TargetGroupName",
			Attributes:      attrs,
			Scope:           scope,
			Tags:            tags,
		}

		if tg.TargetGroupArn != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "elbv2-target-health",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *tg.TargetGroupArn,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Target groups and their target health are tightly coupled
					In:  true,
					Out: true,
				},
			})
		}

		if tg.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *tg.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the VPC can affect the target group
					In: true,
					// The target group won't affect the VPC
					Out: false,
				},
			})
		}

		for _, lbArn := range tg.LoadBalancerArns {
			if a, err := adapterhelpers.ParseARN(lbArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "elbv2-load-balancer",
						Method: sdp.QueryMethod_SEARCH,
						Query:  lbArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Load balancers and their target groups are tightly coupled
						In:  true,
						Out: true,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewELBv2TargetGroupAdapter(client elbv2Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*elbv2.DescribeTargetGroupsInput, *elbv2.DescribeTargetGroupsOutput, elbv2Client, *elbv2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*elbv2.DescribeTargetGroupsInput, *elbv2.DescribeTargetGroupsOutput, elbv2Client, *elbv2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "elbv2-target-group",
		AdapterMetadata: targetGroupAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client elbv2Client, input *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error) {
			return client.DescribeTargetGroups(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*elbv2.DescribeTargetGroupsInput, error) {
			return &elbv2.DescribeTargetGroupsInput{
				Names: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*elbv2.DescribeTargetGroupsInput, error) {
			return &elbv2.DescribeTargetGroupsInput{}, nil
		},
		InputMapperSearch: func(ctx context.Context, client elbv2Client, scope, query string) (*elbv2.DescribeTargetGroupsInput, error) {
			arn, err := adapterhelpers.ParseARN(query)

			if err != nil {
				return nil, err
			}

			switch arn.Type() {
			case "targetgroup":
				// Search by target group
				return &elbv2.DescribeTargetGroupsInput{
					TargetGroupArns: []string{
						query,
					},
				}, nil
			case "loadbalancer":
				// Search by load balancer
				return &elbv2.DescribeTargetGroupsInput{
					LoadBalancerArn: &query,
				}, nil
			default:
				return nil, fmt.Errorf("unsupported resource type: %s", arn.Resource)
			}
		},
		PaginatorBuilder: func(client elbv2Client, params *elbv2.DescribeTargetGroupsInput) adapterhelpers.Paginator[*elbv2.DescribeTargetGroupsOutput, *elbv2.Options] {
			return elbv2.NewDescribeTargetGroupsPaginator(client, params)
		},
		OutputMapper: targetGroupOutputMapper,
	}
}

var targetGroupAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "elbv2-target-group",
	DescriptiveName: "Target Group",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a target group by name",
		ListDescription:   "List all target groups",
		SearchDescription: "Search for target groups by load balancer ARN or target group ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_alb_target_group.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
		{
			TerraformQueryMap: "aws_lb_target_group.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"ec2-vpc", "elbv2-load-balancer", "elbv2-target-health"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
