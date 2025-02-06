package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rds"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func dBSubnetGroupOutputMapper(ctx context.Context, client rdsClient, scope string, _ *rds.DescribeDBSubnetGroupsInput, output *rds.DescribeDBSubnetGroupsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, sg := range output.DBSubnetGroups {
		var tags map[string]string

		// Get tags
		tagsOut, err := client.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
			ResourceName: sg.DBSubnetGroupArn,
		})

		if err == nil {
			tags = rdsTagsToMap(tagsOut.TagList)
		} else {
			tags = adapterhelpers.HandleTagsError(ctx, err)
		}

		attributes, err := adapterhelpers.ToAttributesWithExclude(sg)

		if err != nil {
			return nil, err
		}

		item := sdp.Item{
			Type:            "rds-db-subnet-group",
			UniqueAttribute: "DBSubnetGroupName",
			Attributes:      attributes,
			Scope:           scope,
			Tags:            tags,
		}

		var a *adapterhelpers.ARN

		if sg.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *sg.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the VPC can affect the subnet group
					In: true,
					// The subnet group won't affect the VPC
					Out: false,
				},
			})
		}

		for _, subnet := range sg.Subnets {
			if subnet.SubnetIdentifier != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-subnet",
						Method: sdp.QueryMethod_GET,
						Query:  *subnet.SubnetIdentifier,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the subnet can affect the subnet group
						In: true,
						// The subnet group won't affect the subnet
						Out: false,
					},
				})
			}

			if subnet.SubnetOutpost != nil {
				if subnet.SubnetOutpost.Arn != nil {
					if a, err = adapterhelpers.ParseARN(*subnet.SubnetOutpost.Arn); err == nil {
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "outposts-outpost",
								Method: sdp.QueryMethod_SEARCH,
								Query:  *subnet.SubnetOutpost.Arn,
								Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
							},
							BlastPropagation: &sdp.BlastPropagation{
								// Changing the outpost can affect the subnet group
								In: true,
								// The subnet group won't affect the outpost
								Out: false,
							},
						})
					}
				}
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewRDSDBSubnetGroupAdapter(client rdsClient, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*rds.DescribeDBSubnetGroupsInput, *rds.DescribeDBSubnetGroupsOutput, rdsClient, *rds.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*rds.DescribeDBSubnetGroupsInput, *rds.DescribeDBSubnetGroupsOutput, rdsClient, *rds.Options]{
		ItemType:        "rds-db-subnet-group",
		Region:          region,
		AccountID:       accountID,
		Client:          client,
		AdapterMetadata: dbSubnetGroupAdapterMetadata,
		PaginatorBuilder: func(client rdsClient, params *rds.DescribeDBSubnetGroupsInput) adapterhelpers.Paginator[*rds.DescribeDBSubnetGroupsOutput, *rds.Options] {
			return rds.NewDescribeDBSubnetGroupsPaginator(client, params)
		},
		DescribeFunc: func(ctx context.Context, client rdsClient, input *rds.DescribeDBSubnetGroupsInput) (*rds.DescribeDBSubnetGroupsOutput, error) {
			return client.DescribeDBSubnetGroups(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*rds.DescribeDBSubnetGroupsInput, error) {
			return &rds.DescribeDBSubnetGroupsInput{
				DBSubnetGroupName: &query,
			}, nil
		},
		InputMapperList: func(scope string) (*rds.DescribeDBSubnetGroupsInput, error) {
			return &rds.DescribeDBSubnetGroupsInput{}, nil
		},
		OutputMapper: dBSubnetGroupOutputMapper,
	}
}

var dbSubnetGroupAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "rds-db-subnet-group",
	DescriptiveName: "RDS Subnet Group",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a subnet group by name",
		ListDescription:   "List all subnet groups",
		SearchDescription: "Search for subnet groups by ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_db_subnet_group.arn",
			TerraformMethod:   sdp.QueryMethod_SEARCH,
		},
	},
	PotentialLinks: []string{"ec2-vpc", "ec2-subnet", "outposts-outpost"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})
