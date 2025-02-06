package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

type mockEndpointClient struct{}

func (m *mockEndpointClient) ListTagsForResource(ctx context.Context, input *sns.ListTagsForResourceInput, f ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error) {
	// intentionally returns nil to test the nil case
	return nil, nil
}

func (m *mockEndpointClient) GetEndpointAttributes(ctx context.Context, params *sns.GetEndpointAttributesInput, optFns ...func(*sns.Options)) (*sns.GetEndpointAttributesOutput, error) {
	return &sns.GetEndpointAttributesOutput{
		Attributes: map[string]string{
			"Enabled": "true",
			"Token":   "EXAMPLE12345...",
		},
	}, nil
}

func (m *mockEndpointClient) ListEndpointsByPlatformApplication(ctx context.Context, params *sns.ListEndpointsByPlatformApplicationInput, optFns ...func(*sns.Options)) (*sns.ListEndpointsByPlatformApplicationOutput, error) {
	return &sns.ListEndpointsByPlatformApplicationOutput{
		Endpoints: []types.Endpoint{
			{
				Attributes: map[string]string{
					"Token":   "EXAMPLE12345...",
					"Enabled": "true",
				},
			},
		},
	}, nil
}

func TestGetEndpointFunc(t *testing.T) {
	ctx := context.Background()
	cli := &mockEndpointClient{}

	item, err := getEndpointFunc(ctx, cli, "scope", &sns.GetEndpointAttributesInput{
		EndpointArn: adapterhelpers.PtrString("arn:aws:sns:us-west-2:123456789012:endpoint/GCM/MyApplication/12345678-abcd-9012-efgh-345678901234"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestNewSNSEndpointAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := sns.NewFromConfig(config)

	adapter := NewSNSEndpointAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
