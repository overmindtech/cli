package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

/*
An example list grants response:

{
    "Grants": [
        {
            "Constraints": {
                "EncryptionContextSubset": {
                    "aws:dynamodb:subscriberId": "123456789012",
                    "aws:dynamodb:tableName": "Services"
                }
            },
            "IssuingAccount": "arn:aws:iam::123456789012:root",
            "Name": "8276b9a6-6cf0-46f1-b2f0-7993a7f8c89a",
            "Operations": [
                "Decrypt",
                "Encrypt",
                "GenerateDataKey",
                "ReEncryptFrom",
                "ReEncryptTo",
                "RetireGrant",
                "DescribeKey"
            ],
            "GrantId": "1667b97d27cf748cf05b487217dd4179526c949d14fb3903858e25193253fe59",
            "KeyId": "arn:aws:kms:us-west-2:123456789012:key/1234abcd-12ab-34cd-56ef-1234567890ab",
            "RetiringPrincipal": "dynamodb.us-west-2.amazonaws.com",
            "GranteePrincipal": "dynamodb.us-west-2.amazonaws.com",
            "CreationDate": "2021-05-13T18:32:45.144000+00:00"
        }
    ]
}
*/

func TestIsAWSServicePrincipal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		principal string
		expected  bool
	}{
		{
			name:      "RDS service principal",
			principal: "rds.eu-west-2.amazonaws.com",
			expected:  true,
		},
		{
			name:      "DynamoDB service principal",
			principal: "dynamodb.us-west-2.amazonaws.com",
			expected:  true,
		},
		{
			name:      "EC2 service principal",
			principal: "ec2.amazonaws.com",
			expected:  true,
		},
		{
			name:      "China region service principal (aws-cn)",
			principal: "rds.cn-north-1.amazonaws.com.cn",
			expected:  true,
		},
		{
			name:      "EU partition service principal (aws-eu)",
			principal: "rds.eu-central-1.amazonaws.eu",
			expected:  true,
		},
		{
			name:      "ISO partition service principal (aws-iso)",
			principal: "rds.us-iso-east-1.c2s.ic.gov",
			expected:  true,
		},
		{
			name:      "ISO-B partition service principal (aws-iso-b)",
			principal: "rds.us-isob-east-1.sc2s.sgov.gov",
			expected:  true,
		},
		{
			name:      "IAM role ARN",
			principal: "arn:aws:iam::123456789012:role/MyRole",
			expected:  false,
		},
		{
			name:      "IAM user ARN",
			principal: "arn:aws:iam::123456789012:user/MyUser",
			expected:  false,
		},
		{
			name:      "Account root ARN",
			principal: "arn:aws:iam::123456789012:root",
			expected:  false,
		},
		{
			name:      "Random string",
			principal: "not-a-principal",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isAWSServicePrincipal(tt.principal)
			if result != tt.expected {
				t.Errorf("isAWSServicePrincipal(%q) = %v, expected %v", tt.principal, result, tt.expected)
			}
		})
	}
}

func TestGrantOutputMapper(t *testing.T) {
	output := &kms.ListGrantsOutput{
		Grants: []types.GrantListEntry{
			{
				Constraints: &types.GrantConstraints{
					EncryptionContextSubset: map[string]string{
						"aws:dynamodb:subscriberId": "123456789012",
						"aws:dynamodb:tableName":    "Services",
					},
				},
				IssuingAccount:    PtrString("arn:aws:iam::123456789012:root"),
				Name:              PtrString("8276b9a6-6cf0-46f1-b2f0-7993a7f8c89a"),
				Operations:        []types.GrantOperation{"Decrypt", "Encrypt", "GenerateDataKey", "ReEncryptFrom", "ReEncryptTo", "RetireGrant", "DescribeKey"},
				GrantId:           PtrString("1667b97d27cf748cf05b487217dd4179526c949d14fb3903858e25193253fe59"),
				KeyId:             PtrString("arn:aws:kms:us-west-2:123456789012:key/1234abcd-12ab-34cd-56ef-1234567890ab"),
				RetiringPrincipal: PtrString("arn:aws:iam::account:role/role-name-with-path"),
				GranteePrincipal:  PtrString("arn:aws:iam::account:user/user-name-with-path"),
				CreationDate:      PtrTime(time.Now()),
			},
		},
	}

	items, err := grantOutputMapper(context.Background(), nil, "foo", nil, output)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	scope := FormatScope("123456789012", "us-west-2")

	tests := QueryTests{
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "1234abcd-12ab-34cd-56ef-1234567890ab",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "role-name-with-path",
			ExpectedScope:  scope,
		},
		{
			ExpectedType:   "iam-user",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "user-name-with-path",
			ExpectedScope:  scope,
		},
	}

	tests.Execute(t, item)
}

func TestGrantOutputMapperWithServicePrincipal(t *testing.T) {
	// Test that service principals (like dynamodb.us-west-2.amazonaws.com) are
	// properly skipped and don't cause errors or generate linked item queries
	output := &kms.ListGrantsOutput{
		Grants: []types.GrantListEntry{
			{
				Constraints: &types.GrantConstraints{
					EncryptionContextSubset: map[string]string{
						"aws:dynamodb:subscriberId": "123456789012",
						"aws:dynamodb:tableName":    "Services",
					},
				},
				IssuingAccount: PtrString("arn:aws:iam::123456789012:root"),
				Name:           PtrString("8276b9a6-6cf0-46f1-b2f0-7993a7f8c89a"),
				Operations:     []types.GrantOperation{"Decrypt", "Encrypt"},
				GrantId:        PtrString("1667b97d27cf748cf05b487217dd4179526c949d14fb3903858e25193253fe59"),
				KeyId:          PtrString("arn:aws:kms:us-west-2:123456789012:key/1234abcd-12ab-34cd-56ef-1234567890ab"),
				// These are service principals, not ARNs - they should be skipped
				RetiringPrincipal: PtrString("dynamodb.us-west-2.amazonaws.com"),
				GranteePrincipal:  PtrString("rds.eu-west-2.amazonaws.com"),
				CreationDate:      PtrTime(time.Now()),
			},
		},
	}

	items, err := grantOutputMapper(context.Background(), nil, "foo", nil, output)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	// Should only have the kms-key link, not the service principals
	if len(item.GetLinkedItemQueries()) != 1 {
		t.Errorf("expected 1 linked item query (kms-key only), got %v", len(item.GetLinkedItemQueries()))
		for i, liq := range item.GetLinkedItemQueries() {
			t.Logf("  [%d] type=%s query=%s", i, liq.GetQuery().GetType(), liq.GetQuery().GetQuery())
		}
	}

	if item.GetLinkedItemQueries()[0].GetQuery().GetType() != "kms-key" {
		t.Errorf("expected linked item query to be kms-key, got %s", item.GetLinkedItemQueries()[0].GetQuery().GetType())
	}
}

func TestNewKMSGrantAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)
	client := kms.NewFromConfig(config)

	adapter := NewKMSGrantAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter:  adapter,
		Timeout:  10 * time.Second,
		SkipList: true,
	}

	test.Run(t)
}
