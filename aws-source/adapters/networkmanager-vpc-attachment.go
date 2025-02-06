package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func vpcAttachmentGetFunc(ctx context.Context, client *networkmanager.Client, _, query string) (*types.VpcAttachment, error) {
	out, err := client.GetVpcAttachment(ctx, &networkmanager.GetVpcAttachmentInput{
		AttachmentId: &query,
	})
	if err != nil {
		return nil, err
	}

	return out.VpcAttachment, nil
}

func vpcAttachmentItemMapper(_, scope string, awsItem *types.VpcAttachment) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	// The uniqueAttributeValue for this is a nested value of AttachmentId:
	if awsItem != nil && awsItem.Attachment != nil {
		attributes.Set("AttachmentId", *awsItem.Attachment.AttachmentId)
	}

	item := sdp.Item{
		Type:            "networkmanager-vpc-attachment",
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
				Out: true,
			},
		})

	}

	return &item, nil
}

func NewNetworkManagerVPCAttachmentAdapter(client *networkmanager.Client, accountID, region string) *adapterhelpers.GetListAdapter[*types.VpcAttachment, *networkmanager.Client, *networkmanager.Options] {
	return &adapterhelpers.GetListAdapter[*types.VpcAttachment, *networkmanager.Client, *networkmanager.Options]{
		Client:          client,
		Region:          region,
		AccountID:       accountID,
		ItemType:        "networkmanager-vpc-attachment",
		AdapterMetadata: vpcAttachmentAdapterMetadata,
		GetFunc: func(ctx context.Context, client *networkmanager.Client, scope string, query string) (*types.VpcAttachment, error) {
			return vpcAttachmentGetFunc(ctx, client, scope, query)
		},
		ItemMapper: vpcAttachmentItemMapper,
		ListFunc: func(ctx context.Context, client *networkmanager.Client, scope string) ([]*types.VpcAttachment, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for networkmanager-vpc-attachment, use get",
			}
		},
	}
}

var vpcAttachmentAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "networkmanager-vpc-attachment",
	DescriptiveName: "Networkmanager VPC Attachment",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:            true,
		GetDescription: "Get a Networkmanager VPC Attachment by id",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_networkmanager_vpc_attachment.id"},
	},
	PotentialLinks: []string{"networkmanager-core-network"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
