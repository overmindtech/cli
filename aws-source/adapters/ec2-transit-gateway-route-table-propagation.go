package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

// APIs used:
//   - DescribeTransitGatewayRouteTables — list route tables (to then fetch propagations per table).
//     https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGatewayRouteTables.html
//   - GetTransitGatewayRouteTablePropagations — list propagations for a route table.
//     https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_GetTransitGatewayRouteTablePropagations.html

type transitGatewayRouteTablePropagationItem struct {
	RouteTableID string
	Propagation  types.TransitGatewayRouteTablePropagation
}

const propagationIDSep = "|"

func transitGatewayRouteTablePropagationID(routeTableID, attachmentID string) string {
	return routeTableID + propagationIDSep + attachmentID
}

func parsePropagationQuery(query string) (routeTableID, attachmentID string, err error) {
	if a, b := parseCompositeID(query, propagationIDSep); a != "" {
		return a, b, nil
	}
	if a, b := parseCompositeID(query, "_"); a != "" {
		return a, b, nil
	}
	return "", "", fmt.Errorf("query must be TransitGatewayRouteTableId|TransitGatewayAttachmentId")
}

func getTransitGatewayRouteTablePropagation(ctx context.Context, client *ec2.Client, _, query string) (*transitGatewayRouteTablePropagationItem, error) {
	routeTableID, attachmentID, err := parsePropagationQuery(query)
	if err != nil {
		return nil, err
	}
	pg := ec2.NewGetTransitGatewayRouteTablePropagationsPaginator(client, &ec2.GetTransitGatewayRouteTablePropagationsInput{
		TransitGatewayRouteTableId: &routeTableID,
	})
	for pg.HasMorePages() {
		out, err := pg.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for i := range out.TransitGatewayRouteTablePropagations {
			p := &out.TransitGatewayRouteTablePropagations[i]
			if p.TransitGatewayAttachmentId != nil && *p.TransitGatewayAttachmentId == attachmentID {
				return &transitGatewayRouteTablePropagationItem{RouteTableID: routeTableID, Propagation: *p}, nil
			}
		}
	}
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: fmt.Sprintf("propagation %s not found", query),
	}
}

func listTransitGatewayRouteTablePropagations(ctx context.Context, client *ec2.Client, _ string) ([]*transitGatewayRouteTablePropagationItem, error) {
	rtPaginator := ec2.NewDescribeTransitGatewayRouteTablesPaginator(client, &ec2.DescribeTransitGatewayRouteTablesInput{})
	var items []*transitGatewayRouteTablePropagationItem
	for rtPaginator.HasMorePages() {
		rtOut, err := rtPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rt := range rtOut.TransitGatewayRouteTables {
			if rt.TransitGatewayRouteTableId == nil {
				continue
			}
			rtID := *rt.TransitGatewayRouteTableId
			propPaginator := ec2.NewGetTransitGatewayRouteTablePropagationsPaginator(client, &ec2.GetTransitGatewayRouteTablePropagationsInput{
				TransitGatewayRouteTableId: &rtID,
			})
			for propPaginator.HasMorePages() {
				propOut, err := propPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for i := range propOut.TransitGatewayRouteTablePropagations {
					items = append(items, &transitGatewayRouteTablePropagationItem{
						RouteTableID: rtID,
						Propagation:  propOut.TransitGatewayRouteTablePropagations[i],
					})
				}
			}
		}
	}
	return items, nil
}

// searchTransitGatewayRouteTablePropagations returns all propagations for a single route table.
// Query must be a TransitGatewayRouteTableId (e.g. tgw-rtb-xxxxx).
func searchTransitGatewayRouteTablePropagations(ctx context.Context, client *ec2.Client, _, query string) ([]*transitGatewayRouteTablePropagationItem, error) {
	routeTableID := query
	var items []*transitGatewayRouteTablePropagationItem
	pg := ec2.NewGetTransitGatewayRouteTablePropagationsPaginator(client, &ec2.GetTransitGatewayRouteTablePropagationsInput{
		TransitGatewayRouteTableId: &routeTableID,
	})
	for pg.HasMorePages() {
		out, err := pg.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for i := range out.TransitGatewayRouteTablePropagations {
			items = append(items, &transitGatewayRouteTablePropagationItem{
				RouteTableID: routeTableID,
				Propagation:  out.TransitGatewayRouteTablePropagations[i],
			})
		}
	}
	return items, nil
}

