package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func securityGroupInputMapperGet(scope string, query string) (*ec2.DescribeSecurityGroupsInput, error) {
	return &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{
			query,
		},
	}, nil
}

func securityGroupInputMapperList(scope string) (*ec2.DescribeSecurityGroupsInput, error) {
	return &ec2.DescribeSecurityGroupsInput{}, nil
}

func securityGroupOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeSecurityGroupsInput, output *ec2.DescribeSecurityGroupsOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, securityGroup := range output.SecurityGroups {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(securityGroup, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-security-group",
			UniqueAttribute: "GroupId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(securityGroup.Tags),
		}

		// VPC
		if securityGroup.VpcId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-vpc",
					Method: sdp.QueryMethod_GET,
					Query:  *securityGroup.VpcId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the VPC could affect the security group
					In: true,
					// The security group won't affect the VPC though
					Out: false,
				},
			})
		}

		item.LinkedItemQueries = append(item.LinkedItemQueries, extractLinkedSecurityGroups(securityGroup.IpPermissions, scope)...)
		item.LinkedItemQueries = append(item.LinkedItemQueries, extractLinkedSecurityGroups(securityGroup.IpPermissionsEgress, scope)...)

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2SecurityGroupAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeSecurityGroupsInput, *ec2.DescribeSecurityGroupsOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeSecurityGroupsInput, *ec2.DescribeSecurityGroupsOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-security-group",
		AdapterMetadata: securityGroupAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeSecurityGroupsInput) (*ec2.DescribeSecurityGroupsOutput, error) {
			return client.DescribeSecurityGroups(ctx, input)
		},
		InputMapperGet:  securityGroupInputMapperGet,
		InputMapperList: securityGroupInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeSecurityGroupsInput) adapterhelpers.Paginator[*ec2.DescribeSecurityGroupsOutput, *ec2.Options] {
			return ec2.NewDescribeSecurityGroupsPaginator(client, params)
		},
		OutputMapper: securityGroupOutputMapper,
		InputMapperSearch: func(ctx context.Context, client *ec2.Client, scope, query string) (*ec2.DescribeSecurityGroupsInput, error) {
			return &ec2.DescribeSecurityGroupsInput{
				GroupNames: []string{query},
			}, nil
		},
	}
}

var securityGroupAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-security-group",
	DescriptiveName: "Security Group",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a security group by ID",
		ListDescription:   "List all security groups",
		SearchDescription: "Search for security groups by ARN",
	},
	PotentialLinks: []string{"ec2-vpc"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_security_group.id"},
		{TerraformQueryMap: "aws_security_group_rule.security_group_id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})

// extractLinkedSecurityGroups Extracts related security groups from IP
// permissions
func extractLinkedSecurityGroups(permissions []types.IpPermission, scope string) []*sdp.LinkedItemQuery {
	currentAccount, region, err := adapterhelpers.ParseScope(scope)
	requests := make([]*sdp.LinkedItemQuery, 0)
	var relatedAccount string

	if err != nil {
		return requests
	}

	for _, permission := range permissions {
		for _, idGroup := range permission.UserIdGroupPairs {
			if idGroup.UserId != nil {
				relatedAccount = *idGroup.UserId
			} else {
				relatedAccount = currentAccount
			}

			if idGroup.GroupId != nil {
				requests = append(requests, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-security-group",
						Method: sdp.QueryMethod_GET,
						Query:  *idGroup.GroupId,
						Scope:  adapterhelpers.FormatScope(relatedAccount, region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Linked security groups affect each other
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	return requests
}
