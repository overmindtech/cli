package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type ClusterParameterGroup struct {
	types.DBClusterParameterGroup

	Parameters []types.Parameter
}

func dBClusterParameterGroupItemMapper(_, scope string, awsItem *ClusterParameterGroup) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "rds-db-cluster-parameter-group",
		UniqueAttribute: "DBClusterParameterGroupName",
		Attributes:      attributes,
		Scope:           scope,
	}

	return &item, nil
}

func NewRDSDBClusterParameterGroupAdapter(client rdsClient, accountID string, region string) *adapterhelpers.GetListAdapter[*ClusterParameterGroup, rdsClient, *rds.Options] {
	return &adapterhelpers.GetListAdapter[*ClusterParameterGroup, rdsClient, *rds.Options]{
		ItemType:        "rds-db-cluster-parameter-group",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: dbClusterParameterGroupAdapterMetadata,
		GetFunc: func(ctx context.Context, client rdsClient, scope, query string) (*ClusterParameterGroup, error) {
			out, err := client.DescribeDBClusterParameterGroups(ctx, &rds.DescribeDBClusterParameterGroupsInput{
				DBClusterParameterGroupName: &query,
			})

			if err != nil {
				return nil, err
			}

			if len(out.DBClusterParameterGroups) != 1 {
				return nil, fmt.Errorf("expected 1 group, got %v", len(out.DBClusterParameterGroups))
			}

			paramsOut, err := client.DescribeDBClusterParameters(ctx, &rds.DescribeDBClusterParametersInput{
				DBClusterParameterGroupName: out.DBClusterParameterGroups[0].DBClusterParameterGroupName,
			})

			if err != nil {
				return nil, err
			}

			return &ClusterParameterGroup{
				Parameters:              paramsOut.Parameters,
				DBClusterParameterGroup: out.DBClusterParameterGroups[0],
			}, nil
		},
		ListFunc: func(ctx context.Context, client rdsClient, scope string) ([]*ClusterParameterGroup, error) {
			out, err := client.DescribeDBClusterParameterGroups(ctx, &rds.DescribeDBClusterParameterGroupsInput{})

			if err != nil {
				return nil, err
			}

			groups := make([]*ClusterParameterGroup, 0)

			for _, group := range out.DBClusterParameterGroups {
				paramsOut, err := client.DescribeDBClusterParameters(ctx, &rds.DescribeDBClusterParametersInput{
					DBClusterParameterGroupName: group.DBClusterParameterGroupName,
				})

				if err != nil {
					return nil, err
				}

				groups = append(groups, &ClusterParameterGroup{
					Parameters:              paramsOut.Parameters,
					DBClusterParameterGroup: group,
				})
			}

			return groups, nil
		},
		ListTagsFunc: func(ctx context.Context, cpg *ClusterParameterGroup, c rdsClient) (map[string]string, error) {
			out, err := c.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: cpg.DBClusterParameterGroupArn,
			})

			if err != nil {
				return nil, err
			}

			return rdsTagsToMap(out.TagList), nil
		},
		ItemMapper: dBClusterParameterGroupItemMapper,
	}
}

var dbClusterParameterGroupAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "rds-db-cluster-parameter-group",
	DescriptiveName: "RDS Cluster Parameter Group",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a parameter group by name",
		ListDescription:   "List all RDS parameter groups",
		SearchDescription: "Search for a parameter group by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_rds_cluster_parameter_group.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
})
