package kms

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/smithy-go"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

// findActiveKeyIDByTags finds a key by tags
// additionalAttr is a variadic parameter that allows to specify additional attributes to search for
func findActiveKeyIDByTags(ctx context.Context, client *kms.Client, additionalAttr ...string) (*string, error) {
	result, err := client.ListKeys(ctx, &kms.ListKeysInput{})
	if err != nil {
		return nil, err
	}

	for _, keyListEntry := range result.Keys {
		key, err := client.DescribeKey(ctx, &kms.DescribeKeyInput{
			KeyId: keyListEntry.KeyId,
		})
		if err != nil {
			return nil, err
		}

		if key.KeyMetadata.KeyState != types.KeyStateEnabled {
			continue
		}

		tags, err := client.ListResourceTags(ctx, &kms.ListResourceTagsInput{
			KeyId: keyListEntry.KeyId,
		})
		// There are some keys that even admins can't list the tags of. Not sure
		// why but they seem to exist, we will ignore permissions errors here.
		var awsErr *smithy.GenericAPIError
		if errors.As(err, &awsErr) && awsErr.ErrorCode() == "AccessDeniedException" {
			continue
		}
		if err != nil {
			return nil, err
		}

		if hasTags(tags.Tags, resourceTags(keySrc, integration.TestID(), additionalAttr...)) {
			return keyListEntry.KeyId, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.KMS, keySrc, additionalAttr...))
}

func findAliasesByTargetKey(ctx context.Context, client *kms.Client, keyID string) ([]string, error) {
	aliases, err := client.ListAliases(ctx, &kms.ListAliasesInput{
		KeyId: &keyID,
	})
	if err != nil {
		return nil, err
	}

	var aliasNames []string
	for _, alias := range aliases.Aliases {
		aliasNames = append(aliasNames, *alias.AliasName)
	}

	if len(aliasNames) == 0 {
		return nil, integration.NewNotFoundError(integration.ResourceName(integration.KMS, aliasSrc))
	}

	return aliasNames, nil
}

func findGrant(ctx context.Context, client *kms.Client, keyID, principal string) (*string, error) {
	// Get grants for the key
	grants, err := client.ListGrants(ctx, &kms.ListGrantsInput{
		KeyId: &keyID,
	})
	if err != nil {
		return nil, err
	}

	for _, grant := range grants.Grants {
		if *grant.GranteePrincipal == principal {
			return grant.GrantId, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.KMS, grantSrc))
}

func findKeyPolicy(ctx context.Context, client *kms.Client, keyID string) (*string, error) {
	policy, err := client.GetKeyPolicy(ctx, &kms.GetKeyPolicyInput{
		KeyId: &keyID,
	})
	if err != nil {
		return nil, err
	}

	if policy.Policy == nil {
		return nil, integration.NewNotFoundError(integration.ResourceName(integration.KMS, keyPolicySrc))
	}

	return policy.Policy, nil
}
