package adapters

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

func (t *TestIAMClient) GetPolicy(ctx context.Context, params *iam.GetPolicyInput, optFns ...func(*iam.Options)) (*iam.GetPolicyOutput, error) {
	return &iam.GetPolicyOutput{
		Policy: &types.Policy{
			PolicyName:                    adapterhelpers.PtrString("AWSControlTowerStackSetRolePolicy"),
			PolicyId:                      adapterhelpers.PtrString("ANPA3VLV2U277MP54R2OV"),
			Arn:                           adapterhelpers.PtrString("arn:aws:iam::801795385023:policy/service-role/AWSControlTowerStackSetRolePolicy"),
			Path:                          adapterhelpers.PtrString("/service-role/"),
			DefaultVersionId:              adapterhelpers.PtrString("v1"),
			AttachmentCount:               adapterhelpers.PtrInt32(1),
			PermissionsBoundaryUsageCount: adapterhelpers.PtrInt32(0),
			IsAttachable:                  true,
			CreateDate:                    adapterhelpers.PtrTime(time.Now()),
			UpdateDate:                    adapterhelpers.PtrTime(time.Now()),
		},
	}, nil
}

func (t *TestIAMClient) ListEntitiesForPolicy(context.Context, *iam.ListEntitiesForPolicyInput, ...func(*iam.Options)) (*iam.ListEntitiesForPolicyOutput, error) {
	return &iam.ListEntitiesForPolicyOutput{
		PolicyGroups: []types.PolicyGroup{
			{
				GroupId:   adapterhelpers.PtrString("groupId"),
				GroupName: adapterhelpers.PtrString("groupName"),
			},
		},
		PolicyRoles: []types.PolicyRole{
			{
				RoleId:   adapterhelpers.PtrString("roleId"),
				RoleName: adapterhelpers.PtrString("roleName"),
			},
		},
		PolicyUsers: []types.PolicyUser{
			{
				UserId:   adapterhelpers.PtrString("userId"),
				UserName: adapterhelpers.PtrString("userName"),
			},
		},
	}, nil
}

func (t *TestIAMClient) ListPolicies(context.Context, *iam.ListPoliciesInput, ...func(*iam.Options)) (*iam.ListPoliciesOutput, error) {
	return &iam.ListPoliciesOutput{
		Policies: []types.Policy{
			{
				PolicyName:                    adapterhelpers.PtrString("AWSControlTowerAdminPolicy"),
				PolicyId:                      adapterhelpers.PtrString("ANPA3VLV2U2745H37HTHN"),
				Arn:                           adapterhelpers.PtrString("arn:aws:iam::801795385023:policy/service-role/AWSControlTowerAdminPolicy"),
				Path:                          adapterhelpers.PtrString("/service-role/"),
				DefaultVersionId:              adapterhelpers.PtrString("v1"),
				AttachmentCount:               adapterhelpers.PtrInt32(1),
				PermissionsBoundaryUsageCount: adapterhelpers.PtrInt32(0),
				IsAttachable:                  true,
				CreateDate:                    adapterhelpers.PtrTime(time.Now()),
				UpdateDate:                    adapterhelpers.PtrTime(time.Now()),
			},
			{
				PolicyName:                    adapterhelpers.PtrString("AWSControlTowerCloudTrailRolePolicy"),
				PolicyId:                      adapterhelpers.PtrString("ANPA3VLV2U27UOP7KSM6I"),
				Arn:                           adapterhelpers.PtrString("arn:aws:iam::801795385023:policy/service-role/AWSControlTowerCloudTrailRolePolicy"),
				Path:                          adapterhelpers.PtrString("/service-role/"),
				DefaultVersionId:              adapterhelpers.PtrString("v1"),
				AttachmentCount:               adapterhelpers.PtrInt32(1),
				PermissionsBoundaryUsageCount: adapterhelpers.PtrInt32(0),
				IsAttachable:                  true,
				CreateDate:                    adapterhelpers.PtrTime(time.Now()),
				UpdateDate:                    adapterhelpers.PtrTime(time.Now()),
			},
		},
	}, nil
}

func (t *TestIAMClient) ListPolicyTags(ctx context.Context, params *iam.ListPolicyTagsInput, optFns ...func(*iam.Options)) (*iam.ListPolicyTagsOutput, error) {
	return &iam.ListPolicyTagsOutput{
		Tags: []types.Tag{
			{
				Key:   adapterhelpers.PtrString("foo"),
				Value: adapterhelpers.PtrString("foo"),
			},
		},
	}, nil
}

const testPolicy = `{
    "Version": "2012-10-17",
    "Statement": {
        "Effect": "Allow",
        "Action": [
            "iam:AddUserToGroup",
            "iam:RemoveUserFromGroup",
            "iam:GetGroup"
        ],
        "Resource": [
            "arn:aws:iam::609103258633:group/Developers",
            "arn:aws:iam::609103258633:group/Operators",
			"arn:aws:iam::609103258633:user/*"
        ]
    }
}`

