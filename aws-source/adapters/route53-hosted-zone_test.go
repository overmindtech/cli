package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestHostedZoneItemMapper(t *testing.T) {
	zone := types.HostedZone{
		Id:              new("/hostedzone/Z08416862SZP5DJXIDB29"),
		Name:            new("overmind-demo.com."),
		CallerReference: new("RISWorkflow-RD:144d3779-1574-42bf-9e75-f309838ea0a1"),
		Config: &types.HostedZoneConfig{
			Comment:     new("HostedZone created by Route53 Registrar"),
			PrivateZone: false,
		},
		ResourceRecordSetCount: new(int64(3)),
		LinkedService: &types.LinkedService{
			Description:      new("service description"),
			ServicePrincipal: new("principal"),
		},
	}

	item, err := hostedZoneItemMapper("", "foo", &zone)

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "route53-resource-record-set",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "/hostedzone/Z08416862SZP5DJXIDB29",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewRoute53HostedZoneAdapter(t *testing.T) {
	client, account, region := route53GetAutoConfig(t)

	adapter := NewRoute53HostedZoneAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
