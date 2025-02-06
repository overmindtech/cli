package adapters

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func linkOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, _ *networkmanager.GetLinksInput, output *networkmanager.GetLinksOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, s := range output.Links {
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

		attrs.Set("GlobalNetworkIdLinkId", idWithGlobalNetwork(*s.GlobalNetworkId, *s.LinkId))

		item := sdp.Item{
			Type:            "networkmanager-link",
			UniqueAttribute: "GlobalNetworkIdLinkId",
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
						Query:  idWithTypeAndGlobalNetwork(*s.GlobalNetworkId, "link", *s.LinkId),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
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
					Out: true,
				},
			})
		}

		if s.LinkArn != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-network-resource-relationship",
					Method: sdp.QueryMethod_GET,
					Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.LinkArn),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		switch s.State {
		case types.LinkStatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.LinkStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.LinkStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.LinkStateUpdating:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerLinkAdapter(client *networkmanager.Client, accountID string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetLinksInput, *networkmanager.GetLinksOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetLinksInput, *networkmanager.GetLinksOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		ItemType:        "networkmanager-link",
		AdapterMetadata: linkAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetLinksInput) (*networkmanager.GetLinksOutput, error) {
			return client.GetLinks(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*networkmanager.GetLinksInput, error) {
			// We are using a custom id of {globalNetworkId}|{linkId}
			sections := strings.Split(query, "|")

			if len(sections) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-link get function",
				}
			}
			return &networkmanager.GetLinksInput{
				GlobalNetworkId: &sections[0],
				LinkIds: []string{
					sections[1],
				},
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.GetLinksInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-link, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetLinksInput) adapterhelpers.Paginator[*networkmanager.GetLinksOutput, *networkmanager.Options] {
			return networkmanager.NewGetLinksPaginator(client, params)
		},
		OutputMapper: linkOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetLinksInput, error) {
			// We may search by only globalNetworkId or by using a custom id of {globalNetworkId}|{siteId}
			sections := strings.Split(query, "|")
			switch len(sections) {
			case 1:
				// globalNetworkId
				return &networkmanager.GetLinksInput{
					GlobalNetworkId: &sections[0],
				}, nil
			case 2:
				// {globalNetworkId}|{siteId}
				return &networkmanager.GetLinksInput{
					GlobalNetworkId: &sections[0],
					SiteId:          &sections[1],
				}, nil
			default:
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-link get function",
				}
			}

		},
	}
}

var linkAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-link",
	DescriptiveName: "Networkmanager Link",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a Networkmanager Link",
		SearchDescription: "Search for Networkmanager Links by GlobalNetworkId, or by GlobalNetworkId with SiteId",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_networkmanager_link.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"networkmanager-global-network", "networkmanager-link-association", "networkmanager-site", "networkmanager-network-resource-relationship"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