func (c *TestIAMClient) GetPolicyVersion(ctx context.Context, params *iam.GetPolicyVersionInput, optFns ...func(*iam.Options)) (*iam.GetPolicyVersionOutput, error) {
	create := time.Now()
	document := url.QueryEscape(testPolicy)
	versionId := "v2"

	return &iam.GetPolicyVersionOutput{
		PolicyVersion: &types.PolicyVersion{
			CreateDate:       &create,
			Document:         &document,
			IsDefaultVersion: true,
			VersionId:        &versionId,
		},
	}, nil
}

func TestGetCurrentPolicyVersion(t *testing.T) {
	client := &TestIAMClient{}
	ctx := context.Background()

	t.Run("with a good query", func(t *testing.T) {
		arn := "arn:aws:iam::609103258633:policy/DevelopersPolicy"
		version := "v2"
		policy := PolicyDetails{
			Policy: &types.Policy{
				Arn:              &arn,
				DefaultVersionId: &version,
			},
		}

		err := addPolicyDocument(ctx, client, &policy)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with empty values", func(t *testing.T) {
		arn := ""
		version := ""
		policy := PolicyDetails{
			Policy: &types.Policy{
				Arn:              &arn,
				DefaultVersionId: &version,
			},
		}

		err := addPolicyDocument(ctx, client, &policy)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with nil", func(t *testing.T) {
		policy := PolicyDetails{}

		err := addPolicyDocument(ctx, client, &policy)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPolicyGetFunc(t *testing.T) {
	policy, err := policyGetFunc(context.Background(), &TestIAMClient{}, "foo", "bar")
	if err != nil {
		t.Error(err)
	}

	if policy.Policy == nil {
		t.Error("policy was nil")
	}

	if len(policy.PolicyGroups) != 1 {
		t.Errorf("expected 1 Group, got %v", len(policy.PolicyGroups))
	}

	if len(policy.PolicyRoles) != 1 {
		t.Errorf("expected 1 Role, got %v", len(policy.PolicyRoles))
	}

	if len(policy.PolicyUsers) != 1 {
		t.Errorf("expected 1 User, got %v", len(policy.PolicyUsers))
	}

	if policy.Document.Version != "2012-10-17" {
		t.Errorf("expected version 2012-10-17, got %v", policy.Document.Version)
	}

	if len(policy.Document.Statements.Values()) != 1 {
		t.Errorf("expected 1 statement, got %v", len(policy.Document.Statements.Values()))
	}
}

func TestPolicyListTagsFunc(t *testing.T) {
	tags, err := policyListTagsFunc(context.Background(), &PolicyDetails{
		Policy: &types.Policy{
			Arn: adapterhelpers.PtrString("arn:aws:iam::801795385023:policy/service-role/AWSControlTowerAdminPolicy"),
		},
	}, &TestIAMClient{})
	if err != nil {
		t.Error(err)
	}

	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %v", len(tags))
	}
}

func TestPolicyItemMapper(t *testing.T) {
	details := &PolicyDetails{
		Policy: &types.Policy{
			PolicyName:                    adapterhelpers.PtrString("AWSControlTowerAdminPolicy"),
			PolicyId:                      adapterhelpers.PtrString("ANPA3VLV2U2745H37HTHN"),
			Arn:                           adapterhelpers.PtrString("arn:aws:iam::801795385023:policy/service-role/AWSControlTowerAdminPolicy"),
			Path:                          adapterhelpers.PtrString("/service-role/"),
			DefaultVersionId:              adapterhelpers.PtrString("v1"),
			AttachmentCount:               adapterhelpers.PtrInt32(1),
			PermissionsBoundaryUsageCount: adapterhelpers.PtrInt32(0),
			IsAttachable:                  true,
			CreateDate:                    adapterhelpers.PtrTime(time.Now()),
			UpdateDate:                    adapterhelpers.PtrTime(time.Now()),
		},
		PolicyGroups: []types.PolicyGroup{
			{
				GroupId:   adapterhelpers.PtrString("groupId"),
				GroupName: adapterhelpers.PtrString("groupName"),
			},
		},
		PolicyRoles: []types.PolicyRole{
			{
				RoleId:   adapterhelpers.PtrString("roleId"),
				RoleName: adapterhelpers.PtrString("roleName"),
			},
		},
		PolicyUsers: []types.PolicyUser{
			{
				UserId:   adapterhelpers.PtrString("userId"),
				UserName: adapterhelpers.PtrString("userName"),
			},
		},
	}
	err := addPolicyDocument(context.Background(), &TestIAMClient{}, details)
	if err != nil {
		t.Fatal(err)
	}
	item, err := policyItemMapper(nil, "foo", details)
	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "iam-group",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "groupName",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "iam-user",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "userName",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "roleName",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "iam-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:iam::609103258633:group/Developers",
			ExpectedScope:  "609103258633",
		},
		{
			ExpectedType:   "iam-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:iam::609103258633:group/Operators",
			ExpectedScope:  "609103258633",
		},
	}

	tests.Execute(t, item)

	if item.UniqueAttributeValue() != "service-role/AWSControlTowerAdminPolicy" {
		t.Errorf("unexpected unique attribute value, got %s", item.UniqueAttributeValue())
	}
}

