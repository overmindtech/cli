package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func placementGroupInputMapperGet(scope string, query string) (*ec2.DescribePlacementGroupsInput, error) {
	return &ec2.DescribePlacementGroupsInput{
		GroupIds: []string{
			query,
		},
	}, nil
}

func placementGroupInputMapperList(scope string) (*ec2.DescribePlacementGroupsInput, error) {
	return &ec2.DescribePlacementGroupsInput{}, nil
}

func placementGroupOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribePlacementGroupsInput, output *ec2.DescribePlacementGroupsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, ng := range output.PlacementGroups {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(ng, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-placement-group",
			UniqueAttribute: "GroupId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(ng.Tags),
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2PlacementGroupAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribePlacementGroupsInput, *ec2.DescribePlacementGroupsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribePlacementGroupsInput, *ec2.DescribePlacementGroupsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-placement-group",
		AdapterMetadata: placementGroupAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribePlacementGroupsInput) (*ec2.DescribePlacementGroupsOutput, error) {
			return client.DescribePlacementGroups(ctx, input)
		},
		InputMapperGet:  placementGroupInputMapperGet,
		InputMapperList: placementGroupInputMapperList,
		OutputMapper:    placementGroupOutputMapper,
	}
}

var placementGroupAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-placement-group",
	DescriptiveName: "Placement Group",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a placement group by ID",
		ListDescription:   "List all placement groups",
		SearchDescription: "Search for placement groups by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_placement_group.id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
})
