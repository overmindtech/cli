package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func subnetInputMapperGet(scope string, query string) (*ec2.DescribeSubnetsInput, error) {
	return &ec2.DescribeSubnetsInput{
		SubnetIds: []string{
			query,
		},
	}, nil
}

func subnetInputMapperList(scope string) (*ec2.DescribeSubnetsInput, error) {
	return &ec2.DescribeSubnetsInput{}, nil
}

func subnetOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeSubnetsInput, output *ec2.DescribeSubnetsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, subnet := range output.Subnets {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(subnet, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-subnet",
			UniqueAttribute: "SubnetId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(subnet.Tags),
		}

		if subnet.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *subnet.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the VPC would affect the subnet
					In: true,
					// Changing the subnet won't affect the VPC
					Out: false,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2SubnetAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeSubnetsInput, *ec2.DescribeSubnetsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeSubnetsInput, *ec2.DescribeSubnetsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-subnet",
		AdapterMetadata: subnetAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error) {
			return client.DescribeSubnets(ctx, input)
		},
		InputMapperGet:  subnetInputMapperGet,
		InputMapperList: subnetInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeSubnetsInput) adapterhelpers.Paginator[*ec2.DescribeSubnetsOutput, *ec2.Options] {
			return ec2.NewDescribeSubnetsPaginator(client, params)
		},
		OutputMapper: subnetOutputMapper,
	}
}

var subnetAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-subnet",
	DescriptiveName: "EC2 Subnet",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a subnet by ID",
		ListDescription:   "List all subnets",
		SearchDescription: "Search for subnets by ARN",
	},
	PotentialLinks: []string{"ec2-vpc"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_route_table_association.subnet_id"},
		{TerraformQueryMap: "aws_subnet.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