func TestNewIAMPolicyAdapter(t *testing.T) {
	config, account, _ := adapterhelpers.GetAutoConfig(t)
	client := iam.NewFromConfig(config, func(o *iam.Options) {
		o.RetryMode = aws.RetryModeAdaptive
		o.RetryMaxAttempts = 10
	})

	adapter := NewIAMPolicyAdapter(client, account)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 30 * time.Second,
	}

	test.Run(t)

	// Test "aws" scoped resources
	t.Run("aws scoped resources in a specific scope", func(t *testing.T) {
		ctx, span := tracer.Start(context.Background(), t.Name())

		defer span.End()

		t.Parallel()
		// This item shouldn't be found since it lives globally
		_, err := adapter.Get(ctx, adapterhelpers.FormatScope(account, ""), "ReadOnlyAccess", false)

		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("aws scoped resources in the aws scope", func(t *testing.T) {
		ctx, span := tracer.Start(context.Background(), t.Name())
		defer span.End()

		t.Parallel()
		// This item shouldn't be found since it lives globally
		item, err := adapter.Get(ctx, "aws", "ReadOnlyAccess", false)
		if err != nil {
			t.Error(err)
		}

		if item.UniqueAttributeValue() != "ReadOnlyAccess" {
			t.Errorf("expected globally unique name to be ReadOnlyAccess, got %v", item.GloballyUniqueName())
		}
	})

	t.Run("listing resources in a specific scope", func(t *testing.T) {
		ctx, span := tracer.Start(context.Background(), t.Name())
		defer span.End()

		stream := discovery.NewRecordingQueryResultStream()
		adapter.ListStream(ctx, adapterhelpers.FormatScope(account, ""), false, stream)

		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Error(errs)
		}

		for _, item := range stream.GetItems() {
			arnString, err := item.GetAttributes().Get("Arn")
			if err != nil {
				t.Errorf("expected item to have an arn attribute, got %v", err)
			}

			arn, err := adapterhelpers.ParseARN(arnString.(string))
			if err != nil {
				t.Error(err)
			}

			if arn.AccountID != account {
				t.Errorf("expected item account to be %v, got %v", account, arn.AccountID)
			}
		}

		if len(stream.GetItems()) == 0 {
			t.Fatal("no items found")
		}

		arn, _ := stream.GetItems()[0].GetAttributes().Get("Arn")

		t.Run("searching via ARN for a resource in a specific scope", func(t *testing.T) {
			ctx, span := tracer.Start(context.Background(), t.Name())
			defer span.End()

			t.Parallel()

			stream := discovery.NewRecordingQueryResultStream()
			adapter.SearchStream(ctx, adapterhelpers.FormatScope(account, ""), arn.(string), false, stream)

			errs := stream.GetErrors()
			if len(errs) > 0 {
				t.Error(errs)
			}
		})

		t.Run("searching via ARN for a resource in the aws scope", func(t *testing.T) {
			ctx, span := tracer.Start(context.Background(), t.Name())
			defer span.End()

			t.Parallel()

			stream := discovery.NewRecordingQueryResultStream()
			adapter.SearchStream(ctx, "aws", arn.(string), false, stream)

			if len(errs) == 0 {
				t.Error("expected error, got nil")
			}
		})
	})

	t.Run("listing resources in the AWS scope", func(t *testing.T) {
		ctx, span := tracer.Start(context.Background(), t.Name())
		defer span.End()

		stream := discovery.NewRecordingQueryResultStream()
		adapter.ListStream(ctx, "aws", false, stream)

		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Error(errs)
		}

		items := stream.GetItems()
		if len(items) == 0 {
			t.Fatal("expected items, got none")
		}

		for _, item := range items {
			arnString, err := item.GetAttributes().Get("Arn")
			if err != nil {
				t.Errorf("expected item to have an arn attribute, got %v", err)
			}

			arn, err := adapterhelpers.ParseARN(arnString.(string))
			if err != nil {
				t.Error(err)
			}

			if arn.AccountID != "aws" {
				t.Errorf("expected item account to be aws, got %v", arn.AccountID)
			}
		}

		t.Run("searching via ARN for a resource in a specific scope", func(t *testing.T) {
			ctx, span := tracer.Start(context.Background(), t.Name())
			defer span.End()

			t.Parallel()

			arn, _ := items[0].GetAttributes().Get("Arn")
			stream := discovery.NewRecordingQueryResultStream()
			adapter.SearchStream(ctx, adapterhelpers.FormatScope(account, ""), arn.(string), false, stream)

			errs := stream.GetErrors()
			if len(errs) == 0 {
				t.Error("expected error, got nil")
			}
		})

		t.Run("searching via ARN for a resource in the aws scope", func(t *testing.T) {
			ctx, span := tracer.Start(context.Background(), t.Name())
			defer span.End()

			t.Parallel()

			arn, _ := items[0].GetAttributes().Get("Arn")
			stream := discovery.NewRecordingQueryResultStream()
			adapter.SearchStream(ctx, "aws", arn.(string), false, stream)

			errs := stream.GetErrors()
			if len(errs) > 0 {
				t.Error(errs)
			}
		})
	})
}
