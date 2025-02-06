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
			return nil
		} else {
			return err
		}
	}

	return deleteRestAPI(ctx, client, *restAPIID)
}
