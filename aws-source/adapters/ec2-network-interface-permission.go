package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func networkInterfacePermissionInputMapperGet(scope string, query string) (*ec2.DescribeNetworkInterfacePermissionsInput, error) {
	return &ec2.DescribeNetworkInterfacePermissionsInput{
		NetworkInterfacePermissionIds: []string{
			query,
		},
	}, nil
}

func networkInterfacePermissionInputMapperList(scope string) (*ec2.DescribeNetworkInterfacePermissionsInput, error) {
	return &ec2.DescribeNetworkInterfacePermissionsInput{}, nil
}

func networkInterfacePermissionOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeNetworkInterfacePermissionsInput, output *ec2.DescribeNetworkInterfacePermissionsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, ni := range output.NetworkInterfacePermissions {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(ni)

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-network-interface-permission",
			UniqueAttribute: "NetworkInterfacePermissionId",
			Scope:           scope,
			Attributes:      attrs,
		}

		if ni.NetworkInterfaceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-network-interface",
					Method: sdp.QueryMethod_GET,
					Query:  *ni.NetworkInterfaceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These permissions are tightly linked
					In:  true,
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2NetworkInterfacePermissionAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeNetworkInterfacePermissionsInput, *ec2.DescribeNetworkInterfacePermissionsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeNetworkInterfacePermissionsInput, *ec2.DescribeNetworkInterfacePermissionsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-network-interface-permission",
		AdapterMetadata: networkInterfacePermissionAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeNetworkInterfacePermissionsInput) (*ec2.DescribeNetworkInterfacePermissionsOutput, error) {
			return client.DescribeNetworkInterfacePermissions(ctx, input)
		},
		InputMapperGet:  networkInterfacePermissionInputMapperGet,
		InputMapperList: networkInterfacePermissionInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeNetworkInterfacePermissionsInput) adapterhelpers.Paginator[*ec2.DescribeNetworkInterfacePermissionsOutput, *ec2.Options] {
			return ec2.NewDescribeNetworkInterfacePermissionsPaginator(client, params)
		},
		OutputMapper: networkInterfacePermissionOutputMapper,
	}
}

var networkInterfacePermissionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-network-interface-permission",
	DescriptiveName: "Network Interface Permission",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a network interface permission by ID",
		ListDescription:   "List all network interface permissions",
		SearchDescription: "Search network interface permissions by ARN",
	},
	PotentialLinks: []string{"ec2-network-interface"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
