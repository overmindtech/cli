package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

// APIs used:
//   - DescribeTransitGatewayRouteTables — list route tables (to then fetch associations per table).
//     https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGatewayRouteTables.html
//   - GetTransitGatewayRouteTableAssociations — list associations for a route table.
//     https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_GetTransitGatewayRouteTableAssociations.html

// transitGatewayRouteTableAssociationItem holds an association plus its route table ID for unique identification.
type transitGatewayRouteTableAssociationItem struct {
	RouteTableID string
	Association  types.TransitGatewayRouteTableAssociation
}

const associationIDSep = "|"

func transitGatewayRouteTableAssociationID(routeTableID, attachmentID string) string {
	return routeTableID + associationIDSep + attachmentID
}

// parseCompositeID splits query by the given separator; accepts both `|` and `_` (Terraform uses `_`).
// Returns (left, right); empty left means invalid.
func parseCompositeID(query, sep string) (string, string) {
	parts := strings.SplitN(query, sep, 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

func parseAssociationQuery(query string) (routeTableID, attachmentID string, err error) {
	if a, b := parseCompositeID(query, associationIDSep); a != "" {
		return a, b, nil
	}
	if a, b := parseCompositeID(query, "_"); a != "" {
		return a, b, nil
	}
	return "", "", fmt.Errorf("query must be TransitGatewayRouteTableId|TransitGatewayAttachmentId")
}

func getTransitGatewayRouteTableAssociation(ctx context.Context, client *ec2.Client, _, query string) (*transitGatewayRouteTableAssociationItem, error) {
	routeTableID, attachmentID, err := parseAssociationQuery(query)
	if err != nil {
		return nil, err
	}
	pg := ec2.NewGetTransitGatewayRouteTableAssociationsPaginator(client, &ec2.GetTransitGatewayRouteTableAssociationsInput{
		TransitGatewayRouteTableId: &routeTableID,
	})
	for pg.HasMorePages() {
		out, err := pg.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for i := range out.Associations {
			a := &out.Associations[i]
			if a.TransitGatewayAttachmentId != nil && *a.TransitGatewayAttachmentId == attachmentID {
				return &transitGatewayRouteTableAssociationItem{RouteTableID: routeTableID, Association: *a}, nil
			}
		}
	}
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: fmt.Sprintf("association %s not found", query),
	}
}

func listTransitGatewayRouteTableAssociations(ctx context.Context, client *ec2.Client, _ string) ([]*transitGatewayRouteTableAssociationItem, error) {
	// List all route tables, then get associations for each.
	rtPaginator := ec2.NewDescribeTransitGatewayRouteTablesPaginator(client, &ec2.DescribeTransitGatewayRouteTablesInput{})
	var items []*transitGatewayRouteTableAssociationItem
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
			assocPaginator := ec2.NewGetTransitGatewayRouteTableAssociationsPaginator(client, &ec2.GetTransitGatewayRouteTableAssociationsInput{
				TransitGatewayRouteTableId: &rtID,
			})
			for assocPaginator.HasMorePages() {
				assocOut, err := assocPaginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for i := range assocOut.Associations {
					items = append(items, &transitGatewayRouteTableAssociationItem{
						RouteTableID: rtID,
						Association:  assocOut.Associations[i],
					})
				}
			}
		}
	}
	return items, nil
}

// searchTransitGatewayRouteTableAssociations returns all associations for a single route table.
// Query must be a TransitGatewayRouteTableId (e.g. tgw-rtb-xxxxx).
func searchTransitGatewayRouteTableAssociations(ctx context.Context, client *ec2.Client, _, query string) ([]*transitGatewayRouteTableAssociationItem, error) {
	routeTableID := query
	var items []*transitGatewayRouteTableAssociationItem
	pg := ec2.NewGetTransitGatewayRouteTableAssociationsPaginator(client, &ec2.GetTransitGatewayRouteTableAssociationsInput{
		TransitGatewayRouteTableId: &routeTableID,
	})
	for pg.HasMorePages() {
		out, err := pg.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for i := range out.Associations {
			items = append(items, &transitGatewayRouteTableAssociationItem{
				RouteTableID: routeTableID,
				Association:  out.Associations[i],
			})
		}
	}
	return items, nil
}

