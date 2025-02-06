package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func getSiteToSiteVpnAttachmentGetFunc(ctx context.Context, client *networkmanager.Client, _, query string) (*types.SiteToSiteVpnAttachment, error) {
	out, err := client.GetSiteToSiteVpnAttachment(ctx, &networkmanager.GetSiteToSiteVpnAttachmentInput{
		AttachmentId: &query,
	})
	if err != nil {
		return nil, err
	}

	return out.SiteToSiteVpnAttachment, nil
}

func siteToSiteVpnAttachmentItemMapper(_, scope string, awsItem *types.SiteToSiteVpnAttachment) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	// The uniqueAttributeValue for this is a nested value of peeringId:
	if awsItem != nil && awsItem.Attachment != nil {
		attributes.Set("AttachmentId", *awsItem.Attachment.AttachmentId)
	}

	item := sdp.Item{
		Type:            "networkmanager-site-to-site-vpn-attachment",
		UniqueAttribute: "AttachmentId",
		Attributes:      attributes,
		Scope:           scope,
	}

	if awsItem.Attachment != nil {
		if awsItem.Attachment.CoreNetworkId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					// Search for core network
					Type:   "networkmanager-core-network",
					Method: sdp.QueryMethod_GET,
					Query:  *awsItem.Attachment.CoreNetworkId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}

		switch awsItem.Attachment.State { //nolint:exhaustive
		case types.AttachmentStateCreating:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.AttachmentStateAvailable:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.AttachmentStateDeleting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.AttachmentStateFailed:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}
	if awsItem.VpnConnectionArn != nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ec2-vpn-connection",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *awsItem.VpnConnectionArn,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	return &item, nil
}

func NewNetworkManagerSiteToSiteVpnAttachmentAdapter(client *networkmanager.Client, accountID, region string) *adapterhelpers.GetListAdapter[*types.SiteToSiteVpnAttachment, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.GetListAdapter[*types.SiteToSiteVpnAttachment, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ItemType:        "networkmanager-site-to-site-vpn-attachment",
		AdapterMetadata: siteToSiteVpnAttachmentAdapterMetadata,
		GetFunc: func(ctx context.Context, client *networkmanager.Client, scope string, query string) (*types.SiteToSiteVpnAttachment, error) {
			return getSiteToSiteVpnAttachmentGetFunc(ctx, client, scope, query)
		},
		ItemMapper: siteToSiteVpnAttachmentItemMapper,
		ListFunc: func(ctx context.Context, client *networkmanager.Client, scope string) ([]*types.SiteToSiteVpnAttachment, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-site-to-site-vpn-attachment, use get",
			}
		},
	}
}

var siteToSiteVpnAttachmentAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-site-to-site-vpn-attachment",
	DescriptiveName: "Networkmanager Site To Site Vpn Attachment",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:            true,
		GetDescription: "Get a Networkmanager Site To Site Vpn Attachment by id",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_networkmanager_site_to_site_vpn_attachment.id"},
	},
	PotentialLinks: []string{"networkmanager-core-network", "ec2-vpn-connection"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
