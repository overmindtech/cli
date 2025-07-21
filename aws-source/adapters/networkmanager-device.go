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

func deviceOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, _ *networkmanager.GetDevicesInput, output *networkmanager.GetDevicesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, s := range output.Devices {
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

		if s.GlobalNetworkId == nil || s.DeviceId == nil {
			return nil, sdp.NewQueryError(errors.New("globalNetworkId or deviceId is nil for device"))
		}

		attrs.Set("GlobalNetworkIdDeviceId", idWithGlobalNetwork(*s.GlobalNetworkId, *s.DeviceId))

		item := sdp.Item{
			Type:            "networkmanager-device",
			UniqueAttribute: "GlobalNetworkIdDeviceId",
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
				{
					Query: &sdp.Query{
						Type:   "networkmanager-link-association",
						Method: sdp.QueryMethod_SEARCH,
						Query:  idWithTypeAndGlobalNetwork(*s.GlobalNetworkId, "device", *s.DeviceId),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-connection",
						Method: sdp.QueryMethod_SEARCH,
						Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.DeviceId),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
		}

		if s.SiteId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-site",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.SiteId),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}

		if s.DeviceArn != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-network-resource-relationship",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.DeviceArn),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		switch s.State {
		case types.DeviceStatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.DeviceStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.DeviceStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.DeviceStateUpdating:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerDeviceAdapter(client *networkmanager.Client, accountID string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetDevicesInput, *networkmanager.GetDevicesOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetDevicesInput, *networkmanager.GetDevicesOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:    client,
		AccountID: accountID,
		ItemType:  "networkmanager-device",
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetDevicesInput) (*networkmanager.GetDevicesOutput, error) {
			return client.GetDevices(ctx, input)
		},
		AdapterMetadata: networkmanagerDeviceAdapterMetadata,
		InputMapperGet: func(scope, query string) (*networkmanager.GetDevicesInput, error) {
			// We are using a custom id of {globalNetworkId}|{deviceId}
			sections := strings.Split(query, "|")

			if len(sections) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-device get function",
				}
			}
			return &networkmanager.GetDevicesInput{
				GlobalNetworkId: &sections[0],
				DeviceIds: []string{
					sections[1],
				},
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.GetDevicesInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-device, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetDevicesInput) adapterhelpers.Paginator[*networkmanager.GetDevicesOutput, *networkmanager.Options] {
			return networkmanager.NewGetDevicesPaginator(client, params)
		},
		OutputMapper: deviceOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetDevicesInput, error) {
			// Try to parse as ARN first
			arn, err := adapterhelpers.ParseARN(query)
			if err == nil {
				// Check if it's a networkmanager-device ARN
				if arn.Service == "networkmanager" && arn.Type() == "device" {
					// Parse the resource part: device/global-network-{id}/device-{id}
					// Expected format: device/global-network-01231231231231231/device-07f6fd08867abc123
					resourceParts := strings.Split(arn.Resource, "/")
					if len(resourceParts) == 3 && resourceParts[0] == "device" && strings.HasPrefix(resourceParts[1], "global-network-") && strings.HasPrefix(resourceParts[2], "device-") {
						globalNetworkId := resourceParts[1] // Keep full ID including "global-network-" prefix
						deviceId := resourceParts[2]        // Keep full ID including "device-" prefix

						return &networkmanager.GetDevicesInput{
							GlobalNetworkId: &globalNetworkId,
							DeviceIds:       []string{deviceId},
						}, nil
					}
				}

				// If it's not a valid networkmanager-device ARN, return an error
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "ARN is not a valid networkmanager-device ARN",
				}
			}

			// If not an ARN, fall back to the original logic
			// We may search by only globalNetworkId or by using a custom id of {globalNetworkId}|{siteId}
			sections := strings.Split(query, "|")
			switch len(sections) {
			case 1:
				// globalNetworkId
				return &networkmanager.GetDevicesInput{
					GlobalNetworkId: &sections[0],
				}, nil
			case 2:
				// {globalNetworkId}|{siteId}
				return &networkmanager.GetDevicesInput{
					GlobalNetworkId: &sections[0],
					SiteId:          &sections[1],
				}, nil
			default:
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-device get function",
				}
			}

		},
	}
}

var networkmanagerDeviceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-device",
	DescriptiveName: "Networkmanager Device",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a Networkmanager Device",
		SearchDescription: "Search for Networkmanager Devices by GlobalNetworkId, {GlobalNetworkId|SiteId} or ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_networkmanager_device.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"networkmanager-global-network", "networkmanager-site", "networkmanager-link-association", "networkmanager-connection", "networkmanager-network-resource-relationship"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
