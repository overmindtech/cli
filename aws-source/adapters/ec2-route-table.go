package adapters

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func routeTableInputMapperGet(scope string, query string) (*ec2.DescribeRouteTablesInput, error) {
	return &ec2.DescribeRouteTablesInput{
		RouteTableIds: []string{
			query,
		},
	}, nil
}

func routeTableInputMapperList(scope string) (*ec2.DescribeRouteTablesInput, error) {
	return &ec2.DescribeRouteTablesInput{}, nil
}

func routeTableOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeRouteTablesInput, output *ec2.DescribeRouteTablesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, rt := range output.RouteTables {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(rt, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-route-table",
			UniqueAttribute: "RouteTableId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(rt.Tags),
		}

		for _, assoc := range rt.Associations {
			if assoc.SubnetId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-subnet",
						Method: sdp.QueryMethod_GET,
						Query:  *assoc.SubnetId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// All things in a route table could affect each other
						// since changing the target could affect the
						// traffic that is routed to it. And changing the route
						// table could affect the target
						In:  true,
						Out: true,
					},
				})
			}

			if assoc.GatewayId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-internet-gateway",
						Method: sdp.QueryMethod_GET,
						Query:  *assoc.GatewayId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		}

		for _, route := range rt.Routes {
			if route.GatewayId != nil {
				if strings.HasPrefix(*route.GatewayId, "igw") {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-internet-gateway",
							Method: sdp.QueryMethod_GET,
							Query:  *route.GatewayId,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					})
				}
				if strings.HasPrefix(*route.GatewayId, "vpce") {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ec2-vpc-endpoint",
							Method: sdp.QueryMethod_GET,
							Query:  *route.GatewayId,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					})
				}
			}
			if route.CarrierGatewayId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-carrier-gateway",
						Method: sdp.QueryMethod_GET,
						Query:  *route.CarrierGatewayId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
			if route.EgressOnlyInternetGatewayId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-egress-only-internet-gateway",
						Method: sdp.QueryMethod_GET,
						Query:  *route.EgressOnlyInternetGatewayId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
			if route.InstanceId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-instance",
						Method: sdp.QueryMethod_GET,
						Query:  *route.InstanceId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
			if route.LocalGatewayId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-local-gateway",
						Method: sdp.QueryMethod_GET,
						Query:  *route.LocalGatewayId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
			if route.NatGatewayId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-nat-gateway",
						Method: sdp.QueryMethod_GET,
						Query:  *route.NatGatewayId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
			if route.NetworkInterfaceId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-network-interface",
						Method: sdp.QueryMethod_GET,
						Query:  *route.NetworkInterfaceId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
			if route.TransitGatewayId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-transit-gateway",
						Method: sdp.QueryMethod_GET,
						Query:  *route.TransitGatewayId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
			if route.VpcPeeringConnectionId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-vpc-peering-connection",
						Method: sdp.QueryMethod_GET,
						Query:  *route.VpcPeeringConnectionId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		}

		if rt.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *rt.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2RouteTableAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeRouteTablesInput, *ec2.DescribeRouteTablesOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeRouteTablesInput, *ec2.DescribeRouteTablesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-route-table",
		AdapterMetadata: routeTableAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error) {
			return client.DescribeRouteTables(ctx, input)
		},
		InputMapperGet:  routeTableInputMapperGet,
		InputMapperList: routeTableInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeRouteTablesInput) adapterhelpers.Paginator[*ec2.DescribeRouteTablesOutput, *ec2.Options] {
			return ec2.NewDescribeRouteTablesPaginator(client, params)
		},
		OutputMapper: routeTableOutputMapper,
	}
}

var routeTableAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-route-table",
	DescriptiveName: "Route Table",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a route table by ID",
		ListDescription:   "List all route tables",
		SearchDescription: "Search route tables by ARN",
	},
	PotentialLinks: []string{"ec2-vpc", "ec2-subnet", "ec2-internet-gateway", "ec2-vpc-endpoint", "ec2-carrier-gateway", "ec2-egress-only-internet-gateway", "ec2-instance", "ec2-local-gateway", "ec2-nat-gateway", "ec2-network-interface", "ec2-transit-gateway", "ec2-vpc-peering-connection"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_route_table.id"},
		{TerraformQueryMap: "aws_route_table_association.route_table_id"},
		{TerraformQueryMap: "aws_default_route_table.default_route_table_id"},
		{TerraformQueryMap: "aws_route.route_table_id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
