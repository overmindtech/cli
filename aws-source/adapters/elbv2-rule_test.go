package adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

func TestRuleOutputMapper(t *testing.T) {
	output := elbv2.DescribeRulesOutput{
		Rules: []types.Rule{
			{
				RuleArn:  adapterhelpers.PtrString("arn:aws:elasticloadbalancing:eu-west-2:944651592624:listener-rule/app/ingress/1bf10920c5bd199d/9d28f512be129134/0f73a74d21b008f7"),
				Priority: adapterhelpers.PtrString("1"),
				Conditions: []types.RuleCondition{
					{
						Field: adapterhelpers.PtrString("path-pattern"),
						Values: []string{
							"/api/gateway",
						},
						PathPatternConfig: &types.PathPatternConditionConfig{
							Values: []string{
								"/api/gateway",
							},
						},
						HostHeaderConfig: &types.HostHeaderConditionConfig{
							Values: []string{
								"foo.bar.com", // link
							},
						},
						HttpHeaderConfig: &types.HttpHeaderConditionConfig{
							HttpHeaderName: adapterhelpers.PtrString("SOMETHING"),
							Values: []string{
								"foo",
							},
						},
						HttpRequestMethodConfig: &types.HttpRequestMethodConditionConfig{
							Values: []string{
								"GET",
							},
						},
						QueryStringConfig: &types.QueryStringConditionConfig{
							Values: []types.QueryStringKeyValuePair{
								{
									Key:   adapterhelpers.PtrString("foo"),
									Value: adapterhelpers.PtrString("bar"),
								},
							},
						},
						SourceIpConfig: &types.SourceIpConditionConfig{
							Values: []string{
								"1.1.1.1/24",
							},
						},
					},
				},
				Actions: []types.Action{
					// Tested in actions.go
				},
				IsDefault: adapterhelpers.PtrBool(false),
			},
		},
	}

	items, err := ruleOutputMapper(context.Background(), mockElbv2Client{}, "foo", nil, &output)
	if err != nil {
		t.Error(err)
	}

	if len(items) != 1 {
		t.Error("expected 1 item")
	}

	item := items[0]

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "foo.bar.com",
			ExpectedScope:  "global",
		},
	}

	tests.Execute(t, item)
}

func TestNewELBv2RuleAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := elbv2.NewFromConfig(config)

	lbSource := NewELBv2LoadBalancerAdapter(client, account, region)
	listenerSource := NewELBv2ListenerAdapter(client, account, region)
	ruleSource := NewELBv2RuleAdapter(client, account, region)

	stream := discovery.NewRecordingQueryResultStream()
	lbSource.ListStream(context.Background(), lbSource.Scopes()[0], false, stream)

	errs := stream.GetErrors()
	if len(errs) > 0 {
		t.Error(errs)
	}

	items := stream.GetItems()
	if len(items) == 0 {
		t.Skip("no load balancers found")
	}

	lbARN, err := items[0].GetAttributes().Get("LoadBalancerArn")
	if err != nil {
		t.Fatal(err)
	}

	stream = discovery.NewRecordingQueryResultStream()
	listenerSource.SearchStream(context.Background(), listenerSource.Scopes()[0], fmt.Sprint(lbARN), false, stream)

	errs = stream.GetErrors()
	if len(errs) > 0 {
		t.Error(errs)
	}

	items = stream.GetItems()
	if len(items) == 0 {
		t.Skip("no listeners found")
	}

	listenerARN, err := items[0].GetAttributes().Get("ListenerArn")
	if err != nil {
		t.Fatal(err)
	}

	goodSearch := fmt.Sprint(listenerARN)

	test := adapterhelpers.E2ETest{
		Adapter:         ruleSource,
		Timeout:         10 * time.Second,
		GoodSearchQuery: &goodSearch,
		SkipList:        true,
	}

	test.Run(t)
}
