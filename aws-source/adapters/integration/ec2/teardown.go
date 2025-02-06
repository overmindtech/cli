package ec2

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func teardown(ctx context.Context, logger *slog.Logger, client *ec2.Client) error {
	instanceID, err := findActiveInstanceIDByTags(ctx, client)
	if err != nil {
		nf := integration.NewNotFoundError(instanceSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Instance not found")
			return nil
		} else {
			return err
		}
	}

	return deleteInstance(ctx, client, *instanceID)
}
