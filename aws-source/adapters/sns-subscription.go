package adapters

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

type subsCli interface {
	GetSubscriptionAttributes(ctx context.Context, params *sns.GetSubscriptionAttributesInput, optFns ...func(*sns.Options)) (*sns.GetSubscriptionAttributesOutput, error)
	ListSubscriptions(context.Context, *sns.ListSubscriptionsInput, ...func(*sns.Options)) (*sns.ListSubscriptionsOutput, error)
	ListTagsForResource(context.Context, *sns.ListTagsForResourceInput, ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error)
}

func getSubsFunc(ctx context.Context, client subsCli, scope string, input *sns.GetSubscriptionAttributesInput) (*sdp.Item, error) {
	output, err := client.GetSubscriptionAttributes(ctx, input)
	if err != nil {
		return nil, err
	}

	if output.Attributes == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "get subscription attributes response was nil",
		}
	}

	attributes, err := adapterhelpers.ToAttributesWithExclude(output.Attributes)
	if err != nil {
		return nil, err
	}

	item := &sdp.Item{
		Type:            "sns-subscription",
		UniqueAttribute: "SubscriptionArn",
		Attributes:      attributes,
		Scope:           scope,
	}

	if resourceTags, err := tagsByResourceARN(ctx, client, *input.SubscriptionArn); err == nil {
		item.Tags = tagsToMap(resourceTags)
	}

	if topicArn, err := attributes.Get("topicArn"); err == nil {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "sns-topic",
				Method: sdp.QueryMethod_GET,
				Query:  topicArn.(string),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// If topic is not healthy, subscription will not work
				In: true,
				// Subscription won't affect the topic
				Out: false,
			},
		})
	}

	if subsRoleArn, err := attributes.Get("subscriptionRoleArn"); err == nil {
		if arn, err := adapterhelpers.ParseARN(fmt.Sprint(subsRoleArn)); err == nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "iam-role",
					Method: sdp.QueryMethod_GET,
					Query:  arn.ResourceID(),
					Scope:  adapterhelpers.FormatScope(arn.AccountID, arn.Region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// If role is not healthy, subscription will not work
					In: true,
					// Subscription won't affect the role
					Out: false,
				},
			})
		}
	}

	return item, nil
}

func NewSNSSubscriptionAdapter(client subsCli, accountID string, region string) *adapterhelpers.AlwaysGetAdapter[*sns.ListSubscriptionsInput, *sns.ListSubscriptionsOutput, *sns.GetSubscriptionAttributesInput, *sns.GetSubscriptionAttributesOutput, subsCli, *sns.Options] {
	return &adapterhelpers.AlwaysGetAdapter[*sns.ListSubscriptionsInput, *sns.ListSubscriptionsOutput, *sns.GetSubscriptionAttributesInput, *sns.GetSubscriptionAttributesOutput, subsCli, *sns.Options]{
		ItemType:        "sns-subscription",
		Client:          client,
		AccountID:       accountID,
		Region:          region,
		ListInput:       &sns.ListSubscriptionsInput{},
		AdapterMetadata: snsSubscriptionAdapterMetadata,
		GetInputMapper: func(scope, query string) *sns.GetSubscriptionAttributesInput {
			return &sns.GetSubscriptionAttributesInput{
				SubscriptionArn: &query,
			}
		},
		ListFuncPaginatorBuilder: func(client subsCli, input *sns.ListSubscriptionsInput) adapterhelpers.Paginator[*sns.ListSubscriptionsOutput, *sns.Options] {
			return sns.NewListSubscriptionsPaginator(client, input)
		},
		ListFuncOutputMapper: func(output *sns.ListSubscriptionsOutput, _ *sns.ListSubscriptionsInput) ([]*sns.GetSubscriptionAttributesInput, error) {
			var inputs []*sns.GetSubscriptionAttributesInput
			for _, subs := range output.Subscriptions {
				inputs = append(inputs, &sns.GetSubscriptionAttributesInput{
					SubscriptionArn: subs.SubscriptionArn,
				})
			}
			return inputs, nil
		},
		GetFunc: getSubsFunc,
	}
}

var snsSubscriptionAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "sns-subscription",
	DescriptiveName: "SNS Subscription",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		List:              true,
		Search:            true,
		GetDescription:    "Get an SNS subscription by its ARN",
		SearchDescription: "Search SNS subscription by ARN",
		ListDescription:   "List all SNS subscriptions",
	},
	TerraformMappings: []*sdp.TerraformMapping{
		{TerraformQueryMap: "aws_sns_topic_subscription.id"},
	},
	PotentialLinks: []string{"sns-topic", "iam-role"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
})
