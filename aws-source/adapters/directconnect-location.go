package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func locationOutputMapper(_ context.Context, _ *directconnect.Client, scope string, _ *directconnect.DescribeLocationsInput, output *directconnect.DescribeLocationsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, location := range output.Locations {
		attributes, err := adapterhelpers.ToAttributesWithExclude(location, "tags")
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "directconnect-location",
			UniqueAttribute: "LocationCode",
			Attributes:      attributes,
			Scope:           scope,
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewDirectConnectLocationAdapter(client *directconnect.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeLocationsInput, *directconnect.DescribeLocationsOutput, *directconnect.Client, *directconnect.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeLocationsInput, *directconnect.DescribeLocationsOutput, *directconnect.Client, *directconnect.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "directconnect-location",
		AdapterMetadata: directconnectLocationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *directconnect.Client, input *directconnect.DescribeLocationsInput) (*directconnect.DescribeLocationsOutput, error) {
			return client.DescribeLocations(ctx, input)
		},
		// We want to use the list API for get and list operations
		UseListForGet: true,
		InputMapperGet: func(scope, _ string) (*directconnect.DescribeLocationsInput, error) {
			return &directconnect.DescribeLocationsInput{}, nil
		},
		InputMapperList: func(scope string) (*directconnect.DescribeLocationsInput, error) {
			return &directconnect.DescribeLocationsInput{}, nil
		},
		OutputMapper: locationOutputMapper,
	}
}

var directconnectLocationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "directconnect-location",
	DescriptiveName: "Direct Connect Location",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a Location by its code",
		ListDescription:   "List all Direct Connect Locations",
		SearchDescription: "Search Direct Connect Locations by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_dx_location.location_code"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
