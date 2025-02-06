package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func reservedInstanceInputMapperGet(scope, query string) (*ec2.DescribeReservedInstancesInput, error) {
	return &ec2.DescribeReservedInstancesInput{
		ReservedInstancesIds: []string{
			query,
		},
	}, nil
}

func reservedInstanceInputMapperList(scope string) (*ec2.DescribeReservedInstancesInput, error) {
	return &ec2.DescribeReservedInstancesInput{}, nil
}

func reservedInstanceOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeReservedInstancesInput, output *ec2.DescribeReservedInstancesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, reservation := range output.ReservedInstances {
		attrs, err := adapterhelpers.ToAttributesWithExclude(reservation, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-reserved-instance",
			UniqueAttribute: "ReservedInstancesId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(reservation.Tags),
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2ReservedInstanceAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeReservedInstancesInput, *ec2.DescribeReservedInstancesOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeReservedInstancesInput, *ec2.DescribeReservedInstancesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-reserved-instance",
		AdapterMetadata: reservedInstanceAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeReservedInstancesInput) (*ec2.DescribeReservedInstancesOutput, error) {
			return client.DescribeReservedInstances(ctx, input)
		},
		InputMapperGet:  reservedInstanceInputMapperGet,
		InputMapperList: reservedInstanceInputMapperList,
		OutputMapper:    reservedInstanceOutputMapper,
	}
}

var reservedInstanceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-reserved-instance",
	DescriptiveName: "Reserved EC2 Instance",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a reserved EC2 instance by ID",
		ListDescription:   "List all reserved EC2 instances",
		SearchDescription: "Search reserved EC2 instances by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
