package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type ParameterGroup struct {
	types.DBParameterGroup

	Parameters []types.Parameter
}

func dBParameterGroupItemMapper(_, scope string, awsItem *ParameterGroup) (*sdp.Item, error) {
	attributes, err := adapterhelpers.ToAttributesWithExclude(awsItem)

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:            "rds-db-parameter-group",
		UniqueAttribute: "DBParameterGroupName",
		Attributes:      attributes,
		Scope:           scope,
	}

	return &item, nil
}

func NewRDSDBParameterGroupAdapter(client rdsClient, accountID string, region string) *adapterhelpers.GetListAdapter[*ParameterGroup, rdsClient, *rds.Options] {
	return &adapterhelpers.GetListAdapter[*ParameterGroup, rdsClient, *rds.Options]{
		ItemType:        "rds-db-parameter-group",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		AdapterMetadata: dbParameterGroupAdapterMetadata,
		GetFunc: func(ctx context.Context, client rdsClient, scope, query string) (*ParameterGroup, error) {
			out, err := client.DescribeDBParameterGroups(ctx, &rds.DescribeDBParameterGroupsInput{
				DBParameterGroupName: &query,
			})
			if err != nil {
				return nil, err
			}

			if len(out.DBParameterGroups) != 1 {
				return nil, fmt.Errorf("expected 1 group, got %v", len(out.DBParameterGroups))
			}

			paramsOut, err := client.DescribeDBParameters(ctx, &rds.DescribeDBParametersInput{
				DBParameterGroupName: out.DBParameterGroups[0].DBParameterGroupName,
			})
			if err != nil {
				return nil, err
			}

			return &ParameterGroup{
				Parameters:       paramsOut.Parameters,
				DBParameterGroup: out.DBParameterGroups[0],
			}, nil
		},
		ListFunc: func(ctx context.Context, client rdsClient, scope string) ([]*ParameterGroup, error) {
			out, err := client.DescribeDBParameterGroups(ctx, &rds.DescribeDBParameterGroupsInput{})
			if err != nil {
				return nil, err
			}

			groups := make([]*ParameterGroup, 0)

			for _, group := range out.DBParameterGroups {
				paramsOut, err := client.DescribeDBParameters(ctx, &rds.DescribeDBParametersInput{
					DBParameterGroupName: group.DBParameterGroupName,
				})
				if err != nil {
					return nil, err
				}

				groups = append(groups, &ParameterGroup{
					Parameters:       paramsOut.Parameters,
					DBParameterGroup: group,
				})
			}

			return groups, nil
		},
		ListTagsFunc: func(ctx context.Context, pg *ParameterGroup, c rdsClient) (map[string]string, error) {
			out, err := c.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: pg.DBParameterGroupArn,
			})
			if err != nil {
				return nil, err
			}

			return rdsTagsToMap(out.TagList), nil
		},
		ItemMapper: dBParameterGroupItemMapper,
	}
}

var dbParameterGroupAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "rds-db-parameter-group",
	DescriptiveName: "RDS Parameter Group",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a parameter group by name",
		ListDescription:   "List all parameter groups",
		SearchDescription: "Search for a parameter group by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "aws_db_parameter_group.arn",
		},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
})
