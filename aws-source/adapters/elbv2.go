package adapters

import (
	"context"
	"fmt"
	"net/url"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type elbv2Client interface {
	DescribeTags(ctx context.Context, params *elbv2.DescribeTagsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTagsOutput, error)
	DescribeLoadBalancers(ctx context.Context, params *elbv2.DescribeLoadBalancersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
	DescribeListeners(ctx context.Context, params *elbv2.DescribeListenersInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error)
	DescribeRules(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error)
	DescribeTargetGroups(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
}

func elbv2TagsToMap(tags []types.Tag) map[string]string {
	m := make(map[string]string)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			m[*tag.Key] = *tag.Value
		}
	}

	return m
}

// Gets a map of ARN to tags (in map[string]string format) for the given ARNs
func elbv2GetTagsMap(ctx context.Context, client elbv2Client, arns []string) map[string]map[string]string {
	tagsMap := make(map[string]map[string]string)

	if len(arns) > 0 {
		tagsOut, err := client.DescribeTags(ctx, &elbv2.DescribeTagsInput{
			ResourceArns: arns,
		})
		if err != nil {
			tags := adapterhelpers.HandleTagsError(ctx, err)

			// Set these tags for all ARNs
			for _, arn := range arns {
				tagsMap[arn] = tags
			}

			return tagsMap
		}

		for _, tagDescription := range tagsOut.TagDescriptions {
			if tagDescription.ResourceArn != nil {
				tagsMap[*tagDescription.ResourceArn] = elbv2TagsToMap(tagDescription.Tags)
			}
		}
	}

	return tagsMap
}

func ActionToRequests(action types.Action) []*sdp.LinkedItemQuery {
	requests := make([]*sdp.LinkedItemQuery, 0)

	if action.AuthenticateCognitoConfig != nil {
		if action.AuthenticateCognitoConfig.UserPoolArn != nil {
			if a, err := adapterhelpers.ParseARN(*action.AuthenticateCognitoConfig.UserPoolArn); err == nil {
				requests = append(requests, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "cognito-idp-user-pool",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *action.AuthenticateCognitoConfig.UserPoolArn,
						Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the user pool could affect the LB
						In: true,
						// The LB won't affect the user pool
						Out: false,
					},
				})
			}
		}
	}

	if action.AuthenticateOidcConfig != nil {
		if action.AuthenticateOidcConfig.AuthorizationEndpoint != nil {
			requests = append(requests, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  *action.AuthenticateOidcConfig.AuthorizationEndpoint,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the authorization endpoint could affect the LB
					In: true,
					// The LB won't affect the authorization endpoint
					Out: false,
				},
			})
		}

		if action.AuthenticateOidcConfig.TokenEndpoint != nil {
			requests = append(requests, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  *action.AuthenticateOidcConfig.TokenEndpoint,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the authorization endpoint could affect the LB
					In: true,
					// The LB won't affect the authorization endpoint
					Out: false,
				},
			})
		}

		if action.AuthenticateOidcConfig.UserInfoEndpoint != nil {
			requests = append(requests, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  *action.AuthenticateOidcConfig.UserInfoEndpoint,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the authorization endpoint could affect the LB
					In: true,
					// The LB won't affect the authorization endpoint
					Out: false,
				},
			})
		}
	}

	if action.ForwardConfig != nil {
		for _, tg := range action.ForwardConfig.TargetGroups {
			if tg.TargetGroupArn != nil {
				if a, err := adapterhelpers.ParseARN(*tg.TargetGroupArn); err == nil {
					requests = append(requests, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "elbv2-target-group",
							Method: sdp.QueryMethod_SEARCH,
							Query:  *tg.TargetGroupArn,
							Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the target group could affect the LB
							In: true,
							// The LB could also affect the target group
							Out: true,
						},
					})
				}
			}
		}
	}

	if action.RedirectConfig != nil {
		u := url.URL{}

		if action.RedirectConfig.Path != nil {
			u.Path = *action.RedirectConfig.Path
		}

		if action.RedirectConfig.Port != nil {
			u.Port()
		}

		if action.RedirectConfig.Host != nil {
			u.Host = *action.RedirectConfig.Host

			if action.RedirectConfig.Port != nil {
				u.Host = u.Host + fmt.Sprintf(":%v", *action.RedirectConfig.Port)
			}
		}

		if action.RedirectConfig.Protocol != nil {
			u.Scheme = *action.RedirectConfig.Protocol
		}

		if action.RedirectConfig.Query != nil {
			u.RawQuery = *action.RedirectConfig.Query
		}

		if u.Scheme == "http" || u.Scheme == "https" {
			requests = append(requests, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  u.String(),
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are closely linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	if action.TargetGroupArn != nil {
		if a, err := adapterhelpers.ParseARN(*action.TargetGroupArn); err == nil {
			requests = append(requests, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "elbv2-target-group",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *action.TargetGroupArn,
					Scope:  adapterhelpers.FormatScope(a.AccountID, a.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// These are closely linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	return requests
}
