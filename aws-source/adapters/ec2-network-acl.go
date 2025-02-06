package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func networkAclInputMapperGet(scope string, query string) (*ec2.DescribeNetworkAclsInput, error) {
	return &ec2.DescribeNetworkAclsInput{
		NetworkAclIds: []string{
			query,
		},
	}, nil
}

func networkAclInputMapperList(scope string) (*ec2.DescribeNetworkAclsInput, error) {
	return &ec2.DescribeNetworkAclsInput{}, nil
}

func networkAclOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeNetworkAclsInput, output *ec2.DescribeNetworkAclsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, networkAcl := range output.NetworkAcls {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(networkAcl, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-network-acl",
			UniqueAttribute: "NetworkAclId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(networkAcl.Tags),
		}

		for _, assoc := range networkAcl.Associations {
			if assoc.SubnetId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-subnet",
						Method: sdp.QueryMethod_GET,
						Query:  *assoc.SubnetId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the subnet won't affect the ACL
						In: false,
						// Changing the ACL will affect the subnet
						Out: true,
					},
				})
			}
		}

		if networkAcl.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *networkAcl.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the VPC won't affect the ACL
					In: false,
					// Changing the ACL will affect the VPC
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2NetworkAclAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeNetworkAclsInput, *ec2.DescribeNetworkAclsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeNetworkAclsInput, *ec2.DescribeNetworkAclsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-network-acl",
		AdapterMetadata: networkAclAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeNetworkAclsInput) (*ec2.DescribeNetworkAclsOutput, error) {
			return client.DescribeNetworkAcls(ctx, input)
		},
		InputMapperGet:  networkAclInputMapperGet,
		InputMapperList: networkAclInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeNetworkAclsInput) adapterhelpers.Paginator[*ec2.DescribeNetworkAclsOutput, *ec2.Options] {
			return ec2.NewDescribeNetworkAclsPaginator(client, params)
		},
		OutputMapper: networkAclOutputMapper,
	}
}

var networkAclAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-network-acl",
	DescriptiveName: "Network ACL",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a network ACL",
		ListDescription:   "List all network ACLs",
		SearchDescription: "Search for network ACLs by ARN",
	},
	PotentialLinks: []string{"ec2-subnet", "ec2-vpc"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_network_acl.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
