package apigateway

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
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

func findMethod(ctx context.Context, client *apigateway.Client, restAPIID, resourceID *string, method string) error {
	_, err := client.GetMethod(ctx, &apigateway.GetMethodInput{
		RestApiId:  restAPIID,
		ResourceId: resourceID,
		HttpMethod: &method,
	})

	if err != nil {
		var notFoundErr *types.NotFoundException
		if errors.As(err, &notFoundErr) {
			return integration.NewNotFoundError(integration.ResourceName(
				integration.APIGateway,
				methodSrc,
				method,
			))
		}

		return err
	}

	return nil
}

func findMethodResponse(ctx context.Context, client *apigateway.Client, restAPIID, resourceID *string, method string, statusCode string) error {
	_, err := client.GetMethodResponse(ctx, &apigateway.GetMethodResponseInput{
		RestApiId:  restAPIID,
		ResourceId: resourceID,
		HttpMethod: &method,
		StatusCode: &statusCode,
	})

	if err != nil {
		var notFoundErr *types.NotFoundException
		if errors.As(err, &notFoundErr) {
			return integration.NewNotFoundError(integration.ResourceName(
				integration.APIGateway,
				methodResponseSrc,
				method,
				statusCode,
			))
		}

		return err
	}

	return nil
}

func findIntegration(ctx context.Context, client *apigateway.Client, restAPIID, resourceID *string, method string) error {
	_, err := client.GetIntegration(ctx, &apigateway.GetIntegrationInput{
		RestApiId:  restAPIID,
		ResourceId: resourceID,
		HttpMethod: &method,
	})

	if err != nil {
		var notFoundErr *types.NotFoundException
		if errors.As(err, &notFoundErr) {
			return integration.NewNotFoundError(integration.ResourceName(
				integration.APIGateway,
				integrationSrc,
				method,
			))
		}

		return err
	}

	return nil
}

func findAPIKeyByName(ctx context.Context, client *apigateway.Client, name string) (*string, error) {
	result, err := client.GetApiKeys(ctx, &apigateway.GetApiKeysInput{
		NameQuery: &name,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, integration.NewNotFoundError(integration.ResourceName(integration.APIGateway, apiKeySrc, name))
	}

	for _, apiKey := range result.Items {
		if *apiKey.Name == name {
			return apiKey.Id, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.APIGateway, apiKeySrc, name))
}
