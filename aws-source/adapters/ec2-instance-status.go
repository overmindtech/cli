package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func instanceStatusInputMapperGet(scope, query string) (*ec2.DescribeInstanceStatusInput, error) {
	return &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{
			query,
		},
	}, nil
}

func instanceStatusInputMapperList(scope string) (*ec2.DescribeInstanceStatusInput, error) {
	return &ec2.DescribeInstanceStatusInput{}, nil
}

func instanceStatusOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeInstanceStatusInput, output *ec2.DescribeInstanceStatusOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, instanceStatus := range output.InstanceStatuses {
		attrs, err := adapterhelpers.ToAttributesWithExclude(instanceStatus)

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-instance-status",
			UniqueAttribute: "InstanceId",
			Scope:           scope,
			Attributes:      attrs,
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "ec2-instance",
						Method: sdp.QueryMethod_GET,
						Query:  *instanceStatus.InstanceId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// The statius and the instance are closely linked and
						// affect each other
						In:  true,
						Out: true,
					},
				},
			},
		}

		switch instanceStatus.SystemStatus.Status {
		case types.SummaryStatusOk:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case types.SummaryStatusImpaired:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		case types.SummaryStatusInsufficientData:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		case types.SummaryStatusNotApplicable:
			item.Health = nil
		case types.SummaryStatusInitializing:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2InstanceStatusAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeInstanceStatusInput, *ec2.DescribeInstanceStatusOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeInstanceStatusInput, *ec2.DescribeInstanceStatusOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-instance-status",
		AdapterMetadata: instanceStatusAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeInstanceStatusInput) (*ec2.DescribeInstanceStatusOutput, error) {
			return client.DescribeInstanceStatus(ctx, input)
		},
		InputMapperGet:  instanceStatusInputMapperGet,
		InputMapperList: instanceStatusInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeInstanceStatusInput) adapterhelpers.Paginator[*ec2.DescribeInstanceStatusOutput, *ec2.Options] {
			return ec2.NewDescribeInstanceStatusPaginator(client, params)
		},
		OutputMapper: instanceStatusOutputMapper,
	}
}

var instanceStatusAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-instance-status",
	DescriptiveName: "EC2 Instance Status",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an EC2 instance status by Instance ID",
		ListDescription:   "List all EC2 instance statuses",
		SearchDescription: "Search EC2 instance statuses by ARN",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
})
