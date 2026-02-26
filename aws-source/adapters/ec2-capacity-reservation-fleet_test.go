package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestCapacityReservationFleetOutputMapper(t *testing.T) {
	output := &ec2.DescribeCapacityReservationFleetsOutput{
		CapacityReservationFleets: []types.CapacityReservationFleet{
			{
				AllocationStrategy:          new("prioritized"),
				CapacityReservationFleetArn: new("arn:aws:ec2:us-east-1:123456789012:capacity-reservation/fleet/crf-1234567890abcdef0"),
				CapacityReservationFleetId:  new("crf-1234567890abcdef0"),
				CreateTime:                  new(time.Now()),
				EndDate:                     nil,
				InstanceMatchCriteria:       types.FleetInstanceMatchCriteriaOpen,
				InstanceTypeSpecifications: []types.FleetCapacityReservation{
					{
						AvailabilityZone:      new("us-east-1a"), // link
						AvailabilityZoneId:    new("use1-az1"),
						CapacityReservationId: new("cr-1234567890abcdef0"), // link
						CreateDate:            new(time.Now()),
						EbsOptimized:          new(true),
						FulfilledCapacity:     new(float64(1)),
						InstancePlatform:      types.CapacityReservationInstancePlatformLinuxUnix,
						InstanceType:          types.InstanceTypeA12xlarge,
						Priority:              new(int32(1)),
						TotalInstanceCount:    new(int32(1)),
						Weight:                new(float64(1)),
					},
				},
				State:                  types.CapacityReservationFleetStateActive, // health
				Tenancy:                types.FleetCapacityReservationTenancyDefault,
				TotalFulfilledCapacity: new(float64(1)),
				TotalTargetCapacity:    new(int32(1)),
			},
		},
	}

	items, err := capacityReservationFleetOutputMapper(context.Background(), nil, "foo", nil, output)

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
	tests := QueryTests{}

	tests.Execute(t, item)

}

func TestNewEC2CapacityReservationFleetAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2CapacityReservationFleetAdapter(client, account, region, sdpcache.NewNoOpCache())

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
