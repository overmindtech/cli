package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

// APIs used:
//   - DescribeTransitGatewayRouteTables — list/describe transit gateway route tables.
//     https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGatewayRouteTables.html

func transitGatewayRouteTableInputMapperGet(scope string, query string) (*ec2.DescribeTransitGatewayRouteTablesInput, error) {
	return &ec2.DescribeTransitGatewayRouteTablesInput{
		TransitGatewayRouteTableIds: []string{
			query,
		},
	}, nil
}

func transitGatewayRouteTableInputMapperList(scope string) (*ec2.DescribeTransitGatewayRouteTablesInput, error) {
	return &ec2.DescribeTransitGatewayRouteTablesInput{}, nil
}

func transitGatewayRouteTableOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeTransitGatewayRouteTablesInput, output *ec2.DescribeTransitGatewayRouteTablesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, rt := range output.TransitGatewayRouteTables {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = ToAttributesWithExclude(rt, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-transit-gateway-route-table",
			UniqueAttribute: "TransitGatewayRouteTableId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(rt.Tags),
		}

		if rt.TransitGatewayId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  *rt.TransitGatewayId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		// Link to route table associations, propagations, and routes (Search by route table ID).
		if rt.TransitGatewayRouteTableId != nil {
			rtID := *rt.TransitGatewayRouteTableId
			for _, linkType := range []string{"ec2-transit-gateway-route-table-association", "ec2-transit-gateway-route-table-propagation", "ec2-transit-gateway-route"} {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   linkType,
						Method: sdp.QueryMethod_SEARCH,
						Query:  rtID,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2TransitGatewayRouteTableAdapter(client *ec2.Client, accountID string, region string, cache sdpcache.Cache) *DescribeOnlyAdapter[*ec2.DescribeTransitGatewayRouteTablesInput, *ec2.DescribeTransitGatewayRouteTablesOutput, *ec2.Client, *ec2.Options] {
	return &DescribeOnlyAdapter[*ec2.DescribeTransitGatewayRouteTablesInput, *ec2.DescribeTransitGatewayRouteTablesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-transit-gateway-route-table",
		AdapterMetadata: transitGatewayRouteTableAdapterMetadata,
		cache:           cache,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeTransitGatewayRouteTablesInput) (*ec2.DescribeTransitGatewayRouteTablesOutput, error) {
			return client.DescribeTransitGatewayRouteTables(ctx, input)
		},
		InputMapperGet:  transitGatewayRouteTableInputMapperGet,
		InputMapperList: transitGatewayRouteTableInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeTransitGatewayRouteTablesInput) Paginator[*ec2.DescribeTransitGatewayRouteTablesOutput, *ec2.Options] {
			return ec2.NewDescribeTransitGatewayRouteTablesPaginator(client, params)
		},
		OutputMapper: transitGatewayRouteTableOutputMapper,
	}
}

var transitGatewayRouteTableAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-transit-gateway-route-table",
	DescriptiveName: "Transit Gateway Route Table",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a transit gateway route table by ID",
		ListDescription:   "List all transit gateway route tables",
		SearchDescription: "Search transit gateway route tables by ARN",
	},
	PotentialLinks: []string{"ec2-transit-gateway", "ec2-transit-gateway-route-table-association", "ec2-transit-gateway-route-table-propagation", "ec2-transit-gateway-route"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_ec2_transit_gateway_route_table.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
