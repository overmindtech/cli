package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func virtualGatewayOutputMapper(_ context.Context, _ *directconnect.Client, scope string, _ *directconnect.DescribeVirtualGatewaysInput, output *directconnect.DescribeVirtualGatewaysOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, virtualGateway := range output.VirtualGateways {
		attributes, err := adapterhelpers.ToAttributesWithExclude(virtualGateway, "tags")
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "directconnect-virtual-gateway",
			UniqueAttribute: "VirtualGatewayId",
			Attributes:      attributes,
			Scope:           scope,
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewDirectConnectVirtualGatewayAdapter(client *directconnect.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeVirtualGatewaysInput, *directconnect.DescribeVirtualGatewaysOutput, *directconnect.Client, *directconnect.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeVirtualGatewaysInput, *directconnect.DescribeVirtualGatewaysOutput, *directconnect.Client, *directconnect.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "directconnect-virtual-gateway",
		AdapterMetadata: virtualGatewayAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *directconnect.Client, input *directconnect.DescribeVirtualGatewaysInput) (*directconnect.DescribeVirtualGatewaysOutput, error) {
			return client.DescribeVirtualGateways(ctx, input)
		},
		// We want to use the list API for get and list operations
		UseListForGet: true,
		InputMapperGet: func(scope, _ string) (*directconnect.DescribeVirtualGatewaysInput, error) {
			return &directconnect.DescribeVirtualGatewaysInput{}, nil
		},
		InputMapperList: func(scope string) (*directconnect.DescribeVirtualGatewaysInput, error) {
			return &directconnect.DescribeVirtualGatewaysInput{}, nil
		},
		OutputMapper: virtualGatewayOutputMapper,
	}
}

var virtualGatewayAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "directconnect-virtual-gateway",
	DescriptiveName: "Direct Connect Virtual Gateway",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a virtual gateway by ID",
		ListDescription:   "List all virtual gateways",
		SearchDescription: "Search virtual gateways by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
