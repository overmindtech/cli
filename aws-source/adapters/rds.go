package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

type rdsClient interface {
	DescribeDBClusterParameterGroups(ctx context.Context, params *rds.DescribeDBClusterParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterParameterGroupsOutput, error)
	DescribeDBClusterParameters(ctx context.Context, params *rds.DescribeDBClusterParametersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterParametersOutput, error)
	DescribeDBParameterGroups(ctx context.Context, params *rds.DescribeDBParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error)
	DescribeDBParameters(ctx context.Context, params *rds.DescribeDBParametersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParametersOutput, error)
	ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error)
	DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)
	DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
	DescribeDBSubnetGroups(ctx context.Context, params *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error)
	DescribeOptionGroups(ctx context.Context, params *rds.DescribeOptionGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeOptionGroupsOutput, error)
}

type mockRdsClient struct{}

func (m mockRdsClient) DescribeDBClusterParameterGroups(ctx context.Context, params *rds.DescribeDBClusterParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterParameterGroupsOutput, error) {
	return nil, nil
}

func (m mockRdsClient) DescribeDBClusterParameters(ctx context.Context, params *rds.DescribeDBClusterParametersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterParametersOutput, error) {
	return nil, nil
}

func (m mockRdsClient) ListTagsForResource(ctx context.Context, params *rds.ListTagsForResourceInput, optFns ...func(*rds.Options)) (*rds.ListTagsForResourceOutput, error) {
	return &rds.ListTagsForResourceOutput{
		TagList: []types.Tag{
			{
				Key:   adapterhelpers.PtrString("key"),
				Value: adapterhelpers.PtrString("value"),
			},
		},
	}, nil
}

func (m mockRdsClient) DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
	return nil, nil
}

func (m mockRdsClient) DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	return nil, nil
}

func (m mockRdsClient) DescribeDBSubnetGroups(ctx context.Context, params *rds.DescribeDBSubnetGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSubnetGroupsOutput, error) {
	return nil, nil
}

func (m mockRdsClient) DescribeOptionGroups(ctx context.Context, params *rds.DescribeOptionGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeOptionGroupsOutput, error) {
	return nil, nil
}

func (m mockRdsClient) DescribeDBParameterGroups(ctx context.Context, params *rds.DescribeDBParameterGroupsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParameterGroupsOutput, error) {
	return nil, nil
}

func (m mockRdsClient) DescribeDBParameters(ctx context.Context, params *rds.DescribeDBParametersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBParametersOutput, error) {
	return nil, nil
}

func rdsTagsToMap(tags []types.Tag) map[string]string {
	tagsMap := make(map[string]string)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			tagsMap[*tag.Key] = *tag.Value
		}
	}

	return tagsMap
}
