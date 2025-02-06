package kms

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

const (
	keySrc       = "key"
	aliasSrc     = "alias"
	grantSrc     = "grant"
	keyPolicySrc = "key-policy"
)

func setup(ctx context.Context, logger *slog.Logger, client *kms.Client) error {
	testID := integration.TestID()

	// Create KMS key
	keyID, err := createKey(ctx, logger, client, testID)
	if err != nil {
		return err
	}

	// Create KMS alias
	err = createAlias(ctx, logger, client, *keyID)
	if err != nil {
		return err
	}

	principal, err := integration.GetCallerIdentityARN(ctx)
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	// Create KMS grant
	err = createGrant(ctx, logger, client, *keyID, *principal)
	if err != nil {
		return err
	}

	// Create KMS key policy
	return putKeyPolicy(ctx, logger, client, *keyID, *principal)
}
