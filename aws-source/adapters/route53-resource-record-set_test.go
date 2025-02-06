package adapters

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func TestResourceRecordSetItemMapper(t *testing.T) {
	recordSet := types.ResourceRecordSet{
		Name: adapterhelpers.PtrString("overmind-demo.com."),
		Type: types.RRTypeNs,
		TTL:  adapterhelpers.PtrInt64(172800),
		GeoProximityLocation: &types.GeoProximityLocation{
			AWSRegion:      adapterhelpers.PtrString("us-east-1"),
			Bias:           adapterhelpers.PtrInt32(100),
			Coordinates:    &types.Coordinates{},
			LocalZoneGroup: adapterhelpers.PtrString("group"),
		},
		ResourceRecords: []types.ResourceRecord{
			{
				Value: adapterhelpers.PtrString("ns-1673.awsdns-17.co.uk."), // link
			},
			{
				Value: adapterhelpers.PtrString("ns-1505.awsdns-60.org."), // link
			},
			{
				Value: adapterhelpers.PtrString("ns-955.awsdns-55.net."), // link
			},
			{
				Value: adapterhelpers.PtrString("ns-276.awsdns-34.com."), // link
			},
		},
		AliasTarget: &types.AliasTarget{
			DNSName:              adapterhelpers.PtrString("foo.bar.com"), // link
			EvaluateTargetHealth: true,
			HostedZoneId:         adapterhelpers.PtrString("id"),
		},
		CidrRoutingConfig: &types.CidrRoutingConfig{
			CollectionId: adapterhelpers.PtrString("id"),
			LocationName: adapterhelpers.PtrString("somewhere"),
		},
		Failover: types.ResourceRecordSetFailoverPrimary,
		GeoLocation: &types.GeoLocation{
			ContinentCode:   adapterhelpers.PtrString("GB"),
			CountryCode:     adapterhelpers.PtrString("GB"),
			SubdivisionCode: adapterhelpers.PtrString("ENG"),
		},
		HealthCheckId:           adapterhelpers.PtrString("id"), // link
		MultiValueAnswer:        adapterhelpers.PtrBool(true),
		Region:                  types.ResourceRecordSetRegionApEast1,
		SetIdentifier:           adapterhelpers.PtrString("identifier"),
		TrafficPolicyInstanceId: adapterhelpers.PtrString("id"),
		Weight:                  adapterhelpers.PtrInt64(100),
	}

	item, err := resourceRecordSetItemMapper("", "foo", &recordSet)

	if err != nil {
		t.Error(err)
	}

	if err = item.Validate(); err != nil {
		t.Error(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "foo.bar.com",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "ns-1673.awsdns-17.co.uk.",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "ns-1505.awsdns-60.org.",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "ns-955.awsdns-55.net.",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "dns",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "ns-276.awsdns-34.com.",
			ExpectedScope:  "global",
		},
		{
			ExpectedType:   "route53-health-check",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "id",
			ExpectedScope:  "foo",
		},
	}

	tests.Execute(t, item)
}

func TestNewRoute53ResourceRecordSetAdapter(t *testing.T) {
	client, account, region := route53GetAutoConfig(t)

	zoneSource := NewRoute53HostedZoneAdapter(client, account, region)

	zones, err := zoneSource.List(context.Background(), zoneSource.Scopes()[0], true)
	if err != nil {
		t.Fatal(err)
	}

	if len(zones) == 0 {
		t.Skip("no zones found")
	}

	adapter := NewRoute53ResourceRecordSetAdapter(client, account, region)

	search := zones[0].UniqueAttributeValue()
	test := adapterhelpers.E2ETest{
		Adapter:         adapter,
		Timeout:         10 * time.Second,
		SkipGet:         true,
		GoodSearchQuery: &search,
	}

	test.Run(t)

	items, err := adapter.Search(context.Background(), zoneSource.Scopes()[0], search, true)
	if err != nil {
		t.Fatal(err)
	}

	numItems := len(items)

	rawZone := strings.TrimPrefix(search, "/hostedzone/")

	items, err = adapter.Search(context.Background(), zoneSource.Scopes()[0], rawZone, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != numItems {
		t.Errorf("expected %d items, got %d", numItems, len(items))
	}

	for _, item := range items {
		// Only use CNAME records
		typ, _ := item.GetAttributes().Get("Type")
		if typ != "CNAME" {
			continue
		}

		// Construct a terraform style ID
		fqdn, _ := item.GetAttributes().Get("Name")
		sections := strings.Split(fqdn.(string), ".")
		name := sections[0]
		search = fmt.Sprintf("%s_%s_%s", rawZone, name, typ)

		items, err := adapter.Search(context.Background(), zoneSource.Scopes()[0], search, true)
		if err != nil {
			t.Fatal(err)
		}

		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}

		// Only need to test this once
		break
	}
}
