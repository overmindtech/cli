package kms

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kms"
)

func deleteKey(ctx context.Context, client *kms.Client, keyID string) error {
	seven := int32(7)
	_, err := client.ScheduleKeyDeletion(ctx, &kms.ScheduleKeyDeletionInput{
		KeyId:               &keyID,
		PendingWindowInDays: &seven, // it can be minimum 7 days
	})
	return err
}

func deleteAlias(ctx context.Context, client *kms.Client, aliasName string) error {
	_, err := client.DeleteAlias(ctx, &kms.DeleteAliasInput{
		AliasName: &aliasName,
	})
	return err
}

func deleteGrant(ctx context.Context, client *kms.Client, keyID, grantID string) error {
	_, err := client.RevokeGrant(ctx, &kms.RevokeGrantInput{
		KeyId:   &keyID,
		GrantId: &grantID,
	})
	return err
}
