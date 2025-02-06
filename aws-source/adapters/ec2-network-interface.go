package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func networkInterfaceInputMapperGet(scope string, query string) (*ec2.DescribeNetworkInterfacesInput, error) {
	return &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []string{
			query,
		},
	}, nil
}

func networkInterfaceInputMapperList(scope string) (*ec2.DescribeNetworkInterfacesInput, error) {
	return &ec2.DescribeNetworkInterfacesInput{}, nil
}

func networkInterfaceOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeNetworkInterfacesInput, output *ec2.DescribeNetworkInterfacesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, ni := range output.NetworkInterfaces {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(ni, "tagSet")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-network-interface",
			UniqueAttribute: "NetworkInterfaceId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(ni.TagSet),
		}

		if ni.Attachment != nil {
			if ni.Attachment.InstanceId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-instance",
						Method: sdp.QueryMethod_GET,
						Query:  *ni.Attachment.InstanceId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// The instance and the interface are closely linked
						// and affect each other
						In:  true,
						Out: true,
					},
				})
			}
		}

		for _, sg := range ni.Groups {
			if sg.GroupId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-security-group",
						Method: sdp.QueryMethod_GET,
						Query:  *sg.GroupId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// A security group will affect an interface
						In: true,
						// An interface won't affect a security group
						Out: false,
					},
				})
			}
		}

		for _, ip := range ni.Ipv6Addresses {
			if ip.Ipv6Address != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *ip.Ipv6Address,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// IPs are always linked
						In:  true,
						Out: true,
					},
				})
			}
		}

		for _, ip := range ni.PrivateIpAddresses {
			if assoc := ip.Association; assoc != nil {
				if assoc.PublicDnsName != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "dns",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *assoc.PublicDnsName,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// DNS names are always linked
							In:  true,
							Out: true,
						},
					})
				}

				if assoc.PublicIp != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *assoc.PublicIp,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// IPs are always linked
							In:  true,
							Out: true,
						},
					})
				}

				if assoc.CarrierIp != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *assoc.CarrierIp,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// IPs are always linked
							In:  true,
							Out: true,
						},
					})
				}

				if assoc.CustomerOwnedIp != nil {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *assoc.CustomerOwnedIp,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// IPs are always linked
							In:  true,
							Out: true,
						},
					})
				}
			}

			if ip.PrivateDnsName != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *ip.PrivateDnsName,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// DNS names are always linked
						In:  true,
						Out: true,
					},
				})
			}

			if ip.PrivateIpAddress != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *ip.PrivateIpAddress,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// IPs are always linked
						In:  true,
						Out: true,
					},
				})
			}
		}

		if ni.SubnetId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-subnet",
					Method: sdp.QueryMethod_GET,
					Query:  *ni.SubnetId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the subnet will affect interfaces within that
					// subnet
					In: true,
					// Changing the interface won't affect the subnet
					Out: false,
				},
			})
		}

		if ni.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *ni.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the VPC will affect interfaces within that VPC
					In: true,
					// Changing the interface won't affect the VPC
					Out: false,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2NetworkInterfaceAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeNetworkInterfacesInput, *ec2.DescribeNetworkInterfacesOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeNetworkInterfacesInput, *ec2.DescribeNetworkInterfacesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-network-interface",
		AdapterMetadata: networkInterfaceAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeNetworkInterfacesInput) (*ec2.DescribeNetworkInterfacesOutput, error) {
			return client.DescribeNetworkInterfaces(ctx, input)
		},
		InputMapperGet:  networkInterfaceInputMapperGet,
		InputMapperList: networkInterfaceInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeNetworkInterfacesInput) adapterhelpers.Paginator[*ec2.DescribeNetworkInterfacesOutput, *ec2.Options] {
			return ec2.NewDescribeNetworkInterfacesPaginator(client, params)
		},
		OutputMapper: networkInterfaceOutputMapper,
	}
}

var networkInterfaceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-network-interface",
	DescriptiveName: "EC2 Network Interface",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a network interface by ID",
		ListDescription:   "List all network interfaces",
		SearchDescription: "Search network interfaces by ARN",
	},
	PotentialLinks: []string{"ec2-instance", "ec2-security-group", "ip", "dns", "ec2-subnet", "ec2-vpc"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_network_interface.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
