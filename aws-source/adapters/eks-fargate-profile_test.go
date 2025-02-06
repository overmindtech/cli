package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

var FargateTestClient = EKSTestClient{
	DescribeFargateProfileOutput: &eks.DescribeFargateProfileOutput{
		FargateProfile: &types.FargateProfile{
			ClusterName:         adapterhelpers.PtrString("cluster"),
			CreatedAt:           adapterhelpers.PtrTime(time.Now()),
			FargateProfileArn:   adapterhelpers.PtrString("arn:partition:service:region:account-id:resource-type/resource-id"),
			FargateProfileName:  adapterhelpers.PtrString("name"),
			PodExecutionRoleArn: adapterhelpers.PtrString("arn:partition:service::account-id:resource-type/resource-id"),
			Selectors: []types.FargateProfileSelector{
				{
					Labels:    map[string]string{},
					Namespace: adapterhelpers.PtrString("namespace"),
				},
			},
			Status: types.FargateProfileStatusActive,
			Subnets: []string{
				"subnet",
			},
			Tags: map[string]string{},
		},
	},
}

func TestFargateProfileGetFunc(t *testing.T) {
	item, err := fargateProfileGetFunc(context.Background(), FargateTestClient, "foo", &eks.DescribeFargateProfileInput{})

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "iam-role",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:partition:service::account-id:resource-type/resource-id",
			ExpectedScope:  "account-id",
		},
		{
			ExpectedType:   "ec2-subnet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "subnet",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewEKSFargateProfileAdapter(t *testing.T) {
	client, account, region := eksGetAutoConfig(t)

	adapter := NewEKSFargateProfileAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter:           adapter,
		Timeout:           10 * time.Second,
		SkipNotFoundCheck: true,
	}

	test.Run(t)
}