func transitGatewayRouteTableAssociationItemMapper(query, scope string, awsItem *transitGatewayRouteTableAssociationItem) (*sdp.Item, error) {
	a := &awsItem.Association
	attrs, err := ToAttributesWithExclude(a, "")
	if err != nil {
		return nil, err
	}
	attachmentID := ""
	if a.TransitGatewayAttachmentId != nil {
		attachmentID = *a.TransitGatewayAttachmentId
	}
	uniqueVal := transitGatewayRouteTableAssociationID(awsItem.RouteTableID, attachmentID)
	if err := attrs.Set("TransitGatewayRouteTableIdWithTransitGatewayAttachmentId", uniqueVal); err != nil {
		return nil, err
	}
	item := &sdp.Item{
		Type:            "ec2-transit-gateway-route-table-association",
		UniqueAttribute: "TransitGatewayRouteTableIdWithTransitGatewayAttachmentId",
		Scope:           scope,
		Attributes:      attrs,
	}
	// Link to route table
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "ec2-transit-gateway-route-table",
			Method: sdp.QueryMethod_GET,
			Query:  awsItem.RouteTableID,
			Scope:  scope,
		},
	})
	if a.TransitGatewayAttachmentId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ec2-transit-gateway-attachment",
				Method: sdp.QueryMethod_GET,
				Query:  *a.TransitGatewayAttachmentId,
				Scope:  scope,
			},
		})
	}
	if a.ResourceId != nil && *a.ResourceId != "" {
		switch a.ResourceType {
		case types.TransitGatewayAttachmentResourceTypeVpc:
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *a.ResourceId,
					Scope:  scope,
				},
			})
		case types.TransitGatewayAttachmentResourceTypeVpn:
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpn-connection",
					Method: sdp.QueryMethod_GET,
					Query:  *a.ResourceId,
					Scope:  scope,
				},
			})
		case types.TransitGatewayAttachmentResourceTypeDirectConnectGateway:
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-direct-connect-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  *a.ResourceId,
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
					Query:  *a.ResourceId,
					Scope:  scope,
				},
			})
		case types.TransitGatewayAttachmentResourceTypeVpnConcentrator,
			types.TransitGatewayAttachmentResourceTypeConnect,
			types.TransitGatewayAttachmentResourceTypeNetworkFunction:
			// No Overmind adapter for these resource types; attachment link above is sufficient.
		}
	}
	return item, nil
}

func NewEC2TransitGatewayRouteTableAssociationAdapter(client *ec2.Client, accountID, region string, cache sdpcache.Cache) *GetListAdapter[*transitGatewayRouteTableAssociationItem, *ec2.Client, *ec2.Options] {
	return &GetListAdapter[*transitGatewayRouteTableAssociationItem, *ec2.Client, *ec2.Options]{
		ItemType:        "ec2-transit-gateway-route-table-association",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: transitGatewayRouteTableAssociationAdapterMetadata,
		cache:           cache,
		GetFunc:         getTransitGatewayRouteTableAssociation,
		ListFunc:        listTransitGatewayRouteTableAssociations,
		SearchFunc:      searchTransitGatewayRouteTableAssociations,
		ItemMapper:      transitGatewayRouteTableAssociationItemMapper,
	}
}

var transitGatewayRouteTableAssociationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-transit-gateway-route-table-association",
	DescriptiveName: "Transit Gateway Route Table Association",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:             true,
		List:            true,
		Search:          true,
		GetDescription:  "Get by TransitGatewayRouteTableId|TransitGatewayAttachmentId",
		ListDescription: "List all route table associations",
		SearchDescription: "Search by TransitGatewayRouteTableId to list associations for that route table",
	},
	PotentialLinks: []string{"ec2-transit-gateway", "ec2-transit-gateway-route-table", "ec2-transit-gateway-attachment", "ec2-vpc", "ec2-vpn-connection", "directconnect-direct-connect-gateway"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_ec2_transit_gateway_route_table_association.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
