package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func launchTemplateInputMapperGet(scope string, query string) (*ec2.DescribeLaunchTemplatesInput, error) {
	return &ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateIds: []string{
			query,
		},
	}, nil
}

func launchTemplateInputMapperList(scope string) (*ec2.DescribeLaunchTemplatesInput, error) {
	return &ec2.DescribeLaunchTemplatesInput{}, nil
}

func launchTemplateOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeLaunchTemplatesInput, output *ec2.DescribeLaunchTemplatesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, LaunchTemplate := range output.LaunchTemplates {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(LaunchTemplate, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-launch-template",
			UniqueAttribute: "LaunchTemplateId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(LaunchTemplate.Tags),
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2LaunchTemplateAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeLaunchTemplatesInput, *ec2.DescribeLaunchTemplatesOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeLaunchTemplatesInput, *ec2.DescribeLaunchTemplatesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-launch-template",
		AdapterMetadata: launchTemplateAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeLaunchTemplatesInput) (*ec2.DescribeLaunchTemplatesOutput, error) {
			return client.DescribeLaunchTemplates(ctx, input)
		},
		InputMapperGet:  launchTemplateInputMapperGet,
		InputMapperList: launchTemplateInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeLaunchTemplatesInput) adapterhelpers.Paginator[*ec2.DescribeLaunchTemplatesOutput, *ec2.Options] {
			return ec2.NewDescribeLaunchTemplatesPaginator(client, params)
		},
		OutputMapper: launchTemplateOutputMapper,
	}
}

var launchTemplateAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-launch-template",
	DescriptiveName: "Launch Template",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a launch template by ID",
		ListDescription:   "List all launch templates",
		SearchDescription: "Search for launch templates by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_launch_template.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
