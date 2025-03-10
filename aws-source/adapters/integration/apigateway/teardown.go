package apigateway

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"

	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func teardown(ctx context.Context, logger *slog.Logger, client *apigateway.Client) error {
	restAPIID, err := findRestAPIsByTags(ctx, client)
	if err != nil {
		if nf := integration.NewNotFoundError(restAPISrc); errors.As(err, &nf) {
			logger.WarnContext(ctx, "Rest API not found")
		} else {
			return err
		}
	} else {
		err = deleteRestAPI(ctx, client, *restAPIID)
		if err != nil {
			return err
		}
	}

	keyName := integration.ResourceName(integration.APIGateway, apiKeySrc, integration.TestID())
	apiKeyID, err := findAPIKeyByName(ctx, client, keyName)
	if err != nil {
		if nf := integration.NewNotFoundError(apiKeySrc); errors.As(err, &nf) {
			logger.WarnContext(ctx, "API Key not found", "name", keyName)
			return nil
		} else {
			return err
		}
	} else {
		err = deleteAPIKeyByName(ctx, client, apiKeyID)
		if err != nil {
			return err
		}
	}

	return nil
}
