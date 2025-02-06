package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func networkResourceRelationshipOutputMapper(_ context.Context, _ *networkmanager.Client, scope string, input *networkmanager.GetNetworkResourceRelationshipsInput, output *networkmanager.GetNetworkResourceRelationshipsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)
	// Connecting networkmanager-global-network with internal or external resources happening in
	// networkmanager-network-resource source
	// No point to double-link same resources to networkmanager-global-network here again
	// Instead here we will create connections between these resources itself

	for _, relationship := range output.Relationships {
		if relationship.From == nil || relationship.To == nil {
			continue
		}

		// Parse the ARNs
		fromArn, err := adapterhelpers.ParseARN(*relationship.From)

		if err != nil {
			return nil, err
		}

		toArn, err := adapterhelpers.ParseARN(*relationship.To)

		if err != nil {
			return nil, err
		}

		// We need to create a unique attribute for each item so we'll create a
		// hash to avoid it being too long
		hasher := sha256.New()
		hasher.Write([]byte(fromArn.String()))
		hasher.Write([]byte(toArn.String()))
		sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

		attrs, err := sdp.ToAttributes(map[string]interface{}{
			"Hash": sha,
			"From": fromArn.String(),
			"To":   toArn.String(),
		})
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:              "networkmanager-network-resource-relationship",
			UniqueAttribute:   "Hash",
			Scope:             scope,
			Attributes:        attrs,
			LinkedItemQueries: []*sdp.LinkedItemQuery{},
		}

		toResourceType := fmt.Sprintf("%s-%s", toArn.Service, toArn.Type())
		// For each linked item we must define +overmind:link comment section
		switch toResourceType {
		case "networkmanager-connection":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-connection",
					Method: sdp.QueryMethod_SEARCH,
					Query:  idWithGlobalNetwork(*input.GlobalNetworkId, toArn.ResourceID()),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "networkmanager-device":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-device",
					Method: sdp.QueryMethod_SEARCH,
					Query:  idWithGlobalNetwork(*input.GlobalNetworkId, toArn.ResourceID()),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "networkmanager-link":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-link",
					Method: sdp.QueryMethod_SEARCH,
					Query:  idWithGlobalNetwork(*input.GlobalNetworkId, toArn.ResourceID()),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "networkmanager-site":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-site",
					Method: sdp.QueryMethod_SEARCH,
					Query:  idWithGlobalNetwork(*input.GlobalNetworkId, toArn.ResourceID()),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "directconnect-connection":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-connection",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "directconnect-direct-connect-gateway":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-direct-connect-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "directconnect-virtual-interface":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-virtual-interface",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "ec2-customer-gateway":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-customer-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "ec2-transit-gateway":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "ec2-transit-gateway-attachment":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway-attachment",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "ec2-transit-gateway-connect-peer":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway-connect-peer",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "ec2-transit-gateway-route-table":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway-route-table",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		case "ec2-vpn-connection":
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpn-connection",
					Method: sdp.QueryMethod_GET,
					Query:  toArn.ResourceID(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		default:
			// skip unknown item types
			continue
		}
		items = append(items, &item)
	}

	return items, nil
}

func NewNetworkManagerNetworkResourceRelationshipsAdapter(client *networkmanager.Client, accountID, region string) *adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetNetworkResourceRelationshipsInput, *networkmanager.GetNetworkResourceRelationshipsOutput, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*networkmanager.GetNetworkResourceRelationshipsInput, *networkmanager.GetNetworkResourceRelationshipsOutput, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "networkmanager-network-resource-relationship",
		AdapterMetadata: networkResourceRelationshipAdapterMetadata,
		OutputMapper:    networkResourceRelationshipOutputMapper,
		DescribeFunc: func(ctx context.Context, client *networkmanager.Client, input *networkmanager.GetNetworkResourceRelationshipsInput) (*networkmanager.GetNetworkResourceRelationshipsOutput, error) {
			return client.GetNetworkResourceRelationships(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*networkmanager.GetNetworkResourceRelationshipsInput, error) {
			return nil, sdp.NewQueryError(errors.New("get not supported for networkmanager-network-resource-relationship, use search"))
		},
		InputMapperList: func(scope string) (*networkmanager.GetNetworkResourceRelationshipsInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-network-resource-relationship, use search",
			}
		},
		PaginatorBuilder: func(client *networkmanager.Client, params *networkmanager.GetNetworkResourceRelationshipsInput) adapterhelpers.Paginator[*networkmanager.GetNetworkResourceRelationshipsOutput, *networkmanager.Options] {
			return networkmanager.NewGetNetworkResourceRelationshipsPaginator(client, params)
		},
		InputMapperSearch: func(ctx context.Context, client *networkmanager.Client, scope, query string) (*networkmanager.GetNetworkResourceRelationshipsInput, error) {
			// Search by GlobalNetworkId
			return &networkmanager.GetNetworkResourceRelationshipsInput{
				GlobalNetworkId: &query,
			}, nil
		},
	}
}

var networkResourceRelationshipAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-network-resource-relationship",
	DescriptiveName: "Networkmanager Network Resource Relationships",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Search:            true,
		SearchDescription: "Search for Networkmanager NetworkResourceRelationships by GlobalNetworkId",
	},
	PotentialLinks: []string{"networkmanager-connection", "networkmanager-device", "networkmanager-link", "networkmanager-site", "directconnect-connection", "directconnect-direct-connect-gateway", "directconnect-virtual-interface", "ec2-customer"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
