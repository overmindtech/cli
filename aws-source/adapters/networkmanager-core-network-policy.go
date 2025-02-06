package adapters

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func coreNetworkPolicyGetFunc(ctx context.Context, client *networkmanager.Client, _, query string) (*types.CoreNetworkPolicy, error) {
	out, err := client.GetCoreNetworkPolicy(ctx, &networkmanager.GetCoreNetworkPolicyInput{
		CoreNetworkId: &query,
	})
	if err != nil {
		return nil, err
	}

	return out.CoreNetworkPolicy, nil
}

func coreNetworkPolicyItemMapper(_, scope string, cn *types.CoreNetworkPolicy) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(cn)
	if err != nil {
		return nil, err
	}

	if cn.CoreNetworkId == nil {
		return nil, sdp.NewQueryError(errors.New("coreNetworkId is nil for core network policy"))
	}

	item := sdp.Item{
		Type:            "networkmanager-core-network-policy",
		UniqueAttribute: "CoreNetworkId",
		Attributes:      attributes,
		Scope:           scope,
		LinkedItemQueries: []*sdp.LinkedItemQuery{
			{
				Query: &sdp.Query{
					Type:   "networkmanager-core-network",
					Method: sdp.QueryMethod_GET,
					Query:  *cn.CoreNetworkId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
		},
	}

	return &item, nil
}

func NewNetworkManagerCoreNetworkPolicyAdapter(client *networkmanager.Client, accountID, region string) *adapterhelpers.GetListAdapter[*types.CoreNetworkPolicy, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.GetListAdapter[*types.CoreNetworkPolicy, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "networkmanager-core-network-policy",
		AdapterMetadata: coreNetworkPolicyAdapterMetadata,
		GetFunc: func(ctx context.Context, client *networkmanager.Client, scope string, query string) (*types.CoreNetworkPolicy, error) {
			return coreNetworkPolicyGetFunc(ctx, client, scope, query)
		},
		ItemMapper: coreNetworkPolicyItemMapper,
		ListFunc: func(ctx context.Context, client *networkmanager.Client, scope string) ([]*types.CoreNetworkPolicy, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-core-network-policy, use get",
			}
		},
	}
}

var coreNetworkPolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-core-network-policy",
	DescriptiveName: "Networkmanager Core Network Policy",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:            true,
		GetDescription: "Get a Networkmanager Core Network Policy by Core Network id",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_networkmanager_core_network_policy.core_network_id"},
	},
	PotentialLinks: []string{"networkmanager-core-network"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
