package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestInstanceProfileItemMapper(t *testing.T) {
	profile := types.InstanceProfile{
		Arn:                 new("arn:aws:iam::123456789012:instance-profile/webserver"),
		CreateDate:          new(time.Now()),
		InstanceProfileId:   new("AIDACKCEVSQ6C2EXAMPLE"),
		InstanceProfileName: new("webserver"),
		Path:                new("/"),
		Roles: []types.Role{
			{
				Arn:                      new("arn:aws:iam::123456789012:role/webserver"), // link
				CreateDate:               new(time.Now()),
				Path:                     new("/"),
				RoleId:                   new("AIDACKCEVSQ6C2EXAMPLE"),
				RoleName:                 new("webserver"),
				AssumeRolePolicyDocument: new(`{}`),
				Description:              new("Allows EC2 instances to call AWS services on your behalf."),
				MaxSessionDuration:       new(int32(3600)),
				PermissionsBoundary: &types.AttachedPermissionsBoundary{
					PermissionsBoundaryArn:  new("arn:aws:iam::123456789012:policy/XCompanyBoundaries"), // link
					PermissionsBoundaryType: types.PermissionsBoundaryAttachmentTypePolicy,
				},
				RoleLastUsed: &types.RoleLastUsed{
					LastUsedDate: new(time.Now()),
					Region:       new("us-east-1"),
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
