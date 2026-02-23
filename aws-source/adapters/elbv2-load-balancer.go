package adapters

import (
	"context"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func elbv2LoadBalancerOutputMapper(ctx context.Context, client elbv2Client, scope string, _ *elbv2.DescribeLoadBalancersInput, output *elbv2.DescribeLoadBalancersOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	// Get the ARNs so that we can get the tags
	arns := make([]string, 0)

	for _, lb := range output.LoadBalancers {
		if lb.LoadBalancerArn != nil {
			arns = append(arns, *lb.LoadBalancerArn)
		}
	}

	tagsMap := elbv2GetTagsMap(ctx, client, arns)

	for _, lb := range output.LoadBalancers {
		attrs, err := ToAttributesWithExclude(lb)

		if err != nil {
			return nil, err
		}

		var tags map[string]string

		if lb.LoadBalancerArn != nil {
			tags = tagsMap[*lb.LoadBalancerArn]
		}

		item := sdp.Item{
			Type:            "elbv2-load-balancer",
			UniqueAttribute: "LoadBalancerName",
			Attributes:      attrs,
			Scope:           scope,
			Tags:            tags,
		}

		if lb.LoadBalancerArn != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "elbv2-target-group",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *lb.LoadBalancerArn,
					Scope:  scope,
				},
			})

			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "elbv2-listener",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *lb.LoadBalancerArn,
					Scope:  scope,
				},
			})
		}

		if lb.DNSName != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *lb.DNSName,
					Scope:  "global",
				},
			})
		}

		if lb.CanonicalHostedZoneId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "route53-hosted-zone",
					Method: sdp.QueryMethod_GET,
					Query:  *lb.CanonicalHostedZoneId,
					Scope:  scope,
				},
			})
		}

		if lb.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *lb.VpcId,
					Scope:  scope,
				},
			})
		}

		for _, az := range lb.AvailabilityZones {
			if az.SubnetId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-subnet",
						Method: sdp.QueryMethod_GET,
						Query:  *az.SubnetId,
						Scope:  scope,
					},
				})
			}

			for _, address := range az.LoadBalancerAddresses {
				if address.AllocationId != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-address",
							Method: sdp.QueryMethod_GET,
							Query:  *address.AllocationId,
							Scope:  scope,
						},
					})
				}

				if address.IPv6Address != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *address.IPv6Address,
							Scope:  "global",
						},
					})
				}

				if address.IpAddress != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *address.IpAddress,
							Scope:  "global",
						},
					})
				}

				if address.PrivateIPv4Address != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *address.PrivateIPv4Address,
							Scope:  "global",
						},
					})
				}
			}
		}

		for _, sg := range lb.SecurityGroups {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-security-group",
					Method: sdp.QueryMethod_GET,
					Query:  sg,
					Scope:  scope,
				},
			})
		}

		if lb.CustomerOwnedIpv4Pool != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-coip-pool",
					Method: sdp.QueryMethod_GET,
					Query:  *lb.CustomerOwnedIpv4Pool,
					Scope:  scope,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewELBv2LoadBalancerAdapter(client elbv2Client, accountID string, region string, cache sdpcache.Cache) *DescribeOnlyAdapter[*elbv2.DescribeLoadBalancersInput, *elbv2.DescribeLoadBalancersOutput, elbv2Client, *elbv2.Options] {
	return &DescribeOnlyAdapter[*elbv2.DescribeLoadBalancersInput, *elbv2.DescribeLoadBalancersOutput, elbv2Client, *elbv2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "elbv2-load-balancer",
		AdapterMetadata: loadBalancerAdapterMetadata,
		cache:           cache,
		DescribeFunc: func(ctx context.Context, client elbv2Client, input *elbv2.DescribeLoadBalancersInput) (*elbv2.DescribeLoadBalancersOutput, error) {
			return client.DescribeLoadBalancers(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*elbv2.DescribeLoadBalancersInput, error) {
			return &elbv2.DescribeLoadBalancersInput{
				Names: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*elbv2.DescribeLoadBalancersInput, error) {
			return &elbv2.DescribeLoadBalancersInput{}, nil
		},
		InputMapperSearch: func(ctx context.Context, client elbv2Client, scope, query string) (*elbv2.DescribeLoadBalancersInput, error) {
			return &elbv2.DescribeLoadBalancersInput{
				LoadBalancerArns: []string{query},
			}, nil
		},
		PaginatorBuilder: func(client elbv2Client, params *elbv2.DescribeLoadBalancersInput) Paginator[*elbv2.DescribeLoadBalancersOutput, *elbv2.Options] {
			return elbv2.NewDescribeLoadBalancersPaginator(client, params)
		},
		OutputMapper: elbv2LoadBalancerOutputMapper,
	}
}

var loadBalancerAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "elbv2-load-balancer",
	DescriptiveName: "Elastic Load Balancer",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an ELB by name",
		ListDescription:   "List all ELBs",
		SearchDescription: "Search for ELBs by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_lb.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
		{
			TerraformQueryMap: "aws_lb.id",
			TerraformMethod:   sdp.QueryMethod_GET,
		},
	},
	PotentialLinks: []string{"elbv2-target-group", "elbv2-listener", "dns", "route53-hosted-zone", "ec2-vpc", "ec2-subnet", "ec2-address", "ip", "ec2-security-group", "ec2-coip-pool"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
