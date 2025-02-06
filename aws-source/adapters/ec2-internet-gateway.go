package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func internetGatewayInputMapperGet(scope string, query string) (*ec2.DescribeInternetGatewaysInput, error) {
	return &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []string{
			query,
		},
	}, nil
}

func internetGatewayInputMapperList(scope string) (*ec2.DescribeInternetGatewaysInput, error) {
	return &ec2.DescribeInternetGatewaysInput{}, nil
}

func internetGatewayOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeInternetGatewaysInput, output *ec2.DescribeInternetGatewaysOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, gw := range output.InternetGateways {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(gw, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-internet-gateway",
			UniqueAttribute: "InternetGatewayId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(gw.Tags),
		}

		// VPCs
		for _, attachment := range gw.Attachments {
			if attachment.VpcId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-vpc",
						Method: sdp.QueryMethod_GET,
						Query:  *attachment.VpcId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the VPC won't affect the gateway
						In: false,
						// Changing the gateway will affect the VPC
						Out: true,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2InternetGatewayAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeInternetGatewaysInput, *ec2.DescribeInternetGatewaysOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeInternetGatewaysInput, *ec2.DescribeInternetGatewaysOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-internet-gateway",
		AdapterMetadata: internetGatewayAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeInternetGatewaysInput) (*ec2.DescribeInternetGatewaysOutput, error) {
			return client.DescribeInternetGateways(ctx, input)
		},
		InputMapperGet:  internetGatewayInputMapperGet,
		InputMapperList: internetGatewayInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeInternetGatewaysInput) adapterhelpers.Paginator[*ec2.DescribeInternetGatewaysOutput, *ec2.Options] {
			return ec2.NewDescribeInternetGatewaysPaginator(client, params)
		},
		OutputMapper: internetGatewayOutputMapper,
	}
}

var internetGatewayAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-internet-gateway",
	DescriptiveName: "Internet Gateway",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an internet gateway by ID",
		ListDescription:   "List all internet gateways",
		SearchDescription: "Search internet gateways by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_internet_gateway.id"},
	},
	PotentialLinks: []string{"ec2-vpc"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
