package kms

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func resourceTags(resourceName, testID string, nameAdditionalAttr ...string) []types.Tag {
	return []types.Tag{
		{
			TagKey:   aws.String(integration.TagTestKey),
			TagValue: aws.String(integration.TagTestValue),
		},
		{
			TagKey:   aws.String(integration.TagTestTypeKey),
			TagValue: aws.String(integration.TestName(integration.KMS)),
		},
		{
			TagKey:   aws.String(integration.TagTestIDKey),
			TagValue: aws.String(testID),
		},
		{
			TagKey:   aws.String(integration.TagResourceIDKey),
			TagValue: aws.String(integration.ResourceName(integration.KMS, resourceName, nameAdditionalAttr...)),
		},
	}
}

func hasTags(tags []types.Tag, requiredTags []types.Tag) bool {
	rT := make(map[string]string)
	for _, t := range requiredTags {
		rT[*t.TagKey] = *t.TagValue
	}

	oT := make(map[string]string)
	for _, t := range tags {
		oT[*t.TagKey] = *t.TagValue
	}

	for k, v := range rT {
		if oT[k] != v {
			return false
		}
	}

	return true
}
