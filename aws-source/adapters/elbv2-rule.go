package adapters

import (
	"context"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func ruleOutputMapper(ctx context.Context, client elbv2Client, scope string, _ *elbv2.DescribeRulesInput, output *elbv2.DescribeRulesOutput) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	ruleArns := make([]string, 0)

	for _, rule := range output.Rules {
		if rule.RuleArn != nil {
			ruleArns = append(ruleArns, *rule.RuleArn)
		}
	}

	tagsMap := elbv2GetTagsMap(ctx, client, ruleArns)

	for _, rule := range output.Rules {
		attrs, err := adapterhelpers.ToAttributesWithExclude(rule)

		if err != nil {
			return nil, err
		}

		var tags map[string]string

		if rule.RuleArn != nil {
			tags = tagsMap[*rule.RuleArn]
		}

		item := sdp.Item{
			Type:            "elbv2-rule",
			UniqueAttribute: "RuleArn",
			Attributes:      attrs,
			Scope:           scope,
			Tags:            tags,
		}

		var requests []*sdp.LinkedItemQuery

		for _, action := range rule.Actions {
			requests = ActionToRequests(action)
			item.LinkedItemQueries = append(item.LinkedItemQueries, requests...)
		}

		for _, condition := range rule.Conditions {
			if condition.HostHeaderConfig != nil {
				for _, value := range condition.HostHeaderConfig.Values {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "dns",
							Method: sdp.QueryMethod_SEARCH,
							Query:  value,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Tightly coupled
							In:  true,
							Out: true,
						},
					})
				}
			}
		}

		items = append(items, &item)
	}

	return items, nil
}

func NewELBv2RuleAdapter(client elbv2Client, accountID string, region string) *adapterhelpers.DescribeOnlyAdapter[*elbv2.DescribeRulesInput, *elbv2.DescribeRulesOutput, elbv2Client, *elbv2.Options] {
	return &adapterhelpers.DescribeOnlyAdapter[*elbv2.DescribeRulesInput, *elbv2.DescribeRulesOutput, elbv2Client, *elbv2.Options]{
		Region:          region,
		Client:          client,
		AccountID:       accountID,
		ItemType:        "elbv2-rule",
		AdapterMetadata: ruleAdapterMetadata,
		DescribeFunc: func(ctx context.Context, client elbv2Client, input *elbv2.DescribeRulesInput) (*elbv2.DescribeRulesOutput, error) {
			return client.DescribeRules(ctx, input)
		},
		InputMapperGet: func(scope, query string) (*elbv2.DescribeRulesInput, error) {
			return &elbv2.DescribeRulesInput{
				RuleArns: []string{query},
			}, nil
		},
		InputMapperList: func(scope string) (*elbv2.DescribeRulesInput, error) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "list not supported for elbv2-rule, use search",
			}
		},
		InputMapperSearch: func(ctx context.Context, client elbv2Client, scope, query string) (*elbv2.DescribeRulesInput, error) {
			// Search by listener ARN
			return &elbv2.DescribeRulesInput{
				ListenerArn: &query,
			}, nil
		},
		OutputMapper: ruleOutputMapper,
	}
}

var ruleAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "elbv2-rule",
	DescriptiveName: "ELB Rule",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get a rule by ARN",
		SearchDescription: "Search for rules by listener ARN",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformQueryMap: "aws_alb_listener_rule.arn",
			TerraformMethod:   sdp.QueryMethod_GET,
		},
		{
			TerraformQueryMap: "aws_lb_listener_rule.arn",
			TerraformMethod:   sdp.QueryMethod_GET,
		},
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})
