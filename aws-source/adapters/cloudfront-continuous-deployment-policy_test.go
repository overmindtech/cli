package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestContinuousDeploymentPolicyItemMapper(t *testing.T) {
	item, err := continuousDeploymentPolicyItemMapper("", "test", &types.ContinuousDeploymentPolicy{
		Id:               adapterhelpers.PtrString("test-id"),
		LastModifiedTime: adapterhelpers.PtrTime(time.Now()),
		ContinuousDeploymentPolicyConfig: &types.ContinuousDeploymentPolicyConfig{
			Enabled: adapterhelpers.PtrBool(true),
			StagingDistributionDnsNames: &types.StagingDistributionDnsNames{
				Quantity: adapterhelpers.PtrInt32(1),
				Items: []string{
					"staging.test.com", // link
				},
			},
			TrafficConfig: &types.TrafficConfig{
				Type: types.ContinuousDeploymentPolicyTypeSingleWeight,
				SingleHeaderConfig: &types.ContinuousDeploymentSingleHeaderConfig{
					Header: adapterhelpers.PtrString("test-header"),
					Value:  adapterhelpers.PtrString("test-value"),
				},
				SingleWeightConfig: &types.ContinuousDeploymentSingleWeightConfig{
					Weight: adapterhelpers.PtrFloat32(1),
					SessionStickinessConfig: &types.SessionStickinessConfig{
						IdleTTL:    adapterhelpers.PtrInt32(1),
						MaximumTTL: adapterhelpers.PtrInt32(2),
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "staging.test.com",
			ExpectedScope:  "global",
		},
	}

	tests.Execute(t, item)
}

func TestNewCloudfrontContinuousDeploymentPolicyAdapter(t *testing.T) {
	client, account, _ := CloudfrontGetAutoConfig(t)

	adapter := NewCloudfrontContinuousDeploymentPolicyAdapter(client, account)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
