package adapters

import (
	"context"
	"errors"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func coreNetworkGetFunc(ctx context.Context, client NetworkManagerClient, scope string, input *networkmanager.GetCoreNetworkInput) (*sdp.Item, error) {
	out, err := client.GetCoreNetwork(ctx, input)
	if err != nil {
		return nil, err
	}

	if out.CoreNetwork == nil {
		return nil, sdp.NewQueryError(errors.New("coreNetwork is nil for core network"))
	}

	cn := out.CoreNetwork

	attributes, err := adapterhelpers.ToAttributesWithExclude(cn)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "networkmanager-core-network",
		UniqueAttribute: "CoreNetworkId",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            networkmanagerTagsToMap(cn.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{
			{
				Query: &sdp.Query{
					Type:   "networkmanager-core-network-policy",
					Method: sdp.QueryMethod_GET,
					Query:  *cn.CoreNetworkId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  false,
					Out: true,
				},
			},
			{
				Query: &sdp.Query{
					Type:   "networkmanager-connect-peer",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *cn.CoreNetworkId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  false,
					Out: true,
				},
			},
		},
	}

	if cn.GlobalNetworkId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "networkmanager-global-network",
				Method: sdp.QueryMethod_GET,
				Query:  *cn.GlobalNetworkId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		})
	}

	for _, edge := range cn.Edges {
		if edge.Asn != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "rdap-asn",
					Method: sdp.QueryMethod_GET,
					Query:  strconv.FormatInt(*edge.Asn, 10),
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	switch cn.State {
	case types.CoreNetworkStateCreating:
		item.Health = sdp.Health_HEALTH_PENDING.Enum()
	case types.CoreNetworkStateUpdating:
		item.Health = sdp.Health_HEALTH_PENDING.Enum()
	case types.CoreNetworkStateAvailable:
		item.Health = sdp.Health_HEALTH_OK.Enum()
	case types.CoreNetworkStateDeleting:
		item.Health = sdp.Health_HEALTH_PENDING.Enum()
	}

	return &item, nil
}

func NewNetworkManagerCoreNetworkAdapter(client NetworkManagerClient, accountID, region string) *adapterhelpers.AlwaysGetAdapter[*networkmanager.ListCoreNetworksInput, *networkmanager.ListCoreNetworksOutput, *networkmanager.GetCoreNetworkInput, *networkmanager.GetCoreNetworkOutput, NetworkManagerClient, *networkmanager.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*networkmanager.ListCoreNetworksInput, *networkmanager.ListCoreNetworksOutput, *networkmanager.GetCoreNetworkInput, *networkmanager.GetCoreNetworkOutput, NetworkManagerClient, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		GetFunc:         coreNetworkGetFunc,
		ItemType:        "networkmanager-core-network",
		ListInput:       &networkmanager.ListCoreNetworksInput{},
		AdapterMetadata: coreNetworkAdapterMetadata,
		GetInputMapper: func(scope, query string) *networkmanager.GetCoreNetworkInput {
			return &networkmanager.GetCoreNetworkInput{
				CoreNetworkId: &query,
			}
		},
		ListFuncPaginatorBuilder: func(client NetworkManagerClient, input *networkmanager.ListCoreNetworksInput) adapterhelpers.Paginator[*networkmanager.ListCoreNetworksOutput, *networkmanager.Options] {
			return networkmanager.NewListCoreNetworksPaginator(client, input)
		},
		ListFuncOutputMapper: func(output *networkmanager.ListCoreNetworksOutput, input *networkmanager.ListCoreNetworksInput) ([]*networkmanager.GetCoreNetworkInput, error) {
			queries := make([]*networkmanager.GetCoreNetworkInput, 0, len(output.CoreNetworks))

			for i := range output.CoreNetworks {
				queries = append(queries, &networkmanager.GetCoreNetworkInput{
					CoreNetworkId: output.CoreNetworks[i].CoreNetworkId,
				})
			}

			return queries, nil
		},
	}
}

var coreNetworkAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-core-network",
	DescriptiveName: "Networkmanager Core Network",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:            true,
		GetDescription: "Get a Networkmanager Core Network by id",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_networkmanager_core_network.id"},
	},
	PotentialLinks: []string{"networkmanager-core-network-policy", "networkmanager-connect-peer"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
