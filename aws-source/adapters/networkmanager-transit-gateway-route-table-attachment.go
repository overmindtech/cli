package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func getTransitGatewayRouteTableAttachmentGetFunc(ctx context.Context, client *networkmanager.Client, _, query string) (*types.TransitGatewayRouteTableAttachment, error) {
	out, err := client.GetTransitGatewayRouteTableAttachment(ctx, &networkmanager.GetTransitGatewayRouteTableAttachmentInput{
		AttachmentId: &query,
	})
	if err != nil {
		return nil, err
	}

	return out.TransitGatewayRouteTableAttachment, nil
}

func transitGatewayRouteTableAttachmentItemMapper(_, scope string, awsItem *types.TransitGatewayRouteTableAttachment) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	// The uniqueAttributeValue for this is a nested value of AttachmentId:
	if awsItem != nil && awsItem.Attachment != nil {
		attributes.Set("AttachmentId", *awsItem.Attachment.AttachmentId)
	}

	item := sdp.Item{
		Type:            "networkmanager-transit-gateway-route-table-attachment",
		UniqueAttribute: "AttachmentId",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            networkmanagerTagsToMap(awsItem.Attachment.Tags),
	}

	if awsItem.Attachment != nil && awsItem.Attachment.CoreNetworkId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "networkmanager-core-network",
				Method: sdp.QueryMethod_GET,
				Query:  *awsItem.Attachment.CoreNetworkId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		})
	}

	if awsItem.PeeringId != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "networkmanager-transit-gateway-peering",
				Method: sdp.QueryMethod_GET,
				Query:  *awsItem.PeeringId,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	// ARN example: "arn:aws:ec2:us-west-2:123456789012:transit-gateway-route-table/tgw-rtb-9876543210123456"
	if awsItem.TransitGatewayRouteTableArn != nil {
		if arn, err := adapterhelpers.ParseARN(*awsItem.TransitGatewayRouteTableArn); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-transit-gateway-route-table",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *awsItem.TransitGatewayRouteTableArn,
					Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	return &item, nil
}

func NewNetworkManagerTransitGatewayRouteTableAttachmentAdapter(client *networkmanager.Client, accountID, region string) *adapterhelpers.GetListAdapter[*types.TransitGatewayRouteTableAttachment, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.GetListAdapter[*types.TransitGatewayRouteTableAttachment, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "networkmanager-transit-gateway-route-table-attachment",
		AdapterMetadata: transitGatewayRouteTableAttachmentAdapterMetadata,
		GetFunc: func(ctx context.Context, client *networkmanager.Client, scope string, query string) (*types.TransitGatewayRouteTableAttachment, error) {
			return getTransitGatewayRouteTableAttachmentGetFunc(ctx, client, scope, query)
		},
		ItemMapper: transitGatewayRouteTableAttachmentItemMapper,
		ListFunc: func(ctx context.Context, client *networkmanager.Client, scope string) ([]*types.TransitGatewayRouteTableAttachment, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-transit-gateway-route-table-attachment, use get",
			}
		},
	}
}

var transitGatewayRouteTableAttachmentAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-transit-gateway-route-table-attachment",
	DescriptiveName: "Networkmanager Transit Gateway Route Table Attachment",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:            true,
		GetDescription: "Get a Networkmanager Transit Gateway Route Table Attachment by id",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_networkmanager_transit_gateway_route_table_attachment.id"},
	},
	PotentialLinks: []string{"networkmanager-core-network", "networkmanager-transit-gateway-peering", "ec2-transit-gateway-route-table"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
