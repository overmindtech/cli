package ec2

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func resourceTags(resourceName, testID string, nameAdditionalAttr ...string) []types.Tag {
	return []types.Tag{
		{
			Key:   aws.String(integration.TagTestKey),
			Value: aws.String(integration.TagTestValue),
		},
		{
			Key:   aws.String(integration.TagTestTypeKey),
			Value: aws.String(integration.TestName(integration.EC2)),
		},
		{
			Key:   aws.String(integration.TagTestIDKey),
			Value: aws.String(testID),
		},
		{
			Key:   aws.String(integration.TagResourceIDKey),
			Value: aws.String(integration.ResourceName(integration.EC2, resourceName, nameAdditionalAttr...)),
		},
	}
}

func hasTags(tags []types.Tag, requiredTags []types.Tag) bool {
	rT := make(map[string]string)
	for _, t := range requiredTags {
		rT[*t.Key] = *t.Value
	}

	oT := make(map[string]string)
	for _, t := range tags {
		oT[*t.Key] = *t.Value
	}

	for k, v := range rT {
		if oT[k] != v {
			return false
		}
	}

	return true
}
