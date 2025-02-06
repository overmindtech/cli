package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func tags(ctx context.Context, cli sqsClient, queURL string) (map[string]string, error) {
	if cli == nil {
		return nil, nil
	}

	output, err := cli.ListQueueTags(ctx, &sqs.ListQueueTagsInput{
		QueueUrl: &queURL,
	})
	if err != nil {
		return nil, err
	}

	return output.Tags, nil
}
