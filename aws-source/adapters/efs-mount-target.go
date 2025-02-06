package adapters

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/efs"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func MountTargetOutputMapper(_ context.Context, _ *efs.Client, scope string, input *efs.DescribeMountTargetsInput, output *efs.DescribeMountTargetsOutput) ([]*sdp.Item, error) {
	if output == nil {
		return nil, errors.New("nil output from AWS")
	}

	items := make([]*sdp.Item, 0)

	for _, mt := range output.MountTargets {
		attrs, err := adapterhelpers.ToAttributesWithExclude(mt)

		if err != nil {
			return nil, err
		}

		if mt.MountTargetId == nil {
			return nil, errors.New("efs-mount-target has nil id")
		}

		if mt.FileSystemId == nil {
			return nil, errors.New("efs-mount-target has nil file system ID")
		}

		item := sdp.Item{
			Type:            "efs-mount-target",
			UniqueAttribute: "MountTargetId",
			Scope:           scope,
			Attributes:      attrs,
			Health:          lifeCycleStateToHealth(mt.LifeCycleState),
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "efs-file-system",
						Method: sdp.QueryMethod_GET,
						Query:  *mt.FileSystemId,
						Scope:  scope,
					},
				},
			},
		}

		if mt.SubnetId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-subnet",
					Method: sdp.QueryMethod_GET,
					Query:  *mt.SubnetId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the subnet could affect the mount but no the
					// other way around
					In:  true,
					Out: false,
				},
			})
		}

		if mt.IpAddress != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  *mt.IpAddress,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// IPs are always bidirectional
					In:  true,
					Out: true,
				},
			})
		}

		if mt.NetworkInterfaceId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-network-interface",
					Method: sdp.QueryMethod_GET,
					Query:  *mt.NetworkInterfaceId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Tightly coupled
					In:  true,
					Out: true,
				},
			})
		}

		if mt.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *mt.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the VPC will affect us
					In: true,
					// We can't affect the VPC
					Out: false,
				},
			})
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEFSMountTargetAdapter(client *efs.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*efs.DescribeMountTargetsInput, *efs.DescribeMountTargetsOutput, *efs.Client, *efs.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*efs.DescribeMountTargetsInput, *efs.DescribeMountTargetsOutput, *efs.Client, *efs.Options]{
		ItemType:        "efs-mount-target",
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		AdapterMetadata: efsMountTargetAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *efs.Client, input *efs.DescribeMountTargetsInput) (*efs.DescribeMountTargetsOutput, error) {
			return client.DescribeMountTargets(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*efs.DescribeMountTargetsInput, error) {
			return &efs.DescribeMountTargetsInput{
				MountTargetId: &query,
			}, nil
		},
		// Search by file system ID
		InputMapperSearch: func(ctx context.Context, client *efs.Client, scope, query string) (*efs.DescribeMountTargetsInput, error) {
			return &efs.DescribeMountTargetsInput{
				FileSystemId: &query,
			}, nil
		},
		OutputMapper: MountTargetOutputMapper,
	}
}

var efsMountTargetAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "efs-mount-target",
	DescriptiveName: "EFS Mount Target",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an mount target by ID",
		SearchDescription: "Search for mount targets by file system ID",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_efs_mount_target.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
})
