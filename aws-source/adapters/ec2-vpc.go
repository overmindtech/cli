package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func vpcInputMapperGet(scope string, query string) (*ec2.DescribeVpcsInput, error) {
	return &ec2.DescribeVpcsInput{
		VpcIds: []string{
			query,
		},
	}, nil
}

func vpcInputMapperList(scope string) (*ec2.DescribeVpcsInput, error) {
	return &ec2.DescribeVpcsInput{}, nil
}

func vpcOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeVpcsInput, output *ec2.DescribeVpcsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, vpc := range output.Vpcs {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(vpc, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-vpc",
			UniqueAttribute: "VpcId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(vpc.Tags),
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2VpcAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeVpcsInput, *ec2.DescribeVpcsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeVpcsInput, *ec2.DescribeVpcsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-vpc",
		AdapterMetadata: vpcAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error) {
			return client.DescribeVpcs(ctx, input)
		},
		InputMapperGet:  vpcInputMapperGet,
		InputMapperList: vpcInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeVpcsInput) adapterhelpers.Paginator[*ec2.DescribeVpcsOutput, *ec2.Options] {
			return ec2.NewDescribeVpcsPaginator(client, params)
		},
		OutputMapper: vpcOutputMapper,
	}
}

var vpcAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "VPC",
	Type:            "ec2-vpc",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:             true,
		List:            true,
		GetDescription:  "Get a VPC by ID",
		ListDescription: "List all VPCs",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_vpc.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
