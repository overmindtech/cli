package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	"github.com/aws/aws-sdk-go-v2/service/kms"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func customKeyStoreOutputMapper(_ context.Context, _ *kms.Client, scope string, _ *kms.DescribeCustomKeyStoresInput, output *kms.DescribeCustomKeyStoresOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, customKeyStore := range output.CustomKeyStores {
		attributes, err := adapterhelpers.ToAttributesWithExclude(customKeyStore, "tags")
		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "kms-custom-key-store",
			UniqueAttribute: "CustomKeyStoreId",
			Attributes:      attributes,
			Scope:           scope,
		}

		switch customKeyStore.ConnectionState {
		case types.ConnectionStateTypeConnected:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.ConnectionStateTypeConnecting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case types.ConnectionStateTypeDisconnected:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		case types.ConnectionStateTypeFailed:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		case types.ConnectionStateTypeDisconnecting:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		default:
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: "unknown Connection State",
			}
		}

		if customKeyStore.CloudHsmClusterId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "cloudhsmv2-cluster",
					Method: sdp.QueryMethod_GET,
					Query:  *customKeyStore.CloudHsmClusterId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the CloudHSM cluster will affect the custom key store
					In: true,
					// Updating the custom key store will not affect the CloudHSM cluster
					Out: false,
				},
			})
		}

		if customKeyStore.XksProxyConfiguration != nil &&
			customKeyStore.XksProxyConfiguration.VpcEndpointServiceName != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc-endpoint-service",
					Method: sdp.QueryMethod_SEARCH,
					Query:  fmt.Sprintf("name|%s", *customKeyStore.XksProxyConfiguration.VpcEndpointServiceName),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the VPC endpoint service will affect the custom key store
					In: true,
					// Updating the custom key store will not affect the VPC endpoint service
					Out: false,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewKMSCustomKeyStoreAdapter(client *kms.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*kms.DescribeCustomKeyStoresInput, *kms.DescribeCustomKeyStoresOutput, *kms.Client, *kms.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*kms.DescribeCustomKeyStoresInput, *kms.DescribeCustomKeyStoresOutput, *kms.Client, *kms.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "kms-custom-key-store",
		AdapterMetadata: customKeyStoreAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *kms.Client, input *kms.DescribeCustomKeyStoresInput) (*kms.DescribeCustomKeyStoresOutput, error) {
			return client.DescribeCustomKeyStores(ctx, input)
		},
		InputMapperGet: func(_, query string) (*kms.DescribeCustomKeyStoresInput, error) {
			return &kms.DescribeCustomKeyStoresInput{
				CustomKeyStoreId: &query,
			}, nil
		},
		InputMapperList: func(string) (*kms.DescribeCustomKeyStoresInput, error) {
			return &kms.DescribeCustomKeyStoresInput{}, nil
		},
		OutputMapper: customKeyStoreOutputMapper,
	}
}

var customKeyStoreAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "kms-custom-key-store",
	DescriptiveName: "Custom Key Store",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a custom key store by its ID",
		ListDescription:   "List all custom key stores",
		SearchDescription: "Search custom key store by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_kms_custom_key_store.id",
		},
	},
	PotentialLinks: []string{"cloudhsmv2-cluster", "ec2-vpc-endpoint-service"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
})
