package kms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func createKey(ctx context.Context, logger *slog.Logger, client *kms.Client, testID string) (*string, error) {
	// check if a resource with the same tags already exists
	id, err := findActiveKeyIDByTags(ctx, client)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating KMS key")
		} else {
			return nil, err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "KMS key already exists")
		return id, nil
	}

	response, err := client.CreateKey(ctx, &kms.CreateKeyInput{
		Tags: resourceTags(keySrc, testID),
	})
	if err != nil {
		return nil, err
	}

	return response.KeyMetadata.KeyId, nil
}

func createAlias(ctx context.Context, logger *slog.Logger, client *kms.Client, keyID string) error {
	aliasName := genAliasName()
	aliasNames, err := findAliasesByTargetKey(ctx, client, keyID)
	if err != nil {
		if nf := integration.NewNotFoundError(aliasSrc); errors.As(err, &nf) {
			logger.WarnContext(ctx, "Creating alias for the key", "keyID", keyID)
		} else {
			return err
		}
	}

	for _, aName := range aliasNames {
		if aName == aliasName {
			logger.InfoContext(ctx, "KMS alias already exists", "alias", aliasName, "keyID", keyID)
			return nil
		}
	}

	_, err = client.CreateAlias(ctx, &kms.CreateAliasInput{
		AliasName:   &aliasName,
		TargetKeyId: &keyID,
	})
	if err != nil {
		return err
	}

	return nil
}

func genAliasName() string {
	return fmt.Sprintf("alias/%s", integration.TestID())
}

func createGrant(ctx context.Context, logger *slog.Logger, client *kms.Client, keyID, principal string) error {
	grantID, err := findGrant(ctx, client, keyID, principal)
	if err != nil {
		if nf := integration.NewNotFoundError(grantSrc); errors.As(err, &nf) {
			logger.WarnContext(ctx, "Creating grant for the key", "keyID", keyID, "principal", principal)
		} else {
			return err
		}
	}

	if grantID != nil {
		logger.InfoContext(ctx, "KMS grant already exists", "grantID", *grantID, "keyID", keyID, "principal", principal)

		return nil
	}

	_, err = client.CreateGrant(ctx, &kms.CreateGrantInput{
		GranteePrincipal: &principal,
		KeyId:            &keyID,
		Operations:       []types.GrantOperation{types.GrantOperationDecrypt},
	})
	if err != nil {
		return err
	}

	return nil
}

func putKeyPolicy(ctx context.Context, logger *slog.Logger, client *kms.Client, keyID, principal string) error {
	keyPolicy, err := findKeyPolicy(ctx, client, keyID)
	if err != nil {
		if nf := integration.NewNotFoundError(keyPolicySrc); errors.As(err, &nf) {
			logger.WarnContext(ctx, "Creating key policy for the key", "keyID", keyID)
		} else {
			return err
		}
	}

	if keyPolicy != nil {
		logger.InfoContext(ctx, "KMS key policy already exists", "keyID", keyID)
		return nil
	}

	policy := fmt.Sprintf(
		`{
		  "Sid": "Allow access for Key Administrators",
		  "Effect": "Allow",
		  "Principal": {"AWS":"%s"},
		  "Action": [
			"kms:Create*",
			"kms:Describe*",
			"kms:Enable*",
			"kms:List*",
			"kms:Put*",
			"kms:Update*",
			"kms:Revoke*",
			"kms:Disable*",
			"kms:Get*",
			"kms:Delete*",
			"kms:TagResource",
			"kms:UntagResource",
			"kms:ScheduleKeyDeletion",
			"kms:CancelKeyDeletion",
			"kms:RotateKeyOnDemand"
		  ],
		  "Resource": "*"
		}`, principal)

	_, err = client.PutKeyPolicy(ctx, &kms.PutKeyPolicyInput{
		KeyId:  &keyID,
		Policy: &policy,
	})
	if err != nil {
		return err
	}

	return nil
}
