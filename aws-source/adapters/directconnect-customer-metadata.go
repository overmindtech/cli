package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func customerMetadataOutputMapper(_ context.Context, _ *directconnect.Client, scope string, _ *directconnect.DescribeCustomerMetadataInput, output *directconnect.DescribeCustomerMetadataOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, agreement := range output.Agreements {
		attributes, err := adapterhelpers.ToAttributesWithExclude(agreement, "tags")
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "directconnect-customer-metadata",
			UniqueAttribute: "AgreementName",
			Attributes:      attributes,
			Scope:           scope,
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewDirectConnectCustomerMetadataAdapter(client *directconnect.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeCustomerMetadataInput, *directconnect.DescribeCustomerMetadataOutput, *directconnect.Client, *directconnect.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeCustomerMetadataInput, *directconnect.DescribeCustomerMetadataOutput, *directconnect.Client, *directconnect.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "directconnect-customer-metadata",
		AdapterMetadata: customerMetadataAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *directconnect.Client, input *directconnect.DescribeCustomerMetadataInput) (*directconnect.DescribeCustomerMetadataOutput, error) {
			return client.DescribeCustomerMetadata(ctx, input)
		},
		// We want to use the list API for get and list operations
		UseListForGet: true,
		InputMapperGet: func(scope, _ string) (*directconnect.DescribeCustomerMetadataInput, error) {
			return &directconnect.DescribeCustomerMetadataInput{}, nil
		},
		InputMapperList: func(scope string) (*directconnect.DescribeCustomerMetadataInput, error) {
			return &directconnect.DescribeCustomerMetadataInput{}, nil
		},
		OutputMapper: customerMetadataOutputMapper,
	}
}

var customerMetadataAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "directconnect-customer-metadata",
	DescriptiveName: "Customer Metadata",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a customer agreement by name",
		ListDescription:   "List all customer agreements",
		SearchDescription: "Search customer agreements by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})
