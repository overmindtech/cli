package adapters

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/efs"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func AccessPointOutputMapper(_ context.Context, _ *efs.Client, scope string, input *efs.DescribeAccessPointsInput, output *efs.DescribeAccessPointsOutput) ([]*sdp.Item, error) {
	if output == nil {
		return nil, errors.New("nil output from AWS")
	}

	items := make([]*sdp.Item, 0)

	for _, ap := range output.AccessPoints {
		attrs, err := adapterhelpers.ToAttributesWithExclude(ap, "tags")

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "efs-access-point",
			UniqueAttribute: "AccessPointId",
			Scope:           scope,
			Attributes:      attrs,
			Health:          lifeCycleStateToHealth(ap.LifeCycleState),
			Tags:            efsTagsToMap(ap.Tags),
		}

		if ap.FileSystemId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "efs-file-system",
					Method: sdp.QueryMethod_GET,
					Query:  *ap.FileSystemId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Access points are tightly coupled with filesystems
					In:  true,
					Out: true,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEFSAccessPointAdapter(client *efs.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*efs.DescribeAccessPointsInput, *efs.DescribeAccessPointsOutput, *efs.Client, *efs.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*efs.DescribeAccessPointsInput, *efs.DescribeAccessPointsOutput, *efs.Client, *efs.Options]{
		ItemType:        "efs-access-point",
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		AdapterMetadata: accessPointAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *efs.Client, input *efs.DescribeAccessPointsInput) (*efs.DescribeAccessPointsOutput, error) {
			return client.DescribeAccessPoints(ctx, input)
		},
		PaginatorBuilder: func(client *efs.Client, params *efs.DescribeAccessPointsInput) adapterhelpers.Paginator[*efs.DescribeAccessPointsOutput, *efs.Options] {
			return efs.NewDescribeAccessPointsPaginator(client, params)
		},
		InputMapperGet: func(scope, query string) (*efs.DescribeAccessPointsInput, error) {
			return &efs.DescribeAccessPointsInput{
				AccessPointId: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*efs.DescribeAccessPointsInput, error) {
			return &efs.DescribeAccessPointsInput{}, nil
		},
		OutputMapper: AccessPointOutputMapper,
	}
}

var accessPointAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "efs-access-point",
	DescriptiveName: "EFS Access Point",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an access point by ID",
		ListDescription:   "List all access points",
		SearchDescription: "Search for an access point by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_efs_access_point.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
