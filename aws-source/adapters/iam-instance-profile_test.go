package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/overmindtech/cli/sdpcache"
)

func TestInstanceProfileItemMapper(t *testing.T) {
	profile := types.InstanceProfile{
		Arn:                 PtrString("arn:aws:iam::123456789012:instance-profile/webserver"),
		CreateDate:          PtrTime(time.Now()),
		InstanceProfileId:   PtrString("AIDACKCEVSQ6C2EXAMPLE"),
		InstanceProfileName: PtrString("webserver"),
		Path:                PtrString("/"),
		Roles: []types.Role{
			{
				Arn:                      PtrString("arn:aws:iam::123456789012:role/webserver"), // link
				CreateDate:               PtrTime(time.Now()),
				Path:                     PtrString("/"),
				RoleId:                   PtrString("AIDACKCEVSQ6C2EXAMPLE"),
				RoleName:                 PtrString("webserver"),
				AssumeRolePolicyDocument: PtrString(`{}`),
				Description:              PtrString("Allows EC2 instances to call AWS services on your behalf."),
				MaxSessionDuration:       PtrInt32(3600),
				PermissionsBoundary: &types.AttachedPermissionsBoundary{
					PermissionsBoundaryArn:  PtrString("arn:aws:iam::123456789012:policy/XCompanyBoundaries"), // link
					PermissionsBoundaryType: types.PermissionsBoundaryAttachmentTypePolicy,
				},
				RoleLastUsed: &types.RoleLastUsed{
					LastUsedDate: PtrTime(time.Now()),
					Region:       PtrString("us-east-1"),
				},
			},
		},
	}

	item, err := instanceProfileItemMapper(nil, "foo", &profile)

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

}

func TestNewIAMInstanceProfileAdapter(t *testing.T) {
	config, account, _ := GetAutoConfig(t)
	client := iam.NewFromConfig(config, func(o *iam.Options) {
		o.RetryMode = aws.RetryModeAdaptive
		o.RetryMaxAttempts = 10
	})

	adapter := NewIAMInstanceProfileAdapter(client, account, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 30 * time.Second,
	}

	test.Run(t)
}
