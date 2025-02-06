package kms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func teardown(ctx context.Context, logger *slog.Logger, client *kms.Client) error {
	keyID, err := findActiveKeyIDByTags(ctx, client)
	if err != nil {
		if nf := integration.NewNotFoundError(keySrc); errors.As(err, &nf) {
			logger.WarnContext(ctx, "Key not found")
			return nil
		} else {
			return err
		}
	}

	principal, err := integration.GetCallerIdentityARN(ctx)
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	grantID, err := findGrant(ctx, client, *keyID, *principal)
	if err != nil {
		if nf := integration.NewNotFoundError(grantSrc); errors.As(err, &nf) {
			logger.WarnContext(ctx, "Grant not found")
		} else {
			return err
		}
	}

	err = deleteGrant(ctx, client, *keyID, *grantID)
	if err != nil {
		return err
	}

	aliasNames, err := findAliasesByTargetKey(ctx, client, *keyID)
	if err != nil {
		if nf := integration.NewNotFoundError(aliasSrc); errors.As(err, &nf) {
			logger.WarnContext(ctx, "Alias not found")
		} else {
			return err
		}
	}

	for _, aliasName := range aliasNames {
		err = deleteAlias(ctx, client, aliasName)
		if err != nil {
			return err
		}
	}

	return deleteKey(ctx, client, *keyID)
}
