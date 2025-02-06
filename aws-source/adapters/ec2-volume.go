package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func volumeInputMapperGet(scope string, query string) (*ec2.DescribeVolumesInput, error) {
	return &ec2.DescribeVolumesInput{
		VolumeIds: []string{
			query,
		},
	}, nil
}

func volumeInputMapperList(scope string) (*ec2.DescribeVolumesInput, error) {
	return &ec2.DescribeVolumesInput{}, nil
}

func volumeOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeVolumesInput, output *ec2.DescribeVolumesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, volume := range output.Volumes {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(volume, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-volume",
			UniqueAttribute: "VolumeId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(volume.Tags),
		}

		for _, attachment := range volume.Attachments {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-instance",
					Method: sdp.QueryMethod_GET,
					Query:  *attachment.InstanceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The instance and the volume are closely linked
					In:  true,
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2VolumeAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeVolumesInput, *ec2.DescribeVolumesOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeVolumesInput, *ec2.DescribeVolumesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-volume",
		AdapterMetadata: volumeAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeVolumesInput) (*ec2.DescribeVolumesOutput, error) {
			return client.DescribeVolumes(ctx, input)
		},
		InputMapperGet:  volumeInputMapperGet,
		InputMapperList: volumeInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeVolumesInput) adapterhelpers.Paginator[*ec2.DescribeVolumesOutput, *ec2.Options] {
			return ec2.NewDescribeVolumesPaginator(client, params)
		},
		OutputMapper: volumeOutputMapper,
	}
}

var volumeAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-volume",
	DescriptiveName: "EC2 Volume",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a volume by ID",
		ListDescription:   "List all volumes",
		SearchDescription: "Search volumes by ARN",
	},
	PotentialLinks: []string{"ec2-instance"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_ebs_volume.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
})
