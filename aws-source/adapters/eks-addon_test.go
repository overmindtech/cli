package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

var AddonTestClient = EKSTestClient{
	DescribeAddonOutput: &eks.DescribeAddonOutput{
		Addon: &types.Addon{
			AddonName:           new("aws-ebs-csi-driver"),
			ClusterName:         new("dylan"),
			Status:              types.AddonStatusActive,
			AddonVersion:        new("v1.13.0-eksbuild.3"),
			ConfigurationValues: new("values"),
			MarketplaceInformation: &types.MarketplaceInformation{
				ProductId:  new("id"),
				ProductUrl: new("url"),
			},
			Publisher: new("publisher"),
			Owner:     new("owner"),
			Health: &types.AddonHealth{
				Issues: []types.AddonIssue{},
			},
			AddonArn:              new("arn:aws:eks:eu-west-2:801795385023:addon/dylan/aws-ebs-csi-driver/a2c29d0e-72c4-a702-7887-2f739f4fc189"),
			CreatedAt:             new(time.Now()),
			ModifiedAt:            new(time.Now()),
			ServiceAccountRoleArn: new("arn:aws:iam::801795385023:role/eks-csi-dylan"),
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

	adapter := NewEKSAddonAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
