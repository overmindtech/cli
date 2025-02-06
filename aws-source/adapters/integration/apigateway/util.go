package apigateway

import (
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func resourceTags(resourceName, testID string, nameAdditionalAttr ...string) map[string]string {
	return map[string]string{
		integration.TagTestKey:       integration.TagTestValue,
		integration.TagTestTypeKey:   integration.TestName(integration.APIGateway),
		integration.TagTestIDKey:     testID,
		integration.TagResourceIDKey: integration.ResourceName(integration.APIGateway, resourceName, nameAdditionalAttr...),
	}
}

func hasTags(tags map[string]string, requiredTags map[string]string) bool {
	for k, v := range requiredTags {
		if tags[k] != v {
			return false
		}
	}

	return true
}
