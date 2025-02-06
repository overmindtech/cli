package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func globalNetworkOutputMapper(_ context.Context, client *networkmanager.Client, scope string, _ *networkmanager.DescribeGlobalNetworksInput, output *networkmanager.DescribeGlobalNetworksOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, gn := range output.GlobalNetworks {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(gn, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "networkmanager-global-network",
			UniqueAttribute: "GlobalNetworkId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            networkmanagerTagsToMap(gn.Tags),
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "networkmanager-site",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-transit-gateway-registration",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-connect-peer-association",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-transit-gateway-connect-peer-association",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-network-resource",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-network-resource-relationship",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-link",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-device",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "networkmanager-connection",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *gn.GlobalNetworkId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
			},
		}
		switch gn.State {
		case types.GlobalNetworkStatePending:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.GlobalNetworkStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.GlobalNetworkStateUpdating:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.GlobalNetworkStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}
		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerGlobalNetworkAdapter(client *networkmanager.Client, accountID string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.DescribeGlobalNetworksInput, *networkmanager.DescribeGlobalNetworksOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.DescribeGlobalNetworksInput, *networkmanager.DescribeGlobalNetworksOutput, *networkmanager.Client, *networkmanager.Options]{
		ItemType:        "networkmanager-global-network",
		Client:          client,
		AccountID:       accountID,
		AdapterMetadata: globalNetworkAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.DescribeGlobalNetworksInput) (*networkmanager.DescribeGlobalNetworksOutput, error) {
			return client.DescribeGlobalNetworks(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*networkmanager.DescribeGlobalNetworksInput, error) {
			return &networkmanager.DescribeGlobalNetworksInput{
				GlobalNetworkIds: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*networkmanager.DescribeGlobalNetworksInput, error) {
			return &networkmanager.DescribeGlobalNetworksInput{}, nil
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.DescribeGlobalNetworksInput) adapterhelpers.Paginator[*networkmanager.DescribeGlobalNetworksOutput, *networkmanager.Options] {
			return networkmanager.NewDescribeGlobalNetworksPaginator(client, params)
		},
		OutputMapper: globalNetworkOutputMapper,
	}
}

var globalNetworkAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-global-network",
	DescriptiveName: "Network Manager Global Network",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a global network by id",
		ListDescription:   "List all global networks",
		SearchDescription: "Search for a global network by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_networkmanager_global_network.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"networkmanager-site", "networkmanager-transit-gateway-registration", "networkmanager-connect-peer-association", "networkmanager-transit-gateway-connect-peer-association", "networkmanager-network-resource-relationship", "networkmanager-link", "networkmanager-device", "networkmanager-connection"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})

// idWithGlobalNetwork makes custom ID of given entity with global network ID and this entity ID/ARN
func idWithGlobalNetwork(gn, idOrArn string) string {
	return fmt.Sprintf("%s|%s", gn, idOrArn)
}

// idWithTypeAndGlobalNetwork makes custom ID of given entity with global network ID and this entity type and ID/ARN
func idWithTypeAndGlobalNetwork(gb, rType, idOrArn string) string {
	return fmt.Sprintf("%s|%s|%s", gb, rType, idOrArn)
}
