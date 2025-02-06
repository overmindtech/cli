package adapters

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/directconnect"
	"github.com/aws/aws-sdk-go-v2/service/directconnect/types"
)

// Converts a slice of tags to a map
func directconnectTagsToMap(tags []types.Tag) map[string]string {
	tagsMap := make(map[string]string)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			tagsMap[*tag.Key] = *tag.Value
		}
	}

	return tagsMap
}

func arnToTags(ctx context.Context, cli *directconnect.Client, resourceARNs []string) (map[string][]types.Tag, error) {
	if cli == nil {
		return nil, nil
	}

	tagsOutput, err := cli.DescribeTags(ctx, &directconnect.DescribeTagsInput{
		ResourceArns: resourceARNs,
	})
	if err != nil {
		return nil, err
	}

	tags := make(map[string][]types.Tag, len(tagsOutput.ResourceTags))
	for _, tag := range tagsOutput.ResourceTags {
		tags[*tag.ResourceArn] = tag.Tags
	}

	return tags, nil
}
