package apigateway

import (
	"context"
	"errors"
	"log/slog"

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
		PathPart:  adapterhelpers.PtrString(path),
	})
	if err != nil {
		return nil, err
	}

	return result.Id, nil
}
