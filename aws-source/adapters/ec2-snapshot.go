package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func snapshotInputMapperGet(scope string, query string) (*ec2.DescribeSnapshotsInput, error) {
	return &ec2.DescribeSnapshotsInput{
		SnapshotIds: []string{
			query,
		},
	}, nil
}

func snapshotInputMapperList(scope string) (*ec2.DescribeSnapshotsInput, error) {
	return &ec2.DescribeSnapshotsInput{
		OwnerIds: []string{
			// Avoid getting every snapshot in existence, just get the ones
			// relevant to this scope i.e. owned by this account in this region
			"self",
		},
	}, nil
}

func snapshotOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeSnapshotsInput, output *ec2.DescribeSnapshotsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, snapshot := range output.Snapshots {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(snapshot, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-snapshot",
			UniqueAttribute: "SnapshotId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(snapshot.Tags),
		}

		if snapshot.VolumeId != nil {
			// Ignore the arbitrary ID that is used by Amazon
			if *snapshot.VolumeId != "vol-ffffffff" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-volume",
						Method: sdp.QueryMethod_GET,
						Query:  *snapshot.VolumeId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the volume will probably affect the snapshot
						In: true,
						// Changing the snapshot will affect the volume indirectly
						// as applications might rely on snapshots as backups
						// or other use-cases
						Out: true,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2SnapshotAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeSnapshotsInput, *ec2.DescribeSnapshotsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeSnapshotsInput, *ec2.DescribeSnapshotsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-snapshot",
		AdapterMetadata: snapshotAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
			return client.DescribeSnapshots(ctx, input)
		},
		InputMapperGet:  snapshotInputMapperGet,
		InputMapperList: snapshotInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeSnapshotsInput) adapterhelpers.Paginator[*ec2.DescribeSnapshotsOutput, *ec2.Options] {
			return ec2.NewDescribeSnapshotsPaginator(client, params)
		},
		OutputMapper: snapshotOutputMapper,
	}
}

var snapshotAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-snapshot",
	DescriptiveName: "EC2 Snapshot",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a snapshot by ID",
		ListDescription:   "List all snapshots",
		SearchDescription: "Search snapshots by ARN",
	},
	PotentialLinks: []string{"ec2-volume"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
})
