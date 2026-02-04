package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestContinuousDeploymentPolicyItemMapper(t *testing.T) {
	item, err := continuousDeploymentPolicyItemMapper("", "test", &types.ContinuousDeploymentPolicy{
		Id:               PtrString("test-id"),
		LastModifiedTime: PtrTime(time.Now()),
		ContinuousDeploymentPolicyConfig: &types.ContinuousDeploymentPolicyConfig{
			Enabled: PtrBool(true),
			StagingDistributionDnsNames: &types.StagingDistributionDnsNames{
				Quantity: PtrInt32(1),
				Items: []string{
					"staging.test.com", // link
				},
			},
			TrafficConfig: &types.TrafficConfig{
				Type: types.ContinuousDeploymentPolicyTypeSingleWeight,
				SingleHeaderConfig: &types.ContinuousDeploymentSingleHeaderConfig{
					Header: PtrString("test-header"),
					Value:  PtrString("test-value"),
				},
				SingleWeightConfig: &types.ContinuousDeploymentSingleWeightConfig{
					Weight: PtrFloat32(1),
					SessionStickinessConfig: &types.SessionStickinessConfig{
						IdleTTL:    PtrInt32(1),
						MaximumTTL: PtrInt32(2),
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

	tests := QueryTests{
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

	adapter := NewCloudfrontContinuousDeploymentPolicyAdapter(client, account, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
