package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func optionGroupOutputMapper(ctx context.Context, client rdsClient, scope string, _ *rds.DescribeOptionGroupsInput, output *rds.DescribeOptionGroupsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, group := range output.OptionGroupsList {
		var tags map[string]string

		// Get tags
		tagsOut, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
			ResourceName: group.OptionGroupArn,
		})

		if err == nil {
			tags = rdsTagsToMap(tagsOut.TagList)
		} else {
			tags = adapterhelpers.HandleTagsError(ctx, err)
		}

		attributes, err := adapterhelpers.ToAttributesWithExclude(group)

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "rds-option-group",
			UniqueAttribute: "OptionGroupName",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            tags,
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewRDSOptionGroupAdapter(client rdsClient, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*rds.DescribeOptionGroupsInput, *rds.DescribeOptionGroupsOutput, rdsClient, *rds.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*rds.DescribeOptionGroupsInput, *rds.DescribeOptionGroupsOutput, rdsClient, *rds.Options]{
		ItemType:        "rds-option-group",
		Region:          region,
		AccountID:       accountID,
		Client:          client,
		AdapterMetadata: optionGroupAdapterMetadata,
		PaginatorBuilder: func(client rdsClient, params *rds.DescribeOptionGroupsInput) adapterhelpers.Paginator[*rds.DescribeOptionGroupsOutput, *rds.Options] {
			return rds.NewDescribeOptionGroupsPaginator(client, params)
		},
		DescribeFunc: func(ctx context.Context, client rdsClient, input *rds.DescribeOptionGroupsInput) (*rds.DescribeOptionGroupsOutput, error) {
			return client.DescribeOptionGroups(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*rds.DescribeOptionGroupsInput, error) {
			return &rds.DescribeOptionGroupsInput{
				OptionGroupName: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*rds.DescribeOptionGroupsInput, error) {
			return &rds.DescribeOptionGroupsInput{}, nil
		},
		OutputMapper: optionGroupOutputMapper,
	}
}

var optionGroupAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "rds-option-group",
	DescriptiveName: "RDS Option Group",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an option group by name",
		ListDescription:   "List all RDS option groups",
		SearchDescription: "Search for an option group by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_db_option_group.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
})
