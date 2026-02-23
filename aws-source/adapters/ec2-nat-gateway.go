package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
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
		attrs, err = ToAttributesWithExclude(ng, "tags")

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
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2NatGatewayAdapter(client *ec2.Client, accountID string, region string, cache sdpcache.Cache) *DescribeOnlyAdapter[*ec2.DescribeNatGatewaysInput, *ec2.DescribeNatGatewaysOutput, *ec2.Client, *ec2.Options] {
	return &DescribeOnlyAdapter[*ec2.DescribeNatGatewaysInput, *ec2.DescribeNatGatewaysOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-nat-gateway",
		AdapterMetadata: natGatewayAdapterMetadata,
		cache:        cache,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeNatGatewaysInput) (*ec2.DescribeNatGatewaysOutput, error) {
			return client.DescribeNatGateways(ctx, input)
		},
		InputMapperGet:  natGatewayInputMapperGet,
		InputMapperList: natGatewayInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeNatGatewaysInput) Paginator[*ec2.DescribeNatGatewaysOutput, *ec2.Options] {
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
