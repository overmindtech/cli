package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

type mockPlatformApplicationClient struct{}

func (m mockPlatformApplicationClient) ListTagsForResource(ctx context.Context, input *sns.ListTagsForResourceInput, f ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error) {
	return &sns.ListTagsForResourceOutput{
		Tags: []types.Tag{
			{Key: new("tag1"), Value: new("value1")},
			{Key: new("tag2"), Value: new("value2")},
		},
	}, nil
}

func (m mockPlatformApplicationClient) GetPlatformApplicationAttributes(ctx context.Context, params *sns.GetPlatformApplicationAttributesInput, optFns ...func(*sns.Options)) (*sns.GetPlatformApplicationAttributesOutput, error) {
	return &sns.GetPlatformApplicationAttributesOutput{
		Attributes: map[string]string{
			"Enabled":                   "true",
			"SuccessFeedbackSampleRate": "100",
		},
	}, nil
}

func (m mockPlatformApplicationClient) ListPlatformApplications(ctx context.Context, params *sns.ListPlatformApplicationsInput, optFns ...func(*sns.Options)) (*sns.ListPlatformApplicationsOutput, error) {
	return &sns.ListPlatformApplicationsOutput{
		PlatformApplications: []types.PlatformApplication{
			{
				PlatformApplicationArn: new("arn:aws:sns:us-west-2:123456789012:app/ADM/MyApplication"),
				Attributes: map[string]string{
					"SuccessFeedbackSampleRate": "100",
					"Enabled":                   "true",
				},
			},
			{
				PlatformApplicationArn: new("arn:aws:sns:us-west-2:123456789012:app/MPNS/MyOtherApplication"),
				Attributes: map[string]string{
					"SuccessFeedbackSampleRate": "100",
					"Enabled":                   "true",
				},
			},
		},
	}, nil
}

func TestGetPlatformApplicationFunc(t *testing.T) {
	ctx := context.Background()
	cli := mockPlatformApplicationClient{}

	item, err := getPlatformApplicationFunc(ctx, cli, "scope", &sns.GetPlatformApplicationAttributesInput{
		PlatformApplicationArn: new("arn:aws:sns:us-west-2:123456789012:my-topic"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestNewSNSPlatformApplicationAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := sns.NewFromConfig(config)

	adapter := NewSNSPlatformApplicationAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
