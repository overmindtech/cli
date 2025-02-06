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

func siteOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, _ *networkmanager.GetSitesInput, output *networkmanager.GetSitesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, s := range output.Sites {
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

		if s.GlobalNetworkId == nil || s.SiteId == nil {
			return nil, sdp.NewQueryError(errors.New("globalNetworkId or siteId is nil for site"))
		}

		attrs.Set("GlobalNetworkIdSiteId", idWithGlobalNetwork(*s.GlobalNetworkId, *s.SiteId))

		item := sdp.Item{
			Type:            "networkmanager-site",
			UniqueAttribute: "GlobalNetworkIdSiteId",
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
						Type:   "networkmanager-link",
						Method: sdp.QueryMethod_SEARCH,
						Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.SiteId),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-device",
						Method: sdp.QueryMethod_SEARCH,
						Query:  idWithGlobalNetwork(*s.GlobalNetworkId, *s.SiteId),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			},
		}
		switch s.State {
		case types.SiteStatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.SiteStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.SiteStateUpdating:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.SiteStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerSiteAdapter(client *networkmanager.Client, accountID string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetSitesInput, *networkmanager.GetSitesOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetSitesInput, *networkmanager.GetSitesOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		ItemType:        "networkmanager-site",
		AdapterMetadata: siteAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetSitesInput) (*networkmanager.GetSitesOutput, error) {
			return client.GetSites(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*networkmanager.GetSitesInput, error) {
			// We are using a custom id of {globalNetworkId}|{siteId}
			sections := strings.Split(query, "|")

			if len(sections) != 2 {
				return nil, &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "invalid query for networkmanager-site get function",
				}
			}
			return &networkmanager.GetSitesInput{
				GlobalNetworkId: &sections[0],
				SiteIds: []string{
					sections[1],
				},
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.GetSitesInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-site, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetSitesInput) adapterhelpers.Paginator[*networkmanager.GetSitesOutput, *networkmanager.Options] {
			return networkmanager.NewGetSitesPaginator(client, params)
		},
		OutputMapper: siteOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetSitesInput, error) {
			// Search by GlobalNetworkId
			return &networkmanager.GetSitesInput{
				GlobalNetworkId: &query,
			}, nil
		},
	}
}

var siteAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-site",
	DescriptiveName: "Networkmanager Site",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a Networkmanager Site",
		SearchDescription: "Search for Networkmanager Sites by GlobalNetworkId",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_networkmanager_site.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"networkmanager-global-network", "networkmanager-link", "networkmanager-device"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
