package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func egressOnlyInternetGatewayInputMapperGet(scope string, query string) (*ec2.DescribeEgressOnlyInternetGatewaysInput, error) {
	return &ec2.DescribeEgressOnlyInternetGatewaysInput{
		EgressOnlyInternetGatewayIds: []string{
			query,
		},
	}, nil
}

func egressOnlyInternetGatewayInputMapperList(scope string) (*ec2.DescribeEgressOnlyInternetGatewaysInput, error) {
	return &ec2.DescribeEgressOnlyInternetGatewaysInput{}, nil
}

func egressOnlyInternetGatewayOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeEgressOnlyInternetGatewaysInput, output *ec2.DescribeEgressOnlyInternetGatewaysOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, gw := range output.EgressOnlyInternetGateways {
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
			Type:            "ec2-egress-only-internet-gateway",
			UniqueAttribute: "EgressOnlyInternetGatewayId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(gw.Tags),
		}

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

func NewEC2EgressOnlyInternetGatewayAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeEgressOnlyInternetGatewaysInput, *ec2.DescribeEgressOnlyInternetGatewaysOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeEgressOnlyInternetGatewaysInput, *ec2.DescribeEgressOnlyInternetGatewaysOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-egress-only-internet-gateway",
		AdapterMetadata: egressOnlyInternetGatewayAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeEgressOnlyInternetGatewaysInput) (*ec2.DescribeEgressOnlyInternetGatewaysOutput, error) {
			return client.DescribeEgressOnlyInternetGateways(ctx, input)
		},
		InputMapperGet:  egressOnlyInternetGatewayInputMapperGet,
		InputMapperList: egressOnlyInternetGatewayInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeEgressOnlyInternetGatewaysInput) adapterhelpers.Paginator[*ec2.DescribeEgressOnlyInternetGatewaysOutput, *ec2.Options] {
			return ec2.NewDescribeEgressOnlyInternetGatewaysPaginator(client, params)
		},
		OutputMapper: egressOnlyInternetGatewayOutputMapper,
	}
}

var egressOnlyInternetGatewayAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-egress-only-internet-gateway",
	DescriptiveName: "Egress Only Internet Gateway",
	PotentialLinks:  []string{"ec2-vpc"},
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an egress only internet gateway by ID",
		ListDescription:   "List all egress only internet gateways",
		SearchDescription: "Search egress only internet gateways by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "egress_only_internet_gateway.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
