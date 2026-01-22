package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type snsTestClient struct{}

func (t snsTestClient) GetSubscriptionAttributes(ctx context.Context, params *sns.GetSubscriptionAttributesInput, optFns ...func(*sns.Options)) (*sns.GetSubscriptionAttributesOutput, error) {
	return &sns.GetSubscriptionAttributesOutput{Attributes: map[string]string{
		"Endpoint":                     "my-email@example.com",
		"Protocol":                     "email",
		"RawMessageDelivery":           "false",
		"ConfirmationWasAuthenticated": "false",
		"Owner":                        "123456789012",
		"SubscriptionArn":              "arn:aws:sns:us-west-2:123456789012:my-topic:8a21d249-4329-4871-acc6-7be709c6ea7f",
		"TopicArn":                     "arn:aws:sns:us-west-2:123456789012:my-topic",
		"SubscriptionRoleArn":          "arn:aws:iam::123456789012:role/my-role",
	}}, nil
}

func (t snsTestClient) ListSubscriptions(context.Context, *sns.ListSubscriptionsInput, ...func(*sns.Options)) (*sns.ListSubscriptionsOutput, error) {
	return &sns.ListSubscriptionsOutput{
		Subscriptions: []types.Subscription{
			{
				Owner:           PtrString("123456789012"),
				Endpoint:        PtrString("my-email@example.com"),
				Protocol:        PtrString("email"),
				TopicArn:        PtrString("arn:aws:sns:us-west-2:123456789012:my-topic"),
				SubscriptionArn: PtrString("arn:aws:sns:us-west-2:123456789012:my-topic:8a21d249-4329-4871-acc6-7be709c6ea7f"),
			},
		},
	}, nil
}

func (t snsTestClient) ListTagsForResource(context.Context, *sns.ListTagsForResourceInput, ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error) {
	return &sns.ListTagsForResourceOutput{
		Tags: []types.Tag{
			{Key: PtrString("tag1"), Value: PtrString("value1")},
			{Key: PtrString("tag2"), Value: PtrString("value2")},
		},
	}, nil
}

func TestSNSGetFunc(t *testing.T) {
	ctx := context.Background()
	cli := snsTestClient{}

	item, err := getSubsFunc(ctx, cli, "scope", &sns.GetSubscriptionAttributesInput{
		SubscriptionArn: PtrString("arn:aws:sns:us-west-2:123456789012:my-topic:8a21d249-4329-4871-acc6-7be709c6ea7f"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestNewSNSSubscriptionAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := sns.NewFromConfig(config)

	adapter := NewSNSSubscriptionAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
