package apigateway

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

const (
	restAPISrc  = "rest-api"
	resourceSrc = "resource"
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
	_, err = createResource(ctx, logger, client, restApiID, rootResourceID, "test")
	if err != nil {
		return err
	}

	return nil
}
