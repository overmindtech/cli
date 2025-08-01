package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/discovery"
)

type mockSSMClient struct{}

func (m *mockSSMClient) DescribeParameters(ctx context.Context, input *ssm.DescribeParametersInput, opts ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
	return &ssm.DescribeParametersOutput{
		Parameters: []types.ParameterMetadata{
			{
				ARN:              aws.String("arn:aws:ssm:us-west-2:123456789012:parameter/test"),
				AllowedPattern:   aws.String(".*"),
				DataType:         aws.String("text"),
				Description:      aws.String("test"),
				KeyId:            aws.String("test"),
				LastModifiedDate: aws.Time(time.Now()),
				LastModifiedUser: aws.String("test"),
				Name:             aws.String("test"),
				Policies: []types.ParameterInlinePolicy{
					{
						PolicyStatus: aws.String("Pending"),
						PolicyText:   aws.String("test"),
						PolicyType:   aws.String("ExpirationNotification"),
					},
				},
				Tier:    types.ParameterTierStandard,
				Type:    types.ParameterTypeString,
				Version: 1,
			},
		},
	}, nil
}

func (m *mockSSMClient) ListTagsForResource(ctx context.Context, input *ssm.ListTagsForResourceInput, opts ...func(*ssm.Options)) (*ssm.ListTagsForResourceOutput, error) {
	return &ssm.ListTagsForResourceOutput{
		TagList: []types.Tag{
			{
				Key:   aws.String("foo"),
				Value: aws.String("bar"),
			},
		},
	}, nil
}

func (m *mockSSMClient) GetParameter(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			ARN:              aws.String("arn:aws:ssm:us-west-2:123456789012:parameter/test"),
			DataType:         aws.String("text"),
			LastModifiedDate: aws.Time(time.Now()),
			Name:             aws.String("test"),
			Selector:         aws.String("test"),
			SourceResult:     aws.String("test"),
			Type:             types.ParameterTypeString,
			Value:            aws.String("https://www.google.com"),
			Version:          1,
		},
	}, nil
}

func TestSSMParameterAdapter(t *testing.T) {
	adapter := NewSSMParameterAdapter(&mockSSMClient{}, "123456789", "us-east-1")

	t.Run("Get", func(t *testing.T) {
		item, err := adapter.Get(context.Background(), "123456789.us-east-1", "test", false)
		if err != nil {
			t.Fatal(err)
		}

		err = item.Validate()
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("List", func(t *testing.T) {
		stream := discovery.NewRecordingQueryResultStream()
		adapter.ListStream(context.Background(), "123456789.us-east-1", false, stream)

		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Error(errs)
		}

		items := stream.GetItems()
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}

		err := items[0].Validate()
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Search", func(t *testing.T) {
		stream := discovery.NewRecordingQueryResultStream()
		adapter.SearchStream(context.Background(), "123456789.us-east-1", "arn:aws:ssm:us-east-1:1234567890:parameter/prod/*/service/example-service", false, stream)

		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Error(errs)
		}

		items := stream.GetItems()
		if len(items) != 0 {
			t.Errorf("expected 0 item, got %d", len(items))
		}
	})
}

func TestSSMParameterAdapterE2E(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := ssm.NewFromConfig(config)

	adapter := NewSSMParameterAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
