package adapters

import (
	"context"
	"testing"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/overmindtech/cli/sdp-go"
)

type mockElbv2Client struct{}

func (m mockElbv2Client) DescribeTags(ctx context.Context, params *elbv2.DescribeTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error) {
	tagDescriptions := make([]types.TagDescription, 0)

	for _, arn := range params.ResourceArns {
		tagDescriptions = append(tagDescriptions, types.TagDescription{
			ResourceArn: &arn,
			Tags: []types.Tag{
				{
					Key:   PtrString("foo"),
					Value: PtrString("bar"),
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
		Order: PtrInt32(1),
		FixedResponseConfig: &types.FixedResponseActionConfig{
			StatusCode:  PtrString("404"),
			ContentType: PtrString("text/plain"),
			MessageBody: PtrString("not found"),
		},
		AuthenticateCognitoConfig: &types.AuthenticateCognitoActionConfig{
			UserPoolArn:      PtrString("arn:partition:service:region:account-id:resource-type:resource-id"), // link
			UserPoolClientId: PtrString("clientID"),
			UserPoolDomain:   PtrString("domain.com"),
			AuthenticationRequestExtraParams: map[string]string{
				"foo": "bar",
			},
			OnUnauthenticatedRequest: types.AuthenticateCognitoActionConditionalBehaviorEnumAuthenticate,
			Scope:                    PtrString("foo"),
			SessionCookieName:        PtrString("cookie"),
			SessionTimeout:           PtrInt64(10),
		},
		AuthenticateOidcConfig: &types.AuthenticateOidcActionConfig{
			AuthorizationEndpoint:            PtrString("https://auth.somewhere.com/app1"), // link
			ClientId:                         PtrString("CLIENT-ID"),
			Issuer:                           PtrString("Someone"),
			TokenEndpoint:                    PtrString("https://auth.somewhere.com/app1/tokens"), // link
			UserInfoEndpoint:                 PtrString("https://auth.somewhere.com/app1/users"),  // link
			AuthenticationRequestExtraParams: map[string]string{},
			ClientSecret:                     PtrString("secret"), // Redact
			OnUnauthenticatedRequest:         types.AuthenticateOidcActionConditionalBehaviorEnumAllow,
			Scope:                            PtrString("foo"),
			SessionCookieName:                PtrString("cookie"),
			SessionTimeout:                   PtrInt64(10),
			UseExistingClientSecret:          PtrBool(true),
		},
		ForwardConfig: &types.ForwardActionConfig{
			TargetGroupStickinessConfig: &types.TargetGroupStickinessConfig{
				DurationSeconds: PtrInt32(10),
				Enabled:         PtrBool(true),
			},
			TargetGroups: []types.TargetGroupTuple{
				{
					TargetGroupArn: PtrString("arn:partition:service:region:account-id:resource-type:resource-id1"), // link
					Weight:         PtrInt32(1),
				},
			},
		},
		RedirectConfig: &types.RedirectActionConfig{
			StatusCode: types.RedirectActionStatusCodeEnumHttp302,
			Host:       PtrString("somewhere.else.com"), // combine and link
			Path:       PtrString("/login"),             // combine and link
			Port:       PtrString("8080"),               // combine and link
			Protocol:   PtrString("https"),              // combine and link
			Query:      PtrString("foo=bar"),            // combine and link
		},
		TargetGroupArn: PtrString("arn:partition:service:region:account-id:resource-type:resource-id2"), // link
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
