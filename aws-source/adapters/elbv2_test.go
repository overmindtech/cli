package adapters

import (
	"context"
	"testing"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/overmindtech/cli/go/sdp-go"
)

type mockElbv2Client struct{}

func (m mockElbv2Client) DescribeTags(ctx context.Context, params *elbv2.DescribeTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error) {
	tagDescriptions := make([]types.TagDescription, 0)

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