func transitGatewayRouteTablePropagationItemMapper(query, scope string, awsItem *transitGatewayRouteTablePropagationItem) (*sdp.Item, error) {
	p := &awsItem.Propagation
	attrs, err := ToAttributesWithExclude(p, "")
	if err != nil {
		return nil, err
	}
	attachmentID := ""
	if p.TransitGatewayAttachmentId != nil {
		attachmentID = *p.TransitGatewayAttachmentId
	}
	uniqueVal := transitGatewayRouteTablePropagationID(awsItem.RouteTableID, attachmentID)
	if err := attrs.Set("TransitGatewayRouteTableIdWithTransitGatewayAttachmentId", uniqueVal); err != nil {
		return nil, err
	}
	item := &sdp.Item{
		Type:            "ec2-transit-gateway-route-table-propagation",
		UniqueAttribute: "TransitGatewayRouteTableIdWithTransitGatewayAttachmentId",
		Scope:           scope,
		Attributes:      attrs,
	}
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "ec2-transit-gateway-route-table",
			Method: sdp.QueryMethod_GET,
			Query:  awsItem.RouteTableID,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
	})
	// Link to the route table association (same route table + attachment).
	if p.TransitGatewayAttachmentId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ec2-transit-gateway-route-table-association",
				Method: sdp.QueryMethod_GET,
				Query:  transitGatewayRouteTableAssociationID(awsItem.RouteTableID, *p.TransitGatewayAttachmentId),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		})
	}
	if p.TransitGatewayAttachmentId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ec2-transit-gateway-attachment",
				Method: sdp.QueryMethod_GET,
				Query:  *p.TransitGatewayAttachmentId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		})
	}
	if p.ResourceId != nil && *p.ResourceId != "" {
		switch p.ResourceType {
		case types.TransitGatewayAttachmentResourceTypeVpc:
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *p.ResourceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		case types.TransitGatewayAttachmentResourceTypeVpn:
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpn-connection",
					Method: sdp.QueryMethod_GET,
					Query:  *p.ResourceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		case types.TransitGatewayAttachmentResourceTypeDirectConnectGateway:
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-direct-connect-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  *p.ResourceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		case types.TransitGatewayAttachmentResourceTypePeering,
			types.TransitGatewayAttachmentResourceTypeTgwPeering:
			// ResourceId is the peer transit gateway ID (e.g. tgw-xxxxx).
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  *p.ResourceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		case types.TransitGatewayAttachmentResourceTypeVpnConcentrator,
			types.TransitGatewayAttachmentResourceTypeConnect,
			types.TransitGatewayAttachmentResourceTypeNetworkFunction:
			// No Overmind adapter for these resource types; attachment link above is sufficient.
		}
	}
	return item, nil
}

func NewEC2TransitGatewayRouteTablePropagationAdapter(client *ec2.Client, accountID, region string, cache sdpcache.Cache) *GetListAdapter[*transitGatewayRouteTablePropagationItem, *ec2.Client, *ec2.Options] {
	return &GetListAdapter[*transitGatewayRouteTablePropagationItem, *ec2.Client, *ec2.Options]{
		ItemType:        "ec2-transit-gateway-route-table-propagation",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: transitGatewayRouteTablePropagationAdapterMetadata,
		cache:           cache,
		GetFunc:         getTransitGatewayRouteTablePropagation,
		ListFunc:        listTransitGatewayRouteTablePropagations,
		SearchFunc:      searchTransitGatewayRouteTablePropagations,
		ItemMapper:      transitGatewayRouteTablePropagationItemMapper,
	}
}

var transitGatewayRouteTablePropagationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-transit-gateway-route-table-propagation",
	DescriptiveName: "Transit Gateway Route Table Propagation",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:             true,
		List:            true,
		Search:          true,
		GetDescription:  "Get by TransitGatewayRouteTableId|TransitGatewayAttachmentId",
		ListDescription: "List all route table propagations",
		SearchDescription: "Search by TransitGatewayRouteTableId to list propagations for that route table",
	},
	PotentialLinks: []string{"ec2-transit-gateway", "ec2-transit-gateway-route-table", "ec2-transit-gateway-route-table-association", "ec2-transit-gateway-attachment", "ec2-vpc", "ec2-vpn-connection", "directconnect-direct-connect-gateway"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_ec2_transit_gateway_route_table_propagation.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
