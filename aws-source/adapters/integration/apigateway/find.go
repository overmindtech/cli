package apigateway

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func findRestAPIsByTags(ctx context.Context, client *apigateway.Client, additionalAttr ...string) (*string, error) {
	result, err := client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err != nil {
		return nil, err
	}

	for _, api := range result.Items {
		if hasTags(api.Tags, resourceTags(restAPISrc, integration.TestID(), additionalAttr...)) {
			return api.Id, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.APIGateway, restAPISrc, additionalAttr...))
}

func findResource(ctx context.Context, client *apigateway.Client, restAPIID *string, path string) (*string, error) {
	result, err := client.GetResources(ctx, &apigateway.GetResourcesInput{
		RestApiId: restAPIID,
	})
	if err != nil {
		return nil, err
	}

	for _, resource := range result.Items {
		if *resource.Path == path {
			return resource.Id, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.APIGateway, resourceSrc, path))
}
