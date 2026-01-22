package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type kmsTestClient struct{}

func (t kmsTestClient) DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error) {
	return &kms.DescribeKeyOutput{
		KeyMetadata: &types.KeyMetadata{
			AWSAccountId:          PtrString("846764612917"),
			KeyId:                 PtrString("b8a9477d-836c-491f-857e-07937918959b"),
			Arn:                   PtrString("arn:aws:kms:us-west-2:846764612917:key/b8a9477d-836c-491f-857e-07937918959b"),
			CreationDate:          PtrTime(time.Date(2017, 6, 30, 21, 44, 32, 140000000, time.UTC)),
			Enabled:               true,
			Description:           PtrString("Default KMS key that protects my S3 objects when no other key is defined"),
			KeyUsage:              types.KeyUsageTypeEncryptDecrypt,
			KeyState:              types.KeyStateEnabled,
			Origin:                types.OriginTypeAwsKms,
			KeyManager:            types.KeyManagerTypeAws,
			CustomerMasterKeySpec: types.CustomerMasterKeySpecSymmetricDefault,
			EncryptionAlgorithms: []types.EncryptionAlgorithmSpec{
				types.EncryptionAlgorithmSpecSymmetricDefault,
			},
		},
	}, nil
}

func (t kmsTestClient) ListKeys(context.Context, *kms.ListKeysInput, ...func(*kms.Options)) (*kms.ListKeysOutput, error) {
	return &kms.ListKeysOutput{
		Keys: []types.KeyListEntry{
			{
				KeyArn: PtrString("arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"),
				KeyId:  PtrString("1234abcd-12ab-34cd-56ef-1234567890ab"),
			},
			{
				KeyArn: PtrString("arn:aws:kms:us-west-2:111122223333:key/0987dcba-09fe-87dc-65ba-ab0987654321"),
				KeyId:  PtrString("0987dcba-09fe-87dc-65ba-ab0987654321"),
			},
			{
				KeyArn: PtrString("arn:aws:kms:us-east-2:111122223333:key/1a2b3c4d-5e6f-1a2b-3c4d-5e6f1a2b3c4d"),
				KeyId:  PtrString("1a2b3c4d-5e6f-1a2b-3c4d-5e6f1a2b3c4d"),
			},
		},
	}, nil
}

func (t kmsTestClient) ListResourceTags(context.Context, *kms.ListResourceTagsInput, ...func(*kms.Options)) (*kms.ListResourceTagsOutput, error) {
	return &kms.ListResourceTagsOutput{
		Tags: []types.Tag{
			{
				TagKey:   PtrString("Dept"),
				TagValue: PtrString("IT"),
			},
			{
				TagKey:   PtrString("Purpose"),
				TagValue: PtrString("Test"),
			},
			{
				TagKey:   PtrString("Name"),
				TagValue: PtrString("Test"),
			},
		},
	}, nil
}

func TestKMSGetFunc(t *testing.T) {
	ctx := context.Background()
	cli := kmsTestClient{}

	item, err := kmsKeyGetFunc(ctx, cli, "scope", &kms.DescribeKeyInput{
		KeyId: PtrString("1234abcd-12ab-34cd-56ef-1234567890ab"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestNewKMSKeyAdapter(t *testing.T) {
	t.Skip("This test is currently failing due to a key that none of us can read, even with admin permissions. I think we will need to speak with AWS support to work out how to delete it", nil)
	config, account, region := GetAutoConfig(t)
	client := kms.NewFromConfig(config)

	adapter := NewKMSKeyAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
