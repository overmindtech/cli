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

func connectionOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, _ *networkmanager.GetConnectionsInput, output *networkmanager.GetConnectionsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, s := range output.Connections {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(s, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		if s.GlobalNetworkId == nil || s.ConnectionId == nil {
			return nil, sdp.NewQueryError(errors.New("globalNetworkId or connectionId is nil for connection"))
		}

		attrs.Set("GlobalNetworkIdConnectionId", idWithGlobalNetwork(*s.GlobalNetworkId, *s.ConnectionId))

		item := sdp.Item{
			Type:            "networkmanager-connection",
			UniqueAttribute: "GlobalNetworkIdConnectionId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            networkmanagerTagsToMap(s.Tags),
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "networkmanager-global-network",
						Method: sdp.QueryMethod_GET,
						Query:  *s.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
		}

		if s.LinkId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-link",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.LinkId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		if s.ConnectedLinkId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-link",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.ConnectedLinkId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		if s.DeviceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-device",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.DeviceId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		if s.ConnectedDeviceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-device",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.ConnectedDeviceId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		switch s.State {
		case types.ConnectionStatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.ConnectionStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.ConnectionStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.ConnectionStateUpdating:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerConnectionAdapter(client *networkmanager.Client, accountID string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetConnectionsInput, *networkmanager.GetConnectionsOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetConnectionsInput, *networkmanager.GetConnectionsOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:    client,
		AccountID: accountID,
		ItemType:  "networkmanager-connection",
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetConnectionsInput) (*networkmanager.GetConnectionsOutput, error) {
			return client.GetConnections(ctx, input)
		},
		AdapterMetadata: networkmanagerConnectionAdapterMetadata,
		InputMapperGet: func(scope, query string) (*networkmanager.GetConnectionsInput, error) {
			// We are using a custom id of {globalNetworkId}|{connectionId}
			sections := strings.Split(query, "|")

			if len(sections) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-connection get function",
				}
			}
			return &networkmanager.GetConnectionsInput{
				GlobalNetworkId: &sections[0],
				ConnectionIds: []string{
					sections[1],
				},
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.GetConnectionsInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-connection, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetConnectionsInput) adapterhelpers.Paginator[*networkmanager.GetConnectionsOutput, *networkmanager.Options] {
			return networkmanager.NewGetConnectionsPaginator(client, params)
		},
		OutputMapper: connectionOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetConnectionsInput, error) {
			// Try to parse as ARN first
			arn, err := adapterhelpers.ParseARN(query)
			if err == nil {
				// Check if it's a networkmanager ARN
				if arn.Service == "networkmanager" {
					switch arn.Type() {
					case "device":
						// Parse the resource part which can be:
						// 1. device/global-network-{id}/device-{id} (for device ARNs)
						// 2. device/global-network-{id}/connection-{id} (for connection ARNs)
						resourceParts := strings.Split(arn.Resource, "/")
						if len(resourceParts) == 3 && resourceParts[0] == "device" && strings.HasPrefix(resourceParts[1], "global-network-") {
							globalNetworkId := resourceParts[1] // Keep full ID including "global-network-" prefix

							if strings.HasPrefix(resourceParts[2], "connection-") {
								// This is a connection ARN: device/global-network-{id}/connection-{id}
								connectionId := resourceParts[2] // Keep full ID including "connection-" prefix

								return &networkmanager.GetConnectionsInput{
									GlobalNetworkId: &globalNetworkId,
									ConnectionIds:   []string{connectionId},
								}, nil
							} else if strings.HasPrefix(resourceParts[2], "device-") {
								// This is a device ARN: device/global-network-{id}/device-{id}
								deviceId := resourceParts[2] // Keep full ID including "device-" prefix

								return &networkmanager.GetConnectionsInput{
									GlobalNetworkId: &globalNetworkId,
									DeviceId:        &deviceId,
								}, nil
							}
						}
					}
				}

				// If it's not a valid networkmanager ARN, return an error
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "ARN is not a valid networkmanager-connection or networkmanager-device ARN",
				}
			}

			// If not an ARN, fall back to the original logic
			// We may search by only globalNetworkId or by using a custom id of {globalNetworkId}|{deviceId}
			sections := strings.Split(query, "|")
			switch len(sections) {
			case 1:
				// globalNetworkId
				return &networkmanager.GetConnectionsInput{
					GlobalNetworkId: &sections[0],
				}, nil
			case 2:
				// {globalNetworkId}|{deviceId}
				return &networkmanager.GetConnectionsInput{
					GlobalNetworkId: &sections[0],
					DeviceId:        &sections[1],
				}, nil
			default:
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-connection get function",
				}
			}
		},
	}
}

var networkmanagerConnectionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-connection",
	DescriptiveName: "Networkmanager Connection",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a Networkmanager Connection",
		SearchDescription: "Search for Networkmanager Connections by GlobalNetworkId, Device ARN, or Connection ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_networkmanager_connection.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"networkmanager-global-network", "networkmanager-link", "networkmanager-device"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
