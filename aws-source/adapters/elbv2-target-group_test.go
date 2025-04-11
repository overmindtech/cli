package adapters

import (
	"context"
	"testing"
	"time"

	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestTargetGroupOutputMapper(t *testing.T) {
	output := elbv2.DescribeTargetGroupsOutput{
		TargetGroups: []types.TargetGroup{
			{
				TargetGroupArn:             adapterhelpers.PtrString("arn:aws:elasticloadbalancing:eu-west-2:944651592624:targetgroup/k8s-default-apiserve-d87e8f7010/559d207158e41222"),
				TargetGroupName:            adapterhelpers.PtrString("k8s-default-apiserve-d87e8f7010"),
				Protocol:                   types.ProtocolEnumHttp,
				Port:                       adapterhelpers.PtrInt32(8080),
				VpcId:                      adapterhelpers.PtrString("vpc-0c72199250cd479ea"), // link
				HealthCheckProtocol:        types.ProtocolEnumHttp,
				HealthCheckPort:            adapterhelpers.PtrString("traffic-port"),
				HealthCheckEnabled:         adapterhelpers.PtrBool(true),
				HealthCheckIntervalSeconds: adapterhelpers.PtrInt32(10),
				HealthCheckTimeoutSeconds:  adapterhelpers.PtrInt32(10),
				HealthyThresholdCount:      adapterhelpers.PtrInt32(10),
				UnhealthyThresholdCount:    adapterhelpers.PtrInt32(10),
				HealthCheckPath:            adapterhelpers.PtrString("/"),
				Matcher: &types.Matcher{
					HttpCode: adapterhelpers.PtrString("200"),
					GrpcCode: adapterhelpers.PtrString("code"),
				},
				LoadBalancerArns: []string{
					"arn:aws:elasticloadbalancing:eu-west-2:944651592624:loadbalancer/app/ingress/1bf10920c5bd199d", // link
				},
				TargetType:      types.TargetTypeEnumIp,
				ProtocolVersion: adapterhelpers.PtrString("HTTP1"),
				IpAddressType:   types.TargetGroupIpAddressTypeEnumIpv4,
			},
		},
	}

	items, err := targetGroupOutputMapper(context.Background(), mockElbv2Client{}, "foo", nil, &output)

	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		if item.GetTags()["foo"] != "bar" {
			t.Errorf("expected tag foo to be bar, got %v", item.GetTags()["foo"])
		}

		if err := item.Validate(); err != nil {
			t.Error(err)
		}
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", len(items))
	}

	item := items[0]

	// It doesn't really make sense to test anything other than the linked items
	// since the attributes are converted automatically
	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "ec2-vpc",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "vpc-0c72199250cd479ea",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "elbv2-load-balancer",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:elasticloadbalancing:eu-west-2:944651592624:loadbalancer/app/ingress/1bf10920c5bd199d",
			ExpectedScope:  "944651592624.eu-west-2",
		},
	}

	tests.Execute(t, item)
}

func TestNewELBv2TargetGroupAdapter(t *testing.T) {
	config, account, region := adapterhelpers.GetAutoConfig(t)
	client := elbv2.NewFromConfig(config)

	adapter := NewELBv2TargetGroupAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
