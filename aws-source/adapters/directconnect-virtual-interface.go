package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

const gatewayIDVirtualInterfaceIDFormat = "gateway_id/virtual_interface_id"

func virtualInterfaceOutputMapper(_ context.Context, _ *directconnect.Client, scope string, _ *directconnect.DescribeVirtualInterfacesInput, output *directconnect.DescribeVirtualInterfacesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, virtualInterface := range output.VirtualInterfaces {
		attributes, err := adapterhelpers.ToAttributesWithExclude(virtualInterface, "tags")
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "directconnect-virtual-interface",
			UniqueAttribute: "VirtualInterfaceId",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            directconnectTagsToMap(virtualInterface.Tags),
		}

		if virtualInterface.ConnectionId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-connection",
					Method: sdp.QueryMethod_GET,
					Query:  *virtualInterface.ConnectionId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// We cannot delete a connection if it has virtual interfaces
					In: true,
					// We can't affect the connection
					Out: false,
				},
			})
		}

		if virtualInterface.DirectConnectGatewayId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-direct-connect-gateway",
					Method: sdp.QueryMethod_GET,
					Query:  *virtualInterface.DirectConnectGatewayId,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// We cannot delete a direct connect gateway if it has virtual interfaces
					In: true,
					// We can't affect the direct connect gateway
					Out: false,
				},
			})
		}

		if virtualInterface.AmazonAddress != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "rdap-ip-network",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *virtualInterface.AmazonAddress,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// IPs are always linked
					In:  true,
					Out: true,
				},
			})
		}

		if virtualInterface.CustomerAddress != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "rdap-ip-network",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *virtualInterface.CustomerAddress,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// IPs are always linked
					In:  true,
					Out: true,
				},
			})
		}

		// Pinpoint a single attachment
		if virtualInterface.DirectConnectGatewayId != nil && virtualInterface.VirtualInterfaceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-direct-connect-gateway-attachment",
					Method: sdp.QueryMethod_GET,
					// returns a single attachment
					Query: fmt.Sprintf("%s/%s", *virtualInterface.DirectConnectGatewayId, *virtualInterface.VirtualInterfaceId),
					Scope: scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the attachment won't affect virtual interface
					In: false,
					// If virtual interface is deleted, the attachment state will change to detaching
					// https://docs.aws.amazon.com/directconnect/latest/APIReference/API_DirectConnectGatewayAttachment.html#API_DirectConnectGatewayAttachment_Contents
					Out: true,
				},
			})
		}

		// Find all affected attachments
		if virtualInterface.VirtualInterfaceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "directconnect-direct-connect-gateway-attachment",
					Method: sdp.QueryMethod_SEARCH,
					// returns list of attachments for the given virtual interface id
					Query: *virtualInterface.VirtualInterfaceId,
					Scope: scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the attachment won't affect virtual interface
					In: false,
					// If virtual interface is deleted, the attachment state will change to detaching
					// https://docs.aws.amazon.com/directconnect/latest/APIReference/API_DirectConnectGatewayAttachment.html#API_DirectConnectGatewayAttachment_Contents
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewDirectConnectVirtualInterfaceAdapter(client *directconnect.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeVirtualInterfacesInput, *directconnect.DescribeVirtualInterfacesOutput, *directconnect.Client, *directconnect.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeVirtualInterfacesInput, *directconnect.DescribeVirtualInterfacesOutput, *directconnect.Client, *directconnect.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "directconnect-virtual-interface",
		AdapterMetadata: virtualInterfaceAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *directconnect.Client, input *directconnect.DescribeVirtualInterfacesInput) (*directconnect.DescribeVirtualInterfacesOutput, error) {
			return client.DescribeVirtualInterfaces(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*directconnect.DescribeVirtualInterfacesInput, error) {
			return &directconnect.DescribeVirtualInterfacesInput{
				VirtualInterfaceId: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*directconnect.DescribeVirtualInterfacesInput, error) {
			return &directconnect.DescribeVirtualInterfacesInput{}, nil
		},
		OutputMapper: virtualInterfaceOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *directconnect.Client, scope, query string) (*directconnect.DescribeVirtualInterfacesInput, error) {
			return &directconnect.DescribeVirtualInterfacesInput{
				ConnectionId: &query,
			}, nil
		},
	}
}

var virtualInterfaceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "directconnect-virtual-interface",
	DescriptiveName: "Virtual Interface",
	PotentialLinks:  []string{"directconnect-connection", "directconnect-direct-connect-gateway", "rdap-ip-network", "directconnect-direct-connect-gateway-attachment", "directconnect-virtual-interface"},
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a virtual interface by ID",
		ListDescription:   "List all virtual interfaces",
		SearchDescription: "Search virtual interfaces by connection ID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_dx_private_virtual_interface.id"},
		{TerraformQueryMap: "aws_dx_public_virtual_interface.id"},
		{TerraformQueryMap: "aws_dx_transit_virtual_interface.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
