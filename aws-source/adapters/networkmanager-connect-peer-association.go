package adapters

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func connectPeerAssociationsOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, _ *networkmanager.GetConnectPeerAssociationsInput, output *networkmanager.GetConnectPeerAssociationsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, a := range output.ConnectPeerAssociations {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(a)

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		if a.GlobalNetworkId == nil || a.ConnectPeerId == nil {
			return nil, sdp.NewQueryError(errors.New("globalNetworkId or connectPeerId is nil for connect peer association"))
		}

		attrs.Set("GlobalNetworkIdConnectPeerId", idWithGlobalNetwork(*a.GlobalNetworkId, *a.ConnectPeerId))

		item := sdp.Item{
			Type:            "networkmanager-connect-peer-association",
			UniqueAttribute: "GlobalNetworkIdConnectPeerId",
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
				{
					Query: &sdp.Query{
						Type:   "networkmanager-connect-peer",
						Method: sdp.QueryMethod_GET,
						Query:  *a.ConnectPeerId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
		}

		switch a.State {
		case types.ConnectPeerAssociationStatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.ConnectPeerAssociationStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.ConnectPeerAssociationStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.ConnectPeerAssociationStateDeleted:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}

		if a.DeviceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-device",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*a.GlobalNetworkId, *a.DeviceId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}

		if a.LinkId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-link",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*a.GlobalNetworkId, *a.LinkId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerConnectPeerAssociationAdapter(client *networkmanager.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetConnectPeerAssociationsInput, *networkmanager.GetConnectPeerAssociationsOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetConnectPeerAssociationsInput, *networkmanager.GetConnectPeerAssociationsOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "networkmanager-connect-peer-association",
		AdapterMetadata: connectPeerAssociationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetConnectPeerAssociationsInput) (*networkmanager.GetConnectPeerAssociationsOutput, error) {
			return client.GetConnectPeerAssociations(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*networkmanager.GetConnectPeerAssociationsInput, error) {
			// We are using a custom id of {globalNetworkId}|{connectPeerId}
			sections := strings.Split(query, "|")

			if len(sections) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-connect-peer-association get function",
				}
			}
			return &networkmanager.GetConnectPeerAssociationsInput{
				GlobalNetworkId: &sections[0],
				ConnectPeerIds: []string{
					sections[1],
				},
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.GetConnectPeerAssociationsInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-connect-peer-association, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetConnectPeerAssociationsInput) adapterhelpers.Paginator[*networkmanager.GetConnectPeerAssociationsOutput, *networkmanager.Options] {
			return networkmanager.NewGetConnectPeerAssociationsPaginator(client, params)
		},
		OutputMapper: connectPeerAssociationsOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetConnectPeerAssociationsInput, error) {
			// Search by GlobalNetworkId
			return &networkmanager.GetConnectPeerAssociationsInput{
				GlobalNetworkId: &query,
			}, nil
		},
	}
}

var connectPeerAssociationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-connect-peer-association",
	DescriptiveName: "Networkmanager Connect Peer Association",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a Networkmanager Connect Peer Association",
		ListDescription:   "List all Networkmanager Connect Peer Associations",
		SearchDescription: "Search for Networkmanager ConnectPeerAssociations by GlobalNetworkId",
	},
	PotentialLinks: []string{"networkmanager-global-network", "networkmanager-connect-peer", "networkmanager-device", "networkmanager-link"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
