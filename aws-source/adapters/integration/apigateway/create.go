package apigateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func createRestAPI(ctx context.Context, logger *slog.Logger, client *apigateway.Client, testID string) (*string, error) {
	// check if a resource with the same tags already exists
	id, err := findRestAPIsByTags(ctx, client)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating Rest API")
		} else {
			return nil, err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "Rest API already exists")
		return id, nil
	}

	result, err := client.CreateRestApi(ctx, &apigateway.CreateRestApiInput{
		Name:        adapterhelpers.PtrString(integration.ResourceName(integration.APIGateway, restAPISrc, testID)),
		Description: adapterhelpers.PtrString("Test Rest API"),
		Tags:        resourceTags(restAPISrc, testID),
	})
	if err != nil {
		return nil, err
	}

	return result.Id, nil
}

func createResource(ctx context.Context, logger *slog.Logger, client *apigateway.Client, restAPIID, parentID *string, path string) (*string, error) {
	// check if a resource with the same path already exists
	resourceID, err := findResource(ctx, client, restAPIID, path)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating resource")
		} else {
			return nil, err
		}
	}

	if resourceID != nil {
		logger.InfoContext(ctx, "Resource already exists")
		return resourceID, nil
	}

	result, err := client.CreateResource(ctx, &apigateway.CreateResourceInput{
		RestApiId: restAPIID,
		ParentId:  parentID,
		PathPart:  adapterhelpers.PtrString(cleanPath(path)),
	})
	if err != nil {
		return nil, err
	}

	return result.Id, nil
}

func cleanPath(path string) string {
	p, ok := strings.CutPrefix(path, "/")
	if !ok {
		return path
	}

	return p
}

func createMethod(ctx context.Context, logger *slog.Logger, client *apigateway.Client, restAPIID, resourceID *string, method string) error {
	// check if a method with the same name already exists
	err := findMethod(ctx, client, restAPIID, resourceID, method)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating method")
		} else {
			return err
		}
	}

	if err == nil {
		logger.InfoContext(ctx, "Method already exists")
		return nil
	}

	_, err = client.PutMethod(ctx, &apigateway.PutMethodInput{
		RestApiId:         restAPIID,
		ResourceId:        resourceID,
		HttpMethod:        adapterhelpers.PtrString(method),
		AuthorizationType: adapterhelpers.PtrString("NONE"),
	})
	if err != nil {
		return err
	}

	return nil
}

func createMethodResponse(ctx context.Context, logger *slog.Logger, client *apigateway.Client, restAPIID, resourceID *string, method, statusCode string) error {
	// check if a method response with the same status code already exists
	err := findMethodResponse(ctx, client, restAPIID, resourceID, method, statusCode)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating method response")
		} else {
			return err
		}
	}

	if err == nil {
		logger.InfoContext(ctx, "Method response already exists")
		return nil
	}

	_, err = client.PutMethodResponse(ctx, &apigateway.PutMethodResponseInput{
		RestApiId:  restAPIID,
		ResourceId: resourceID,
		HttpMethod: adapterhelpers.PtrString(method),
		StatusCode: adapterhelpers.PtrString(statusCode),
		ResponseModels: map[string]string{
			"application/json": "Empty",
		},
		ResponseParameters: map[string]bool{
			"method.response.header.Content-Type": true,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func createIntegration(ctx context.Context, logger *slog.Logger, client *apigateway.Client, restAPIID, resourceID *string, method string) error {
	// check if an integration with the same method already exists
	err := findIntegration(ctx, client, restAPIID, resourceID, method)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating integration")
		} else {
			return err
		}
	}

	if err == nil {
		logger.InfoContext(ctx, "Integration already exists")
		return nil
	}

	_, err = client.PutIntegration(ctx, &apigateway.PutIntegrationInput{
		RestApiId:  restAPIID,
		ResourceId: resourceID,
		HttpMethod: adapterhelpers.PtrString(method),
		Type:       "MOCK",
	})
	if err != nil {
		return err
	}

	return nil
}

func createAPIKey(ctx context.Context, logger *slog.Logger, client *apigateway.Client, testID string) error {
	// check if an API key with the same name already exists
	id, err := findAPIKeyByName(ctx, client, integration.ResourceName(integration.APIGateway, apiKeySrc, testID))
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating API key")
		} else {
			return err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "API key already exists")
		return nil
	}

	_, err = client.CreateApiKey(ctx, &apigateway.CreateApiKeyInput{
		Name:    adapterhelpers.PtrString(integration.ResourceName(integration.APIGateway, apiKeySrc, testID)),
		Tags:    resourceTags(apiKeySrc, testID),
		Enabled: true,
	})
	if err != nil {
		return err
	}

	return nil
}
