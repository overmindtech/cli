package adapters

import (
	"context"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type elbClient interface {
	DescribeTags(ctx context.Context, params *elb.DescribeTagsInput, optFns ...func(*elb.Options)) (*elb.DescribeTagsOutput, error)
	DescribeLoadBalancers(ctx context.Context, params *elb.DescribeLoadBalancersInput, optFns ...func(*elb.Options)) (*elb.DescribeLoadBalancersOutput, error)
}

func elbTagsToMap(tags []types.Tag) map[string]string {
	m := make(map[string]string)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			m[*tag.Key] = *tag.Value
		}
	}

	return m
}

func elbLoadBalancerOutputMapper(ctx context.Context, client elbClient, scope string, _ *elb.DescribeLoadBalancersInput, output *elb.DescribeLoadBalancersOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	loadBalancerNames := make([]string, 0)
	for _, desc := range output.LoadBalancerDescriptions {
		if desc.LoadBalancerName != nil {
			loadBalancerNames = append(loadBalancerNames, *desc.LoadBalancerName)
		}
	}

	// Map of load balancer name to tags
	tagsMap := make(map[string][]types.Tag)
	if len(loadBalancerNames) > 0 {
		// Get all tags for all load balancers in this output
		tagsOut, err := client.DescribeTags(ctx, &elb.DescribeTagsInput{
			LoadBalancerNames: loadBalancerNames,
		})

		if err == nil {
			for _, tagDesc := range tagsOut.TagDescriptions {
				if tagDesc.LoadBalancerName != nil {
					tagsMap[*tagDesc.LoadBalancerName] = tagDesc.Tags
				}
			}
		}
	}

	for _, desc := range output.LoadBalancerDescriptions {
		attrs, err := adapterhelpers.ToAttributesWithExclude(desc)

		if err != nil {
			return nil, err
		}

		var tags map[string]string

		if desc.LoadBalancerName != nil {
			m, ok := tagsMap[*desc.LoadBalancerName]

			if ok {
				tags = elbTagsToMap(m)
			}
		}

		item := sdp.Item{
			Type:            "elb-load-balancer",
			UniqueAttribute: "LoadBalancerName",
			Attributes:      attrs,
			Scope:           scope,
			Tags:            tags,
		}

		if desc.DNSName != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *desc.DNSName,
				Scope:  "global",
			}})
		}

		if desc.CanonicalHostedZoneName != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *desc.CanonicalHostedZoneName,
				Scope:  "global",
			}})
		}

		if desc.CanonicalHostedZoneNameID != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
				Type:   "route53-hosted-zone",
				Method: sdp.QueryMethod_GET,
				Query:  *desc.CanonicalHostedZoneNameID,
				Scope:  scope,
			}})
		}

		for _, subnet := range desc.Subnets {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
				Type:   "ec2-subnet",
				Method: sdp.QueryMethod_GET,
				Query:  subnet,
				Scope:  scope,
			}})
		}

		if desc.VPCId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
				Type:   "ec2-vpc",
				Method: sdp.QueryMethod_GET,
				Query:  *desc.VPCId,
				Scope:  scope,
			}})
		}

		for _, instance := range desc.Instances {
			if instance.InstanceId != nil {
				// The EC2 instance itself
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
					Type:   "ec2-instance",
					Method: sdp.QueryMethod_GET,
					Query:  *instance.InstanceId,
					Scope:  scope,
				}})

				if desc.LoadBalancerName != nil {
					name := InstanceHealthName{
						LoadBalancerName: *desc.LoadBalancerName,
						InstanceId:       *instance.InstanceId,
					}

					// The health for that instance
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
						Type:   "elb-instance-health",
						Method: sdp.QueryMethod_GET,
						Query:  name.String(),
						Scope:  scope,
					}})
				}
			}
		}

		if desc.SourceSecurityGroup != nil {
			if desc.SourceSecurityGroup.GroupName != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
					Type:   "ec2-security-group",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *desc.SourceSecurityGroup.GroupName,
					Scope:  scope,
				}})
			}
		}

		for _, sg := range desc.SecurityGroups {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{Query: &sdp.Query{
				Type:   "ec2-security-group",
				Method: sdp.QueryMethod_GET,
				Query:  sg,
				Scope:  scope,
			}})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewELBLoadBalancerAdapter(client elbClient, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*elb.DescribeLoadBalancersInput, *elb.DescribeLoadBalancersOutput, elbClient, *elb.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*elb.DescribeLoadBalancersInput, *elb.DescribeLoadBalancersOutput, elbClient, *elb.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "elb-load-balancer",
		AdapterMetadata: elbLoadBalancerAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client elbClient, input *elb.DescribeLoadBalancersInput) (*elb.DescribeLoadBalancersOutput, error) {
			return client.DescribeLoadBalancers(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*elb.DescribeLoadBalancersInput, error) {
			return &elb.DescribeLoadBalancersInput{
				LoadBalancerNames: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*elb.DescribeLoadBalancersInput, error) {
			return &elb.DescribeLoadBalancersInput{}, nil
		},
		PaginatorBuilder: func(client elbClient, params *elb.DescribeLoadBalancersInput) adapterhelpers.Paginator[*elb.DescribeLoadBalancersOutput, *elb.Options] {
			return elb.NewDescribeLoadBalancersPaginator(client, params)
		},
		OutputMapper: elbLoadBalancerOutputMapper,
	}
}

var elbLoadBalancerAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "elb-load-balancer",
	DescriptiveName: "Classic Load Balancer",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a classic load balancer by name",
		ListDescription:   "List all classic load balancers",
		SearchDescription: "Search for classic load balancers by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_elb.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"dns", "route53-hosted-zone", "ec2-subnet", "ec2-vpc", "ec2-instance", "elb-instance-health", "ec2-security-group"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
