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
//   - DescribeTransitGatewayRouteTables — list route tables (to then search routes per table).
//     https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGatewayRouteTables.html
//   - SearchTransitGatewayRoutes — search routes in a route table.
//     https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SearchTransitGatewayRoutes.html
//
// Note: SearchTransitGatewayRoutes does not support NextToken-based pagination. It returns
// at most 1000 routes per call; AdditionalRoutesAvailable indicates more exist but there is
// no API mechanism to fetch them (route tables can hold up to 10,000 routes).

type transitGatewayRouteItem struct {
	RouteTableID string
	Route        types.TransitGatewayRoute
}

const routeIDSep = "|"
const routeDestPrefixList = "pl:"

func transitGatewayRouteDestination(r *types.TransitGatewayRoute) string {
	if r.PrefixListId != nil && *r.PrefixListId != "" {
		return routeDestPrefixList + *r.PrefixListId
	}
	if r.DestinationCidrBlock != nil {
		return *r.DestinationCidrBlock
	}
	return ""
}

func transitGatewayRouteID(routeTableID, destination string) string {
	return routeTableID + routeIDSep + destination
}

func parseRouteQuery(query string) (routeTableID, destination string, err error) {
	if a, b := parseCompositeID(query, routeIDSep); a != "" {
		return a, b, nil
	}
	if a, b := parseCompositeID(query, "_"); a != "" {
		return a, b, nil
	}
	return "", "", fmt.Errorf("query must be TransitGatewayRouteTableId|Destination (CIDR or pl:PrefixListId)")
}

// searchRoutesFilter returns a filter that returns all routes (active and blackhole).
func searchRoutesFilter() []types.Filter {
	return []types.Filter{
		{Name: new("state"), Values: []string{"active", "blackhole"}},
	}
}

// maxSearchRoutesResults is the maximum routes SearchTransitGatewayRoutes returns per call.
// The API does not support NextToken pagination when AdditionalRoutesAvailable is true.
const maxSearchRoutesResults = 1000

func getTransitGatewayRoute(ctx context.Context, client *ec2.Client, _, query string) (*transitGatewayRouteItem, error) {
	routeTableID, destination, err := parseRouteQuery(query)
	if err != nil {
		return nil, err
	}
	out, err := client.SearchTransitGatewayRoutes(ctx, &ec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: &routeTableID,
		Filters:                    searchRoutesFilter(),
		MaxResults:                 new(int32(maxSearchRoutesResults)),
	})
	if err != nil {
		return nil, err
	}
	for i := range out.Routes {
		r := &out.Routes[i]
		if transitGatewayRouteDestination(r) == destination {
			return &transitGatewayRouteItem{RouteTableID: routeTableID, Route: *r}, nil
		}
	}
	errStr := fmt.Sprintf("route %s not found", query)
	if out.AdditionalRoutesAvailable != nil && *out.AdditionalRoutesAvailable {
		errStr = fmt.Sprintf("route %s not found in first %d routes; route table has additional routes that cannot be retrieved via this API", query, maxSearchRoutesResults)
	}
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: errStr,
	}
}

func listTransitGatewayRoutes(ctx context.Context, client *ec2.Client, _ string) ([]*transitGatewayRouteItem, error) {
	rtPaginator := ec2.NewDescribeTransitGatewayRouteTablesPaginator(client, &ec2.DescribeTransitGatewayRouteTablesInput{})
	var items []*transitGatewayRouteItem
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
			// Single call per route table: SearchTransitGatewayRoutes returns at most 1000 routes
			// and does not support NextToken pagination; AdditionalRoutesAvailable means more
			// exist but cannot be fetched via this API.
			routeOut, err := client.SearchTransitGatewayRoutes(ctx, &ec2.SearchTransitGatewayRoutesInput{
				TransitGatewayRouteTableId: &rtID,
				Filters:                    searchRoutesFilter(),
				MaxResults:                 new(int32(maxSearchRoutesResults)),
			})
			if err != nil {
				return nil, err
			}
			for i := range routeOut.Routes {
				items = append(items, &transitGatewayRouteItem{
					RouteTableID: rtID,
					Route:        routeOut.Routes[i],
				})
			}
		}
	}
	return items, nil
}

// searchTransitGatewayRoutes returns all routes for a single route table.
// Query must be a TransitGatewayRouteTableId (e.g. tgw-rtb-xxxxx).
func searchTransitGatewayRoutes(ctx context.Context, client *ec2.Client, _, query string) ([]*transitGatewayRouteItem, error) {
	routeTableID := query
	routeOut, err := client.SearchTransitGatewayRoutes(ctx, &ec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: &routeTableID,
		Filters:                    searchRoutesFilter(),
		MaxResults:                 new(int32(maxSearchRoutesResults)),
	})
	if err != nil {
		return nil, err
	}
	items := make([]*transitGatewayRouteItem, 0, len(routeOut.Routes))
	for i := range routeOut.Routes {
		items = append(items, &transitGatewayRouteItem{
			RouteTableID: routeTableID,
			Route:        routeOut.Routes[i],
		})
	}
	return items, nil
}

