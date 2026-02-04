package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestHealthCheckItemMapper(t *testing.T) {
	hc := HealthCheck{
		HealthCheck: types.HealthCheck{
			Id:              PtrString("d7ce5d72-6d1f-4147-8246-d0ca3fb505d6"),
			CallerReference: PtrString("85d56b3f-873c-498b-a2dd-554ec13c5289"),
			HealthCheckConfig: &types.HealthCheckConfig{
				IPAddress:                PtrString("1.1.1.1"),
				Port:                     PtrInt32(443),
				Type:                     types.HealthCheckTypeHttps,
				FullyQualifiedDomainName: PtrString("one.one.one.one"),
				RequestInterval:          PtrInt32(30),
				FailureThreshold:         PtrInt32(3),
				MeasureLatency:           PtrBool(false),
				Inverted:                 PtrBool(false),
				Disabled:                 PtrBool(false),
				EnableSNI:                PtrBool(true),
			},
			HealthCheckVersion: PtrInt64(1),
		},
		HealthCheckObservations: []types.HealthCheckObservation{
			{
				Region:    types.HealthCheckRegionApNortheast1,
				IPAddress: PtrString("15.177.62.21"),
				StatusReport: &types.StatusReport{
					Status:      PtrString("Success: HTTP Status Code 200, OK"),
					CheckedTime: PtrTime(time.Now()),
				},
			},
			{
				Region:    types.HealthCheckRegionEuWest1,
				IPAddress: PtrString("15.177.10.21"),
				StatusReport: &types.StatusReport{
					Status:      PtrString("Failure: Connection timed out. The endpoint or the internet connection is down, or requests are being blocked by your firewall. See https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/dns-failover-router-firewall-rules.html"),
					CheckedTime: PtrTime(time.Now()),
				},
			},
		},
	}

	item, err := healthCheckItemMapper("", "foo", &hc)

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "cloudwatch-alarm",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "{\"MetricName\":\"HealthCheckStatus\",\"Namespace\":\"AWS/Route53\",\"Dimensions\":[{\"Name\":\"HealthCheckId\",\"Value\":\"d7ce5d72-6d1f-4147-8246-d0ca3fb505d6\"}],\"ExtendedStatistic\":null,\"Period\":null,\"Statistic\":\"\",\"Unit\":\"\"}",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewRoute53HealthCheckAdapter(t *testing.T) {
	client, account, region := route53GetAutoConfig(t)

	adapter := NewRoute53HealthCheckAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
