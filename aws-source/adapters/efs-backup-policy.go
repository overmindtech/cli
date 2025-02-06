package adapters

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/efs"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func BackupPolicyOutputMapper(_ context.Context, _ *efs.Client, scope string, input *efs.DescribeBackupPolicyInput, output *efs.DescribeBackupPolicyOutput) ([]*sdp.Item, error) {
	if output == nil {
		return nil, errors.New("nil output from AWS")
	}

	if output.BackupPolicy == nil {
		return nil, errors.New("output contains no backup policy")
	}

	if input == nil {
		return nil, errors.New("nil input")
	}

	if input.FileSystemId == nil {
		return nil, errors.New("nil filesystem ID on input")
	}

	attrs, err := adapterhelpers.ToAttributesWithExclude(output)

	if err != nil {
		return nil, err
	}

	// Add the filesystem ID as an attribute
	err = attrs.Set("FileSystemId", *input.FileSystemId)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "efs-backup-policy",
		UniqueAttribute: "FileSystemId",
		Scope:           scope,
		Attributes:      attrs,
	}

	return []*sdp.Item{&item}, nil
}

func NewEFSBackupPolicyAdapter(client *efs.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*efs.DescribeBackupPolicyInput, *efs.DescribeBackupPolicyOutput, *efs.Client, *efs.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*efs.DescribeBackupPolicyInput, *efs.DescribeBackupPolicyOutput, *efs.Client, *efs.Options]{
		ItemType:        "efs-backup-policy",
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		AdapterMetadata: backupPolicyAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *efs.Client, input *efs.DescribeBackupPolicyInput) (*efs.DescribeBackupPolicyOutput, error) {
			return client.DescribeBackupPolicy(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*efs.DescribeBackupPolicyInput, error) {
			return &efs.DescribeBackupPolicyInput{
				FileSystemId: &query,
			}, nil
		},
		OutputMapper: BackupPolicyOutputMapper,
	}
}

var backupPolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "efs-backup-policy",
	DescriptiveName: "EFS Backup Policy",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an Backup Policy by file system ID",
		SearchDescription: "Search for an Backup Policy by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_efs_backup_policy.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
})
