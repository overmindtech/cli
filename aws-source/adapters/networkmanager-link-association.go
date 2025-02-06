package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func linkAssociationOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, _ *networkmanager.GetLinkAssociationsInput, output *networkmanager.GetLinkAssociationsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, s := range output.LinkAssociations {
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

		if s.GlobalNetworkId == nil || s.LinkId == nil || s.DeviceId == nil {
			return nil, sdp.NewQueryError(errors.New("globalNetworkId, linkId or deviceId is nil for link association"))
		}

		attrs.Set("GlobalNetworkIdLinkIdDeviceId", fmt.Sprintf("%s|%s|%s", *s.GlobalNetworkId, *s.LinkId, *s.DeviceId))

		item := sdp.Item{
			Type:            "networkmanager-link-association",
			UniqueAttribute: "GlobalNetworkIdLinkIdDeviceId",
			Scope:           scope,
			Attributes:      attrs,
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
						Type:   "networkmanager-link",
						Method: sdp.QueryMethod_GET,
						Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.LinkId),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-device",
						Method: sdp.QueryMethod_GET,
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

		switch s.LinkAssociationState {
		case types.LinkAssociationStatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.LinkAssociationStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.LinkAssociationStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.LinkAssociationStateDeleted:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerLinkAssociationAdapter(client *networkmanager.Client, accountID string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetLinkAssociationsInput, *networkmanager.GetLinkAssociationsOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetLinkAssociationsInput, *networkmanager.GetLinkAssociationsOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		ItemType:        "networkmanager-link-association",
		AdapterMetadata: linkAssociationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetLinkAssociationsInput) (*networkmanager.GetLinkAssociationsOutput, error) {
			return client.GetLinkAssociations(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*networkmanager.GetLinkAssociationsInput, error) {
			// We are using a custom id of "{globalNetworkId}|{linkId}|{deviceId}"
			sections := strings.Split(query, "|")

			if len(sections) != 3 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-link-association get function",
				}
			}
			// "default|link-1|device-1"
			return &networkmanager.GetLinkAssociationsInput{
				GlobalNetworkId: &sections[0],
				LinkId:          &sections[1],
				DeviceId:        &sections[2],
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.GetLinkAssociationsInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-link-association, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetLinkAssociationsInput) adapterhelpers.Paginator[*networkmanager.GetLinkAssociationsOutput, *networkmanager.Options] {
			return networkmanager.NewGetLinkAssociationsPaginator(client, params)
		},
		OutputMapper: linkAssociationOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetLinkAssociationsInput, error) {
			// We may search by only globalNetworkId or by using a custom id of {globalNetworkId}|recourceType|recourceId f.e.:
			// default|link|link-1
			// default|device|dvc-1
			sections := strings.Split(query, "|")
			switch len(sections) {
			case 1:
				// globalNetworkId
				return &networkmanager.GetLinkAssociationsInput{
					GlobalNetworkId: &sections[0],
				}, nil
			case 3:
				switch sections[1] {
				case "link":
					// default|link|link-1
					return &networkmanager.GetLinkAssociationsInput{
						GlobalNetworkId: &sections[0],
						LinkId:          &sections[2],
					}, nil
				case "device":
					// default|device|dvc-1
					return &networkmanager.GetLinkAssociationsInput{
						GlobalNetworkId: &sections[0],
						DeviceId:        &sections[2],
					}, nil
				default:
					return nil, &sdp.QueryError{
						ErrorType:   sdp.QueryError_NOTFOUND,
						ErrorString: fmt.Sprintf("invalid query for networkmanager-link-association get function, unknown resource type: %v", sections[1]),
					}

				}
			default:
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-link-association get function",
				}
			}
		},
	}
}

var linkAssociationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-link-association",
	DescriptiveName: "Networkmanager LinkAssociation",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a Networkmanager Link Association",
		SearchDescription: "Search for Networkmanager Link Associations by GlobalNetworkId and DeviceId or LinkId",
	},
	PotentialLinks: []string{"networkmanager-global-network", "networkmanager-link", "networkmanager-device"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
