package adapters

import (
	"context"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
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
		attrs, err := adapterhelpers.ToAttributesWithExclude(lb)

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
				BlastPropagation: &sdp.BlastPropagation{
					// Load balancers and their target groups are tightly coupled
					In:  true,
					Out: true,
				},
			})

			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "elbv2-listener",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *lb.LoadBalancerArn,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Load balancers and their listeners are tightly coupled
					In:  true,
					Out: true,
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
				BlastPropagation: &sdp.BlastPropagation{
					// DNS always links
					In:  true,
					Out: true,
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
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the hosted zone could affect the LB
					In: true,
					// The LB won't affect the hosted zone
					Out: false,
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
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the VPC could affect the LB
					In: true,
					// The LB won't affect the VPC
					Out: false,
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
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the subnet could affect the LB
						In: true,
						// The LB won't affect the subnet
						Out: false,
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
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the address could affect the LB
							In: true,
							// The LB can also affect the address
							Out: true,
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
						BlastPropagation: &sdp.BlastPropagation{
							// IPs always link
							In:  true,
							Out: true,
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
						BlastPropagation: &sdp.BlastPropagation{
							// IPs always link
							In:  true,
							Out: true,
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
						BlastPropagation: &sdp.BlastPropagation{
							// IPs always link
							In:  true,
							Out: true,
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
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the security group could affect the LB
					In: true,
					// The LB won't affect the security group
					Out: false,
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
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the COIP pool could affect the LB
					In: true,
					// The LB won't affect the COIP pool
					Out: false,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewELBv2LoadBalancerAdapter(client elbv2Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*elbv2.DescribeLoadBalancersInput, *elbv2.DescribeLoadBalancersOutput, elbv2Client, *elbv2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*elbv2.DescribeLoadBalancersInput, *elbv2.DescribeLoadBalancersOutput, elbv2Client, *elbv2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "elbv2-load-balancer",
		AdapterMetadata: loadBalancerAdapterMetadata,
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
		PaginatorBuilder: func(client elbv2Client, params *elbv2.DescribeLoadBalancersInput) adapterhelpers.Paginator[*elbv2.DescribeLoadBalancersOutput, *elbv2.Options] {
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
