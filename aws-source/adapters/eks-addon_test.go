package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

var AddonTestClient = EKSTestClient{
	DescribeAddonOutput: &eks.DescribeAddonOutput{
		Addon: &types.Addon{
			AddonName:           adapterhelpers.PtrString("aws-ebs-csi-driver"),
			ClusterName:         adapterhelpers.PtrString("dylan"),
			Status:              types.AddonStatusActive,
			AddonVersion:        adapterhelpers.PtrString("v1.13.0-eksbuild.3"),
			ConfigurationValues: adapterhelpers.PtrString("values"),
			MarketplaceInformation: &types.MarketplaceInformation{
				ProductId:  adapterhelpers.PtrString("id"),
				ProductUrl: adapterhelpers.PtrString("url"),
			},
			Publisher: adapterhelpers.PtrString("publisher"),
			Owner:     adapterhelpers.PtrString("owner"),
			Health: &types.AddonHealth{
				Issues: []types.AddonIssue{},
			},
			AddonArn:              adapterhelpers.PtrString("arn:aws:eks:eu-west-2:801795385023:addon/dylan/aws-ebs-csi-driver/a2c29d0e-72c4-a702-7887-2f739f4fc189"),
			CreatedAt:             adapterhelpers.PtrTime(time.Now()),
			ModifiedAt:            adapterhelpers.PtrTime(time.Now()),
			ServiceAccountRoleArn: adapterhelpers.PtrString("arn:aws:iam::801795385023:role/eks-csi-dylan"),
		},
	},
}

func TestAddonGetFunc(t *testing.T) {
	item, err := addonGetFunc(context.Background(), AddonTestClient, "foo", &eks.DescribeAddonInput{})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}
}

func TestNewEKSAddonAdapter(t *testing.T) {
	client, account, region := eksGetAutoConfig(t)

	adapter := NewEKSAddonAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
