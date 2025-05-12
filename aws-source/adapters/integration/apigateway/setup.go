package apigateway

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"

	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

const (
	restAPISrc        = "rest-api"
	resourceSrc       = "resource"
	methodSrc         = "method"
	methodResponseSrc = "method-response"
	integrationSrc    = "integration"
	apiKeySrc         = "api-key"
	authorizerSrc     = "authorizer"
	deploymentSrc     = "deployment"
	stageSrc          = "stage"
	modelSrc          = "model"
)

func setup(ctx context.Context, logger *slog.Logger, client *apigateway.Client) error {
	testID := integration.TestID()

	// Create Rest API
	restApiID, err := createRestAPI(ctx, logger, client, testID)
	if err != nil {
		return err
	}

	// Find root resource
	rootResourceID, err := findResource(ctx, client, restApiID, "/")
	if err != nil {
		return err
	}

	// Create resource
	testResourceID, err := createResource(ctx, logger, client, restApiID, rootResourceID, "/test")
	if err != nil {
		return err
	}

	// Create method
	err = createMethod(ctx, logger, client, restApiID, testResourceID, "GET")
	if err != nil {
		return err
	}

	// Create method response
	err = createMethodResponse(ctx, logger, client, restApiID, testResourceID, "GET", "200")
	if err != nil {
		return err
	}

	// Create integration
	err = createIntegration(ctx, logger, client, restApiID, testResourceID, "GET")
	if err != nil {
		return err
	}

	// Create API Key
	err = createAPIKey(ctx, logger, client, testID)
	if err != nil {
		return err
	}

	// Create Authorizer
	err = createAuthorizer(ctx, logger, client, *restApiID, testID)
	if err != nil {
		return err
	}

	// Create Deployment
	deploymentID, err := createDeployment(ctx, logger, client, *restApiID)
	if err != nil {
		return err
	}

	// Create Stage
	err = createStage(ctx, logger, client, *restApiID, *deploymentID)
	if err != nil {
		return err
	}

	// Create Model
	err = createModel(ctx, logger, client, *restApiID)
	if err != nil {
		return err
	}

	return nil
}