func transitGatewayRouteItemMapper(query, scope string, awsItem *transitGatewayRouteItem) (*sdp.Item, error) {
	r := &awsItem.Route
	attrs, err := ToAttributesWithExclude(r, "")
	if err != nil {
		return nil, err
	}
	dest := transitGatewayRouteDestination(r)
	uniqueVal := transitGatewayRouteID(awsItem.RouteTableID, dest)
	if err := attrs.Set("TransitGatewayRouteTableIdWithDestination", uniqueVal); err != nil {
		return nil, err
	}
	item := &sdp.Item{
		Type:            "ec2-transit-gateway-route",
		UniqueAttribute: "TransitGatewayRouteTableIdWithDestination",
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
	})
	for i := range r.TransitGatewayAttachments {
		att := &r.TransitGatewayAttachments[i]
		if att.TransitGatewayAttachmentId != nil && *att.TransitGatewayAttachmentId != "" {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway-attachment",
					Method: sdp.QueryMethod_GET,
					Query:  *att.TransitGatewayAttachmentId,
					Scope:  scope,
				},
			})
			// Link to the route table association (same route table + attachment).
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway-route-table-association",
					Method: sdp.QueryMethod_GET,
					Query:  transitGatewayRouteTableAssociationID(awsItem.RouteTableID, *att.TransitGatewayAttachmentId),
					Scope:  scope,
				},
			})
		}
		if att.ResourceId != nil && *att.ResourceId != "" {
			switch att.ResourceType {
			case types.TransitGatewayAttachmentResourceTypeVpc:
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-vpc",
						Method: sdp.QueryMethod_GET,
						Query:  *att.ResourceId,
						Scope:  scope,
					},
				})
			case types.TransitGatewayAttachmentResourceTypeVpn:
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-vpn-connection",
						Method: sdp.QueryMethod_GET,
						Query:  *att.ResourceId,
						Scope:  scope,
					},
				})
			case types.TransitGatewayAttachmentResourceTypeDirectConnectGateway:
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "directconnect-direct-connect-gateway",
						Method: sdp.QueryMethod_GET,
						Query:  *att.ResourceId,
						Scope:  scope,
					},
				})
			case types.TransitGatewayAttachmentResourceTypePeering,
				types.TransitGatewayAttachmentResourceTypeTgwPeering:
				// ResourceId is the peer transit gateway ID (e.g. tgw-xxxxx).
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-transit-gateway",
						Method: sdp.QueryMethod_GET,
						Query:  *att.ResourceId,
						Scope:  scope,
					},
				})
			case types.TransitGatewayAttachmentResourceTypeVpnConcentrator,
				types.TransitGatewayAttachmentResourceTypeConnect,
				types.TransitGatewayAttachmentResourceTypeNetworkFunction:
				// No Overmind adapter for these; attachment link above is sufficient.
			}
		}
	}
	if r.PrefixListId != nil && *r.PrefixListId != "" {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ec2-managed-prefix-list",
				Method: sdp.QueryMethod_GET,
				Query:  *r.PrefixListId,
				Scope:  scope,
			},
		})
	}
	if r.TransitGatewayRouteTableAnnouncementId != nil && *r.TransitGatewayRouteTableAnnouncementId != "" {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ec2-transit-gateway-route-table-announcement",
				Method: sdp.QueryMethod_GET,
				Query:  *r.TransitGatewayRouteTableAnnouncementId,
				Scope:  scope,
			},
		})
	}
	return item, nil
}

func NewEC2TransitGatewayRouteAdapter(client *ec2.Client, accountID, region string, cache sdpcache.Cache) *GetListAdapter[*transitGatewayRouteItem, *ec2.Client, *ec2.Options] {
	return &GetListAdapter[*transitGatewayRouteItem, *ec2.Client, *ec2.Options]{
		ItemType:        "ec2-transit-gateway-route",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: transitGatewayRouteAdapterMetadata,
		cache:           cache,
		GetFunc:         getTransitGatewayRoute,
		ListFunc:        listTransitGatewayRoutes,
		SearchFunc:      searchTransitGatewayRoutes,
		ItemMapper:      transitGatewayRouteItemMapper,
	}
}

var transitGatewayRouteAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-transit-gateway-route",
	DescriptiveName: "Transit Gateway Route",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get by TransitGatewayRouteTableId|Destination (CIDR or pl:PrefixListId)",
		ListDescription:   "List all transit gateway routes",
		SearchDescription: "Search by TransitGatewayRouteTableId to list routes for that route table",
	},
	PotentialLinks: []string{"ec2-transit-gateway", "ec2-transit-gateway-route-table", "ec2-transit-gateway-route-table-association", "ec2-transit-gateway-attachment", "ec2-transit-gateway-route-table-announcement", "ec2-vpc", "ec2-vpn-connection", "ec2-managed-prefix-list", "directconnect-direct-connect-gateway"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_ec2_transit_gateway_route.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
