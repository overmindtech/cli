package adapters

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func transitGatewayConnectPeerAssociationsOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, _ *networkmanager.GetTransitGatewayConnectPeerAssociationsInput, output *networkmanager.GetTransitGatewayConnectPeerAssociationsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, a := range output.TransitGatewayConnectPeerAssociations {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(a, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		attrs.Set("GlobalNetworkIdWithTransitGatewayConnectPeerArn", idWithGlobalNetwork(*a.GlobalNetworkId, *a.TransitGatewayConnectPeerArn))

		item := sdp.Item{
			Type:            "networkmanager-transit-gateway-connect-peer-association",
			UniqueAttribute: "GlobalNetworkIdWithTransitGatewayConnectPeerArn",
			Scope:           scope,
			Attributes:      attrs,
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "networkmanager-global-network",
						Method: sdp.QueryMethod_GET,
						Query:  *a.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
		}

		if a.DeviceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-device",
					Method: sdp.QueryMethod_SEARCH,
					Query:  idWithGlobalNetwork(*a.GlobalNetworkId, *a.DeviceId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		if a.LinkId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-link",
					Method: sdp.QueryMethod_SEARCH,
					Query:  idWithGlobalNetwork(*a.GlobalNetworkId, *a.LinkId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		switch a.State {
		case types.TransitGatewayConnectPeerAssociationStatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.TransitGatewayConnectPeerAssociationStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.TransitGatewayConnectPeerAssociationStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.TransitGatewayConnectPeerAssociationStateDeleted:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerTransitGatewayConnectPeerAssociationAdapter(client *networkmanager.Client, accountID, region string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetTransitGatewayConnectPeerAssociationsInput, *networkmanager.GetTransitGatewayConnectPeerAssociationsOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetTransitGatewayConnectPeerAssociationsInput, *networkmanager.GetTransitGatewayConnectPeerAssociationsOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "networkmanager-transit-gateway-connect-peer-association",
		AdapterMetadata: transitGatewayConnectPeerAssociationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetTransitGatewayConnectPeerAssociationsInput) (*networkmanager.GetTransitGatewayConnectPeerAssociationsOutput, error) {
			return client.GetTransitGatewayConnectPeerAssociations(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*networkmanager.GetTransitGatewayConnectPeerAssociationsInput, error) {
			sections := strings.Split(query, "|")

			if len(sections) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-transit-gateway-connect-peer-association. Use {GlobalNetworkId}|{TransitGatewayConnectPeerArn} format",
				}
			}

			// we are using a custom id of {globalNetworkId}|{networkmanager-connect-peer.ID}
			// e.g. searching from networkmanager-connect-peer
			return &networkmanager.GetTransitGatewayConnectPeerAssociationsInput{
				GlobalNetworkId: &sections[0],
				TransitGatewayConnectPeerArns: []string{
					sections[1],
				},
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.GetTransitGatewayConnectPeerAssociationsInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: "list not supported for networkmanager-transit-gateway-connect-peer-association, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetTransitGatewayConnectPeerAssociationsInput) adapterhelpers.Paginator[*networkmanager.GetTransitGatewayConnectPeerAssociationsOutput, *networkmanager.Options] {
			return networkmanager.NewGetTransitGatewayConnectPeerAssociationsPaginator(client, params)
		},
		OutputMapper: transitGatewayConnectPeerAssociationsOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetTransitGatewayConnectPeerAssociationsInput, error) {
			// Search by GlobalNetworkId
			return &networkmanager.GetTransitGatewayConnectPeerAssociationsInput{
				GlobalNetworkId: &query,
			}, nil
		},
	}
}

var transitGatewayConnectPeerAssociationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-transit-gateway-connect-peer-association",
	DescriptiveName: "Networkmanager Transit Gateway Connect Peer Association",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a Networkmanager Transit Gateway Connect Peer Association by id",
		ListDescription:   "List all Networkmanager Transit Gateway Connect Peer Associations",
		SearchDescription: "Search for Networkmanager Transit Gateway Connect Peer Associations by GlobalNetworkId",
	},
	PotentialLinks: []string{"networkmanager-global-network", "networkmanager-device", "networkmanager-link"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
