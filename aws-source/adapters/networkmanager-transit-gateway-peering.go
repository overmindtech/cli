package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func getTransitGatewayPeeringGetFunc(ctx context.Context, client *networkmanager.Client, _, query string) (*types.TransitGatewayPeering, error) {
	out, err := client.GetTransitGatewayPeering(ctx, &networkmanager.GetTransitGatewayPeeringInput{
		PeeringId: &query,
	})
	if err != nil {
		return nil, err
	}

	return out.TransitGatewayPeering, nil
}

func transitGatewayPeeringItemMapper(_, scope string, awsItem *types.TransitGatewayPeering) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	// The uniqueAttributeValue for this is a nested value of peeringId:
	if awsItem != nil && awsItem.Peering != nil {
		attributes.Set("PeeringId", *awsItem.Peering.PeeringId)
	}

	item := sdp.Item{
		Type:            "networkmanager-transit-gateway-peering",
		UniqueAttribute: "PeeringId",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            networkmanagerTagsToMap(awsItem.Peering.Tags),
	}

	if awsItem.Peering != nil {
		if awsItem.Peering.CoreNetworkId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "networkmanager-core-network",
					Method: sdp.QueryMethod_GET,
					Query:  *awsItem.Peering.CoreNetworkId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		switch awsItem.Peering.State {
		case types.PeeringStateCreating:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.PeeringStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.PeeringStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.PeeringStateFailed:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		}
	}
	if awsItem.TransitGatewayPeeringAttachmentId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ec2-transit-gateway-peering-attachment",
				Method: sdp.QueryMethod_GET,
				Query:  *awsItem.TransitGatewayPeeringAttachmentId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	// ARN example: "arn:aws:ec2:us-west-2:123456789012:transit-gateway/tgw-1234"
	if awsItem.TransitGatewayArn != nil {
		if arn, err := adapterhelpers.ParseARN(*awsItem.TransitGatewayArn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *awsItem.TransitGatewayArn,
					Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
	}

	return &item, nil
}

func NewNetworkManagerTransitGatewayPeeringAdapter(client *networkmanager.Client, accountID, region string) *adapterhelpers.GetListAdapter[*types.TransitGatewayPeering, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.GetListAdapter[*types.TransitGatewayPeering, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "networkmanager-transit-gateway-peering",
		AdapterMetadata: transitGatewayPeeringAdapterMetadata,
		GetFunc: func(ctx context.Context, client *networkmanager.Client, scope string, query string) (*types.TransitGatewayPeering, error) {
			return getTransitGatewayPeeringGetFunc(ctx, client, scope, query)
		},
		ItemMapper: transitGatewayPeeringItemMapper,
		ListFunc: func(ctx context.Context, client *networkmanager.Client, scope string) ([]*types.TransitGatewayPeering, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-transit-gateway-peering, use get",
			}
		},
	}
}

var transitGatewayPeeringAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-transit-gateway-peering",
	DescriptiveName: "Networkmanager Transit Gateway Peering",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:            true,
		GetDescription: "Get a Networkmanager Transit Gateway Peering by id",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_networkmanager_transit_gateway_peering.id"},
	},
	PotentialLinks: []string{"networkmanager-core-network", "ec2-transit-gateway-peering-attachment", "ec2-transit-gateway"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
