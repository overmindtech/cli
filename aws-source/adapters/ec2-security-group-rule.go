package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func securityGroupRuleInputMapperGet(scope string, query string) (*ec2.DescribeSecurityGroupRulesInput, error) {
	return &ec2.DescribeSecurityGroupRulesInput{
		SecurityGroupRuleIds: []string{
			query,
		},
	}, nil
}

func securityGroupRuleInputMapperList(scope string) (*ec2.DescribeSecurityGroupRulesInput, error) {
	return &ec2.DescribeSecurityGroupRulesInput{}, nil
}

func securityGroupRuleOutputMapper(_ context.Context, _ *ec2.Client, scope string, _ *ec2.DescribeSecurityGroupRulesInput, output *ec2.DescribeSecurityGroupRulesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	for _, securityGroupRule := range output.SecurityGroupRules {
		var err error
		var attrs *sdp.ItemAttributes
		attrs, err = adapterhelpers.ToAttributesWithExclude(securityGroupRule, "tags")

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		item := sdp.Item{
			Type:            "ec2-security-group-rule",
			UniqueAttribute: "SecurityGroupRuleId",
			Scope:           scope,
			Attributes:      attrs,
			Tags:            ec2TagsToMap(securityGroupRule.Tags),
		}

		if securityGroupRule.GroupId != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ec2-security-group",
					Method: sdp.QueryMethod_GET,
					Query:  *securityGroupRule.GroupId,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are tightly linked
					In:  true,
					Out: true,
				},
			})
		}

		if rg := securityGroupRule.ReferencedGroupInfo; rg != nil {
			if rg.GroupId != nil {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-security-group",
						Method: sdp.QueryMethod_GET,
						Query:  *rg.GroupId,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// These are tightly linked
						In:  true,
						Out: true,
					},
				})
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewEC2SecurityGroupRuleAdapter(client *ec2.Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeSecurityGroupRulesInput, *ec2.DescribeSecurityGroupRulesOutput, *ec2.Client, *ec2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*ec2.DescribeSecurityGroupRulesInput, *ec2.DescribeSecurityGroupRulesOutput, *ec2.Client, *ec2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "ec2-security-group-rule",
		AdapterMetadata: securityGroupRuleAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client *ec2.Client, input *ec2.DescribeSecurityGroupRulesInput) (*ec2.DescribeSecurityGroupRulesOutput, error) {
			return client.DescribeSecurityGroupRules(ctx, input)
		},
		InputMapperGet:  securityGroupRuleInputMapperGet,
		InputMapperList: securityGroupRuleInputMapperList,
		PaginatorBuilder: func(client *ec2.Client, params *ec2.DescribeSecurityGroupRulesInput) adapterhelpers.Paginator[*ec2.DescribeSecurityGroupRulesOutput, *ec2.Options] {
			return ec2.NewDescribeSecurityGroupRulesPaginator(client, params)
		},
		OutputMapper: securityGroupRuleOutputMapper,
	}
}

var securityGroupRuleAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "ec2-security-group-rule",
	DescriptiveName: "Security Group Rule",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get a security group rule by ID",
		ListDescription:   "List all security group rules",
		SearchDescription: "Search security group rules by ARN",
	},
	PotentialLinks: []string{"ec2-security-group"},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_security_group_rule.security_group_rule_id"},
		{TerraformQueryMap: "aws_vpc_security_group_ingress_rule.security_group_rule_id"},
		{TerraformQueryMap: "aws_vpc_security_group_egress_rule.security_group_rule_id"},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})
