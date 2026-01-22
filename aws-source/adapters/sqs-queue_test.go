package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/overmindtech/cli/sdp-go"
)

type testClient struct{}

func (t testClient) GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	return &sqs.GetQueueAttributesOutput{
		Attributes: map[string]string{
			"ApproximateNumberOfMessages":           "0",
			"ApproximateNumberOfMessagesDelayed":    "0",
			"ApproximateNumberOfMessagesNotVisible": "0",
			"CreatedTimestamp":                      "1631616000",
			"DelaySeconds":                          "0",
			"LastModifiedTimestamp":                 "1631616000",
			"MaximumMessageSize":                    "262144",
			"MessageRetentionPeriod":                "345600",
			"QueueArn":                              "arn:aws:sqs:us-west-2:123456789012:MyQueue",
			"ReceiveMessageWaitTimeSeconds":         "0",
			"VisibilityTimeout":                     "30",
			"RedrivePolicy":                         "{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:80398EXAMPLE:MyDeadLetterQueue\",\"maxReceiveCount\":1000}",
		},
	}, nil
}

func (t testClient) ListQueueTags(ctx context.Context, params *sqs.ListQueueTagsInput, optFns ...func(*sqs.Options)) (*sqs.ListQueueTagsOutput, error) {
	return &sqs.ListQueueTagsOutput{
		Tags: map[string]string{
			"tag1": "value1",
			"tag2": "value2",
		},
	}, nil
}

func (t testClient) ListQueues(ctx context.Context, input *sqs.ListQueuesInput, f ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
	return &sqs.ListQueuesOutput{
		QueueUrls: []string{
			"https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue",
			"https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue2",
		},
	}, nil
}

func TestGetFunc(t *testing.T) {
	ctx := context.Background()
	cli := testClient{}

	item, err := getFunc(ctx, cli, "scope", &sqs.GetQueueAttributesInput{
		QueueUrl: PtrString("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	// Test linked item queries
	if len(item.GetLinkedItemQueries()) != 2 {
		t.Errorf("Expected 2 linked item queries, got %d", len(item.GetLinkedItemQueries()))
	}

	// Test HTTP link
	httpLink := item.GetLinkedItemQueries()[0]
	if httpLink.GetQuery().GetType() != "http" {
		t.Errorf("Expected first link type to be 'http', got %s", httpLink.GetQuery().GetType())
	}
	if httpLink.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
		t.Errorf("Expected HTTP link method to be SEARCH, got %v", httpLink.GetQuery().GetMethod())
	}
	// Test HTTP link blast propagation (bidirectional)
	if httpLink.GetBlastPropagation().GetIn() != true {
		t.Errorf("Expected HTTP link blast propagation In to be true, got %v", httpLink.GetBlastPropagation().GetIn())
	}
	if httpLink.GetBlastPropagation().GetOut() != true {
		t.Errorf("Expected HTTP link blast propagation Out to be true, got %v", httpLink.GetBlastPropagation().GetOut())
	}

	// Test Lambda Event Source Mapping link
	lambdaLink := item.GetLinkedItemQueries()[1]
	if lambdaLink.GetQuery().GetType() != "lambda-event-source-mapping" {
		t.Errorf("Expected second link type to be 'lambda-event-source-mapping', got %s", lambdaLink.GetQuery().GetType())
	}
	if lambdaLink.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
		t.Errorf("Expected Lambda link method to be SEARCH, got %v", lambdaLink.GetQuery().GetMethod())
	}
	if lambdaLink.GetQuery().GetQuery() != "arn:aws:sqs:us-west-2:123456789012:MyQueue" {
		t.Errorf("Expected Lambda link query to be the Queue ARN, got %s", lambdaLink.GetQuery().GetQuery())
	}
	// Test Lambda Event Source Mapping link blast propagation (outgoing only)
	if lambdaLink.GetBlastPropagation().GetIn() != false {
		t.Errorf("Expected Lambda link blast propagation In to be false, got %v", lambdaLink.GetBlastPropagation().GetIn())
	}
	if lambdaLink.GetBlastPropagation().GetOut() != true {
		t.Errorf("Expected Lambda link blast propagation Out to be true, got %v", lambdaLink.GetBlastPropagation().GetOut())
	}
}

func TestSqsQueueSearchInputMapper(t *testing.T) {
	tests := []struct {
		name        string
		arn         string
		expectedURL string
	}{
		{
			name:        "aws partition",
			arn:         "arn:aws:sqs:eu-west-2:540044833068:-tfc-notifications-from-s3",
			expectedURL: "https://sqs.eu-west-2.amazonaws.com/540044833068/-tfc-notifications-from-s3",
		},
		{
			name:        "aws-cn partition",
			arn:         "arn:aws-cn:sqs:cn-north-1:540044833068:my-queue",
			expectedURL: "https://sqs.cn-north-1.amazonaws.com.cn/540044833068/my-queue",
		},
		{
			name:        "aws-us-gov partition",
			arn:         "arn:aws-us-gov:sqs:us-gov-west-1:540044833068:gov-queue",
			expectedURL: "https://sqs.us-gov-west-1.amazonaws.com/540044833068/gov-queue",
		},
		{
			name:        "aws-iso partition",
			arn:         "arn:aws-iso:sqs:us-iso-east-1:540044833068:iso-queue",
			expectedURL: "https://sqs.us-iso-east-1.c2s.ic.gov/540044833068/iso-queue",
		},
		{
			name:        "aws-iso-b partition",
			arn:         "arn:aws-iso-b:sqs:us-isob-east-1:540044833068:isob-queue",
			expectedURL: "https://sqs.us-isob-east-1.sc2s.sgov.gov/540044833068/isob-queue",
		},
		{
			name:        "aws-eu partition",
			arn:         "arn:aws-eu:sqs:eu-central-1:540044833068:eu-queue",
			expectedURL: "https://sqs.eu-central-1.amazonaws.eu/540044833068/eu-queue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs, err := sqsQueueSearchInputMapper("scope", tt.arn)
			if err != nil {
				t.Fatalf("sqsQueueSearchInputMapper() error = %v", err)
			}

			if inputs.QueueUrl == nil {
				t.Fatal("QueueUrl is nil")
			}

			if *inputs.QueueUrl != tt.expectedURL {
				t.Errorf("Expected QueueUrl to be %s, got %s", tt.expectedURL, *inputs.QueueUrl)
			}
		})
	}
}

func TestNewQueueAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := sqs.NewFromConfig(config)

	adapter := NewSQSQueueAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
