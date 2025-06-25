package adapters

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/efs"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func FileSystemOutputMapper(_ context.Context, _ *efs.Client, scope string, input *efs.DescribeFileSystemsInput, output *efs.DescribeFileSystemsOutput) ([]*sdp.Item, error) {
	if output == nil {
		return nil, errors.New("nil output from AWS")
	}

	items := make([]*sdp.Item, 0)

	for _, fs := range output.FileSystems {
		attrs, err := adapterhelpers.ToAttributesWithExclude(fs, "tags")

		if err != nil {
			return nil, err
		}

		if fs.FileSystemId == nil {
			return nil, errors.New("filesystem has nil id")
		}

		item := sdp.Item{
			Type:            "efs-file-system",
			UniqueAttribute: "FileSystemId",
			Scope:           scope,
			Attributes:      attrs,
			Health:          lifeCycleStateToHealth(fs.LifeCycleState),
			Tags:            efsTagsToMap(fs.Tags),
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "efs-backup-policy",
						Method: sdp.QueryMethod_GET,
						Query:  *fs.FileSystemId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the backup policy could effect the file
						// system in that it might no longer be backed up
						In: true,
						// Changing the file system will not effect the backup
						Out: false,
					},
				},
				{
					Query: &sdp.Query{
						Type:   "efs-mount-target",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *fs.FileSystemId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// These are tightly coupled
						In:  true,
						Out: true,
					},
				},
			},
		}

		if fs.KmsKeyId != nil {
			// KMS key ID is an ARN
			if arn, err := adapterhelpers.ParseARN(*fs.KmsKeyId); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "kms-key",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *fs.KmsKeyId,
						Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the key will affect us
						In: true,
						// We can't affect the key
						Out: false,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEFSFileSystemAdapter(client *efs.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*efs.DescribeFileSystemsInput, *efs.DescribeFileSystemsOutput, *efs.Client, *efs.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*efs.DescribeFileSystemsInput, *efs.DescribeFileSystemsOutput, *efs.Client, *efs.Options]{
		ItemType:        "efs-file-system",
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		AdapterMetadata: efsFileSystemAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *efs.Client, input *efs.DescribeFileSystemsInput) (*efs.DescribeFileSystemsOutput, error) {
			return client.DescribeFileSystems(ctx, input)
		},
		PaginatorBuilder: func(client *efs.Client, params *efs.DescribeFileSystemsInput) adapterhelpers.Paginator[*efs.DescribeFileSystemsOutput, *efs.Options] {
			return efs.NewDescribeFileSystemsPaginator(client, params)
		},
		InputMapperGet: func(scope, query string) (*efs.DescribeFileSystemsInput, error) {
			return &efs.DescribeFileSystemsInput{
				FileSystemId: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*efs.DescribeFileSystemsInput, error) {
			return &efs.DescribeFileSystemsInput{}, nil
		},
		OutputMapper: FileSystemOutputMapper,
	}
}

var efsFileSystemAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "efs-file-system",
	DescriptiveName: "EFS File System",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a file system by ID",
		ListDescription:   "List file systems",
		SearchDescription: "Search file systems by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_efs_file_system.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
})
