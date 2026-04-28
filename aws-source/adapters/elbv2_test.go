package adapters

import (
	"context"
	"fmt"
	"testing"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/overmindtech/cli/go/sdp-go"
)

type mockElbv2Client struct {
	rejectOver20 bool
}

func (m mockElbv2Client) DescribeTags(ctx context.Context, params *elbv2.DescribeTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error) {
	if m.rejectOver20 && len(params.ResourceArns) > elbv2DescribeTagsMaxItems {
		return nil, fmt.Errorf("cannot describe more than %d ELBv2 resources, got %d", elbv2DescribeTagsMaxItems, len(params.ResourceArns))
	}

	tagDescriptions := make([]types.TagDescription, 0, len(params.ResourceArns))

	for _, arn := range params.ResourceArns {
		tagDescriptions = append(tagDescriptions, types.TagDescription{
			ResourceArn: &arn,
			Tags: []types.Tag{
				{
					Key:   new("foo"),
					Value: new("bar"),
				},
			},
		})
	}

	return &elbv2.DescribeTagsOutput{
		TagDescriptions: tagDescriptions,
	}, nil
}

func (m mockElbv2Client) DescribeLoadBalancers(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
	return nil, nil
}

func (m mockElbv2Client) DescribeListeners(ctx context.Context, params *elbv2.DescribeListenersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
	return nil, nil
}

func (m mockElbv2Client) DescribeRules(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error) {
	return nil, nil
}

func (m mockElbv2Client) DescribeTargetGroups(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
	return nil, nil
}

func TestElbv2GetTagsMapBatching(t *testing.T) {
	client := &mockElbv2Client{rejectOver20: true}

	arns := []string{
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-00/0000000000000000",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-01/0000000000000001",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-02/0000000000000002",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-03/0000000000000003",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-04/0000000000000004",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-05/0000000000000005",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-06/0000000000000006",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-07/0000000000000007",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-08/0000000000000008",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-09/0000000000000009",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-10/0000000000000010",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-11/0000000000000011",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-12/0000000000000012",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-13/0000000000000013",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-14/0000000000000014",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-15/0000000000000015",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-16/0000000000000016",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-17/0000000000000017",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-18/0000000000000018",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-19/0000000000000019",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-20/0000000000000020",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-21/0000000000000021",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-22/0000000000000022",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-23/0000000000000023",
		"arn:aws:elasticloadbalancing:eu-west-2:123456789012:loadbalancer/app/lb-24/0000000000000024",
	}

	tagsMap := elbv2GetTagsMap(context.Background(), client, arns)

	if len(tagsMap) != 25 {
		t.Fatalf("expected 25 tagged resources, got %d", len(tagsMap))
	}

	for _, arn := range arns {
		if got := tagsMap[arn]["foo"]; got != "bar" {
			t.Errorf("expected tag foo=bar for %q, got %q", arn, got)
		}
	}
}

func TestActionToRequests(t *testing.T) {
	action := types.Action{
		Type:  types.ActionTypeEnumFixedResponse,
		Order: new(int32(1)),
		FixedResponseConfig: &types.FixedResponseActionConfig{
			StatusCode:  new("404"),
			ContentType: new("text/plain"),
			MessageBody: new("not found"),
		},
		AuthenticateCognitoConfig: &types.AuthenticateCognitoActionConfig{
			UserPoolArn:      new("arn:partition:service:region:account-id:resource-type:resource-id"), // link
			UserPoolClientId: new("clientID"),
			UserPoolDomain:   new("domain.com"),
			AuthenticationRequestExtraParams: map[string]string{
				"foo": "bar",
			},
			OnUnauthenticatedRequest: types.AuthenticateCognitoActionConditionalBehaviorEnumAuthenticate,
			Scope:                    new("foo"),
			SessionCookieName:        new("cookie"),
			SessionTimeout:           new(int64(10)),
		},
		AuthenticateOidcConfig: &types.AuthenticateOidcActionConfig{
			AuthorizationEndpoint:            new("https://auth.somewhere.com/app1"), // link
			ClientId:                         new("CLIENT-ID"),
			Issuer:                           new("Someone"),
			TokenEndpoint:                    new("https://auth.somewhere.com/app1/tokens"), // link
			UserInfoEndpoint:                 new("https://auth.somewhere.com/app1/users"),  // link
			AuthenticationRequestExtraParams: map[string]string{},
			ClientSecret:                     new("secret"), // Redact
			OnUnauthenticatedRequest:         types.AuthenticateOidcActionConditionalBehaviorEnumAllow,
			Scope:                            new("foo"),
			SessionCookieName:                new("cookie"),
			SessionTimeout:                   new(int64(10)),
			UseExistingClientSecret:          new(true),
		},
		ForwardConfig: &types.ForwardActionConfig{
			TargetGroupStickinessConfig: &types.TargetGroupStickinessConfig{
				DurationSeconds: new(int32(10)),
				Enabled:         new(true),
			},
			TargetGroups: []types.TargetGroupTuple{
				{
					TargetGroupArn: new("arn:partition:service:region:account-id:resource-type:resource-id1"), // link
					Weight:         new(int32(1)),
				},
			},
		},
		RedirectConfig: &types.RedirectActionConfig{
			StatusCode: types.RedirectActionStatusCodeEnumHttp302,
			Host:       new("somewhere.else.com"), // combine and link
			Path:       new("/login"),             // combine and link
			Port:       new("8080"),               // combine and link
			Protocol:   new("https"),              // combine and link
			Query:      new("foo=bar"),            // combine and link
		},
		TargetGroupArn: new("arn:partition:service:region:account-id:resource-type:resource-id2"), // link
	}

	item := sdp.Item{
		Type:              "test",
		UniqueAttribute:   "foo",
		Attributes:        &sdp.ItemAttributes{},
		Scope:             "foo",
		LinkedItemQueries: ActionToRequests(action),
	}

	tests := QueryTests{
		{
			ExpectedType:   "cognito-idp-user-pool",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:partition:service:region:account-id:resource-type:resource-id",
			ExpectedScope:  "account-id.region",
		},
		{
			ExpectedType:   "http",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "https://auth.somewhere.com/app1",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "http",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "https://auth.somewhere.com/app1/tokens",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "http",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "https://auth.somewhere.com/app1/users",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "elbv2-target-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:partition:service:region:account-id:resource-type:resource-id1",
			ExpectedScope:  "account-id.region",
		},
		{
			ExpectedType:   "http",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "https://somewhere.else.com:8080/login?foo=bar",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "elbv2-target-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:partition:service:region:account-id:resource-type:resource-id2",
			ExpectedScope:  "account-id.region",
		},
	}

	tests.Execute(t, &item)
}
