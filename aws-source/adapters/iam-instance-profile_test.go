package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestInstanceProfileItemMapper(t *testing.T) {
	profile := types.InstanceProfile{
		Arn:                 adapterhelpers.PtrString("arn:aws:iam::123456789012:instance-profile/webserver"),
		CreateDate:          adapterhelpers.PtrTime(time.Now()),
		InstanceProfileId:   adapterhelpers.PtrString("AIDACKCEVSQ6C2EXAMPLE"),
		InstanceProfileName: adapterhelpers.PtrString("webserver"),
		Path:                adapterhelpers.PtrString("/"),
		Roles: []types.Role{
			{
				Arn:                      adapterhelpers.PtrString("arn:aws:iam::123456789012:role/webserver"), // link
				CreateDate:               adapterhelpers.PtrTime(time.Now()),
				Path:                     adapterhelpers.PtrString("/"),
				RoleId:                   adapterhelpers.PtrString("AIDACKCEVSQ6C2EXAMPLE"),
				RoleName:                 adapterhelpers.PtrString("webserver"),
				AssumeRolePolicyDocument: adapterhelpers.PtrString(`{}`),
				Description:              adapterhelpers.PtrString("Allows EC2 instances to call AWS services on your behalf."),
				MaxSessionDuration:       adapterhelpers.PtrInt32(3600),
				PermissionsBoundary: &types.AttachedPermissionsBoundary{
					PermissionsBoundaryArn:  adapterhelpers.PtrString("arn:aws:iam::123456789012:policy/XCompanyBoundaries"), // link
					PermissionsBoundaryType: types.PermissionsBoundaryAttachmentTypePolicy,
				},
				RoleLastUsed: &types.RoleLastUsed{
					LastUsedDate: adapterhelpers.PtrTime(time.Now()),
					Region:       adapterhelpers.PtrString("us-east-1"),
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
	config, account, _ := adapterhelpers.GetAutoConfig(t)
	client := iam.NewFromConfig(config, func(o *iam.Options) {
		o.RetryMode = aws.RetryModeAdaptive
		o.RetryMaxAttempts = 10
	})

	adapter := NewIAMInstanceProfileAdapter(client, account)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 30 * time.Second,
	}

	test.Run(t)
}
