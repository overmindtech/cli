package adapters

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/efs/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func ReplicationConfigurationOutputMapper(_ context.Context, _ *efs.Client, scope string, input *efs.DescribeReplicationConfigurationsInput, output *efs.DescribeReplicationConfigurationsOutput) ([]*sdp.Item, error) {
	if output == nil {
		return nil, errors.New("nil output from AWS")
	}

	items := make([]*sdp.Item, 0)

	for _, replication := range output.Replications {
		attrs, err := adapterhelpers.ToAttributesWithExclude(replication)

		if err != nil {
			return nil, err
		}

		if replication.SourceFileSystemId == nil {
			return nil, errors.New("efs-replication-configuration has nil SourceFileSystemId")
		}

		if replication.SourceFileSystemRegion == nil {
			return nil, errors.New("efs-replication-configuration has nil SourceFileSystemRegion")
		}

		accountID, _, err := adapterhelpers.ParseScope(scope)

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "efs-replication-configuration",
			UniqueAttribute: "SourceFileSystemId",
			Scope:           scope,
			Attributes:      attrs,
			Health:          sdp.Health_HEALTH_OK.Enum(), // Default to OK
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "efs-file-system",
						Method: sdp.QueryMethod_GET,
						Query:  *replication.SourceFileSystemId,
						Scope:  adapterhelpers.FormatScope(accountID, *replication.SourceFileSystemRegion),
					},
				},
			},
		}

		for _, destination := range replication.Destinations {
			if destination.FileSystemId != nil && destination.Region != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "efs-file-system",
						Method: sdp.QueryMethod_GET,
						Query:  *destination.FileSystemId,
						Scope:  adapterhelpers.FormatScope(accountID, *destination.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the destination shouldn't affect the source
						In: false,
						// Changes to this can affect the destination
						Out: true,
					},
				})
			}
		}

		// Set the health to the worst of the statuses
		var hasError bool
		for _, destination := range replication.Destinations {
			switch destination.Status { //nolint:exhaustive // handled by default case
			case types.ReplicationStatusError:
				item.Health = sdp.Health_HEALTH_ERROR.Enum()
				hasError = true
			case types.ReplicationStatusEnabling:
				item.Health = sdp.Health_HEALTH_PENDING.Enum()
			case types.ReplicationStatusDeleting:
				item.Health = sdp.Health_HEALTH_PENDING.Enum()
			case types.ReplicationStatusPausing:
				item.Health = sdp.Health_HEALTH_PENDING.Enum()
			default:
				// If there's no error, we don't need to do anything
			}

			if hasError {
				break
			}
		}

		if replication.OriginalSourceFileSystemArn != nil {
			if arn, err := adapterhelpers.ParseARN(*replication.OriginalSourceFileSystemArn); err == nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "efs-file-system",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *replication.OriginalSourceFileSystemArn,
						Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the source file system will affect its replication
						In: true,
						// Changing replication shouldn't affect the filesystem itself
						Out: false,
					},
				})
			}

		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEFSReplicationConfigurationAdapter(client *efs.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*efs.DescribeReplicationConfigurationsInput, *efs.DescribeReplicationConfigurationsOutput, *efs.Client, *efs.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*efs.DescribeReplicationConfigurationsInput, *efs.DescribeReplicationConfigurationsOutput, *efs.Client, *efs.Options]{
		ItemType:        "efs-replication-configuration",
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		AdapterMetadata: replicationConfigurationAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *efs.Client, input *efs.DescribeReplicationConfigurationsInput) (*efs.DescribeReplicationConfigurationsOutput, error) {
			return client.DescribeReplicationConfigurations(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*efs.DescribeReplicationConfigurationsInput, error) {
			return &efs.DescribeReplicationConfigurationsInput{
				FileSystemId: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*efs.DescribeReplicationConfigurationsInput, error) {
			return &efs.DescribeReplicationConfigurationsInput{}, nil
		},
		OutputMapper: ReplicationConfigurationOutputMapper,
	}
}

var replicationConfigurationAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "efs-replication-configuration",
	DescriptiveName: "EFS Replication Configuration",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a replication configuration by file system ID",
		ListDescription:   "List all replication configurations",
		SearchDescription: "Search for a replication configuration by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_efs_replication_configuration.source_file_system_id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
})
