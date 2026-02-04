package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

func TestCapacityReservationOutputMapper(t *testing.T) {
	output := &ec2.DescribeCapacityReservationsOutput{
		CapacityReservations: []types.CapacityReservation{
			{
				AvailabilityZone:           PtrString("us-east-1a"), // links
				AvailabilityZoneId:         PtrString("use1-az1"),
				AvailableInstanceCount:     PtrInt32(1),
				CapacityReservationArn:     PtrString("arn:aws:ec2:us-east-1:123456789012:capacity-reservation/cr-1234567890abcdef0"),
				CapacityReservationId:      PtrString("cr-1234567890abcdef0"),
				CapacityReservationFleetId: PtrString("crf-1234567890abcdef0"), // link
				CreateDate:                 PtrTime(time.Now()),
				EbsOptimized:               PtrBool(true),
				EndDateType:                types.EndDateTypeUnlimited,
				EndDate:                    nil,
				InstanceMatchCriteria:      types.InstanceMatchCriteriaTargeted,
				InstancePlatform:           types.CapacityReservationInstancePlatformLinuxUnix,
				InstanceType:               PtrString("t2.micro"),
				OutpostArn:                 PtrString("arn:aws:ec2:us-east-1:123456789012:outpost/op-1234567890abcdef0"), // link
				OwnerId:                    PtrString("123456789012"),
				PlacementGroupArn:          PtrString("arn:aws:ec2:us-east-1:123456789012:placement-group/pg-1234567890abcdef0"), // link
				StartDate:                  PtrTime(time.Now()),
				State:                      types.CapacityReservationStateActive,
				Tenancy:                    types.CapacityReservationTenancyDefault,
				TotalInstanceCount:         PtrInt32(1),
				CapacityAllocations: []types.CapacityAllocation{
					{
						AllocationType: types.AllocationTypeUsed,
						Count:          PtrInt32(1),
					},
				},
			},
		},
	}

	items, err := capacityReservationOutputMapper(context.Background(), nil, "foo", nil, output)

	if err != nil {
		t.Fatal(err)
	}

	for _, item := range items {
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
	tests := QueryTests{
		{
			ExpectedType:   "ec2-capacity-reservation-fleet",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "crf-1234567890abcdef0",
			ExpectedScope:  "foo",
		},
		{
			ExpectedType:   "outposts-outpost",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:ec2:us-east-1:123456789012:outpost/op-1234567890abcdef0",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "ec2-placement-group",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:ec2:us-east-1:123456789012:placement-group/pg-1234567890abcdef0",
			ExpectedScope:  "123456789012.us-east-1",
		},
	}

	tests.Execute(t, item)

}

func TestNewEC2CapacityReservationAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2CapacityReservationAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
