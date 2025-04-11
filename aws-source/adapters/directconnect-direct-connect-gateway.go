package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func directConnectGatewayOutputMapper(ctx context.Context, cli *directconnect.Client, scope string, _ *directconnect.DescribeDirectConnectGatewaysInput, output *directconnect.DescribeDirectConnectGatewaysOutput) ([]*sdp.Item, error) {
	// create a slice of ARNs for the resources
	resourceARNs := make([]string, 0, len(output.DirectConnectGateways))
	for _, directConnectGateway := range output.DirectConnectGateways {
		resourceARNs = append(resourceARNs, directconnectARN(
			scope,
			*directConnectGateway.OwnerAccount,
			*directConnectGateway.DirectConnectGatewayId,
		))
	}

	tags := make(map[string][]types.Tag)
	var err error

	if len(resourceARNs) > 0 {
		// get tags for the resources in a map by their ARNs
		tags, err = arnToTags(ctx, cli, resourceARNs)
		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: err.Error(),
			}
		}
	}

	items := make([]*sdp.Item, 0)
	for _, directConnectGateway := range output.DirectConnectGateways {
		attributes, err := adapterhelpers.ToAttributesWithExclude(directConnectGateway, "tags")
		if err != nil {
			return nil, err
		}

		relevantTags := tags[directconnectARN(scope, *directConnectGateway.OwnerAccount, *directConnectGateway.DirectConnectGatewayId)]

		item := sdp.Item{
			Type:            "directconnect-direct-connect-gateway",
			UniqueAttribute: "DirectConnectGatewayId",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            directconnectTagsToMap(relevantTags),
		}

		// stateChangeError =>The error message if the state of an object failed to advance.
		if directConnectGateway.StateChangeError != nil {
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		} else {
			item.Health = sdp.Health_HEALTH_OK.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

// arn constructs an ARN for a direct connect gateway
// https://docs.aws.amazon.com/managedservices/latest/userguide/find-arn.html
// https://docs.aws.amazon.com/service-authorization/latest/reference/list_awsdirectconnect.html#awsdirectconnect-resources-for-iam-policies
func directconnectARN(region, accountID, gatewayID string) string {
	// arn:aws:service:region:account-id:resource-type/resource-id
	return fmt.Sprintf("arn:aws:directconnect:%s:%s:dx-gateway/%s", region, accountID, gatewayID)
}

func NewDirectConnectGatewayAdapter(client *directconnect.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeDirectConnectGatewaysInput, *directconnect.DescribeDirectConnectGatewaysOutput, *directconnect.Client, *directconnect.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*directconnect.DescribeDirectConnectGatewaysInput, *directconnect.DescribeDirectConnectGatewaysOutput, *directconnect.Client, *directconnect.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "directconnect-direct-connect-gateway",
		AdapterMetadata: directConnectGatewayAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *directconnect.Client, input *directconnect.DescribeDirectConnectGatewaysInput) (*directconnect.DescribeDirectConnectGatewaysOutput, error) {
			return client.DescribeDirectConnectGateways(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*directconnect.DescribeDirectConnectGatewaysInput, error) {
			return &directconnect.DescribeDirectConnectGatewaysInput{
				DirectConnectGatewayId: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*directconnect.DescribeDirectConnectGatewaysInput, error) {
			return &directconnect.DescribeDirectConnectGatewaysInput{}, nil
		},
		OutputMapper: directConnectGatewayOutputMapper,
	}
}

var directConnectGatewayAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "directconnect-direct-connect-gateway",
	DescriptiveName: "Direct Connect Gateway",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a direct connect gateway by ID",
		ListDescription:   "List all direct connect gateways",
		SearchDescription: "Search direct connect gateway by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_dx_gateway.id",
		},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
