package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func natGatewayInputMapperGet(scope string, query string) (*ec2.DescribeNatGatewaysInput, error) {
	return &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{
			query,
		},
	}, nil
}

func natGatewayInputMapperList(scope string) (*ec2.DescribeNatGatewaysInput, error) {
	return &ec2.DescribeNatGatewaysInput{}, nil
}

func natGatewayOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeNatGatewaysInput, output *ec2.DescribeNatGatewaysOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, ng := range output.NatGateways {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(ng, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-nat-gateway",
			UniqueAttribute: "NatGatewayId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(ng.Tags),
		}

		for _, address := range ng.NatGatewayAddresses {
			if address.NetworkInterfaceId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-network-interface",
						Method: sdp.QueryMethod_GET,
						Query:  *address.NetworkInterfaceId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// The nat gateway and it's interfaces will affect each
						// other
						In:  true,
						Out: true,
					},
				})
			}

			if address.PrivateIp != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *address.PrivateIp,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// IPs always link
						In:  true,
						Out: true,
					},
				})
			}

			if address.PublicIp != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *address.PublicIp,
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

		if ng.SubnetId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-subnet",
					Method: sdp.QueryMethod_GET,
					Query:  *ng.SubnetId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the subnet won't affect the gateway
					In: false,
					// Changing the gateway will affect the subnet since this
					// will be gateway that subnet uses to access the internet
					Out: true,
				},
			})
		}

		if ng.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *ng.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the VPC could affect the gateway
					In: true,
					// Changing the gateway won't affect the VPC
					Out: false,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2NatGatewayAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeNatGatewaysInput, *ec2.DescribeNatGatewaysOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeNatGatewaysInput, *ec2.DescribeNatGatewaysOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-nat-gateway",
		AdapterMetadata: natGatewayAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeNatGatewaysInput) (*ec2.DescribeNatGatewaysOutput, error) {
			return client.DescribeNatGateways(ctx, input)
		},
		InputMapperGet:  natGatewayInputMapperGet,
		InputMapperList: natGatewayInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeNatGatewaysInput) adapterhelpers.Paginator[*ec2.DescribeNatGatewaysOutput, *ec2.Options] {
			return ec2.NewDescribeNatGatewaysPaginator(client, params)
		},
		OutputMapper: natGatewayOutputMapper,
	}
}

var natGatewayAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-nat-gateway",
	DescriptiveName: "NAT Gateway",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a NAT Gateway by ID",
		ListDescription:   "List all NAT gateways",
		SearchDescription: "Search for NAT gateways by ARN",
	},
	PotentialLinks: []string{"ec2-vpc", "ec2-subnet", "ec2-network-interface", "ip"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_nat_gateway.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
