package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func volumeStatusInputMapperGet(scope string, query string) (*ec2.DescribeVolumeStatusInput, error) {
	return &ec2.DescribeVolumeStatusInput{
		VolumeIds: []string{
			query,
		},
	}, nil
}

func volumeStatusInputMapperList(scope string) (*ec2.DescribeVolumeStatusInput, error) {
	return &ec2.DescribeVolumeStatusInput{}, nil
}

func volumeStatusOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeVolumeStatusInput, output *ec2.DescribeVolumeStatusOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, volume := range output.VolumeStatuses {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(volume)

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-volume-status",
			UniqueAttribute: "VolumeId",
			Scope:           scope,
			Attributes:      attrs,
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						// Always get the volume
						Type:   "ec2-volume",
						Method: sdp.QueryMethod_GET,
						Query:  *volume.VolumeId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Volume and status are tightly coupled
						In:  true,
						Out: true,
					},
				},
			},
		}

		if volume.VolumeStatus != nil {
			switch volume.VolumeStatus.Status {
			case types.VolumeStatusInfoStatusImpaired:
				item.Health = sdp.Health_HEALTH_ERROR.Enum()
			case types.VolumeStatusInfoStatusOk:
				item.Health = sdp.Health_HEALTH_OK.Enum()
			case types.VolumeStatusInfoStatusInsufficientData:
				item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
			}
		}

		for _, event := range volume.Events {
			if event.InstanceId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-instance",
						Method: sdp.QueryMethod_GET,
						Query:  *event.InstanceId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Instances and volumes can affect each other
						In:  true,
						Out: true,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2VolumeStatusAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeVolumeStatusInput, *ec2.DescribeVolumeStatusOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeVolumeStatusInput, *ec2.DescribeVolumeStatusOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-volume-status",
		AdapterMetadata: volumeStatusAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeVolumeStatusInput) (*ec2.DescribeVolumeStatusOutput, error) {
			return client.DescribeVolumeStatus(ctx, input)
		},
		InputMapperGet:  volumeStatusInputMapperGet,
		InputMapperList: volumeStatusInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeVolumeStatusInput) adapterhelpers.Paginator[*ec2.DescribeVolumeStatusOutput, *ec2.Options] {
			return ec2.NewDescribeVolumeStatusPaginator(client, params)
		},
		OutputMapper: volumeStatusOutputMapper,
	}
}

var volumeStatusAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-volume-status",
	DescriptiveName: "EC2 Volume Status",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a volume status by volume ID",
		ListDescription:   "List all volume statuses",
		SearchDescription: "Search for volume statuses by ARN",
	},
	PotentialLinks: []string{"ec2-instance"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
})
