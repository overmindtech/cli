package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type tagLister interface {
	ListTagsForResource(ctx context.Context, params *sns.ListTagsForResourceInput, optFns ...func(*sns.Options)) (*sns.ListTagsForResourceOutput, error)
}

// tagsByResourceARN returns the tags for a given resource ARN
func tagsByResourceARN(ctx context.Context, cli tagLister, resourceARN string) ([]types.Tag, error) {
	if cli == nil {
		return nil, nil
	}

	output, err := cli.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{
		ResourceArn: &resourceARN,
	})
	if err != nil {
		return nil, err
	}

	if output != nil && output.Tags != nil {
		return output.Tags, nil
	}

	return nil, nil
}

// tagsToMap converts a slice of tags to a map
func tagsToMap(tags []types.Tag) map[string]string {
	tagsMap := make(map[string]string)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			tagsMap[*tag.Key] = *tag.Value
		}
	}

	return tagsMap
}
