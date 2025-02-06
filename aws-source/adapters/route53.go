package adapters

import "github.com/aws/aws-sdk-go-v2/service/route53/types"

func route53TagsToMap(tags []types.Tag) map[string]string {
	m := make(map[string]string)

	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			m[*tag.Key] = *tag.Value
		}
	}

	return m
}
