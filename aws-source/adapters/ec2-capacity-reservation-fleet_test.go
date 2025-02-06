package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func TestCapacityReservationFleetOutputMapper(t *testing.T) {
	output := &ec2.DescribeCapacityReservationFleetsOutput{
		CapacityReservationFleets: []types.CapacityReservationFleet{
			{
				AllocationStrategy:          adapterhelpers.PtrString("prioritized"),
				CapacityReservationFleetArn: adapterhelpers.PtrString("arn:aws:ec2:us-east-1:123456789012:capacity-reservation/fleet/crf-1234567890abcdef0"),
				CapacityReservationFleetId:  adapterhelpers.PtrString("crf-1234567890abcdef0"),
				CreateTime:                  adapterhelpers.PtrTime(time.Now()),
				EndDate:                     nil,
				InstanceMatchCriteria:       types.FleetInstanceMatchCriteriaOpen,
				InstanceTypeSpecifications: []types.FleetCapacityReservation{
					{
						AvailabilityZone:      adapterhelpers.PtrString("us-east-1a"), // link
						AvailabilityZoneId:    adapterhelpers.PtrString("use1-az1"),
						CapacityReservationId: adapterhelpers.PtrString("cr-1234567890abcdef0"), // link
						CreateDate:            adapterhelpers.PtrTime(time.Now()),
						EbsOptimized:          adapterhelpers.PtrBool(true),
						FulfilledCapacity:     adapterhelpers.PtrFloat64(1),
						InstancePlatform:      types.CapacityReservationInstancePlatformLinuxUnix,
						InstanceType:          types.InstanceTypeA12xlarge,
						Priority:              adapterhelpers.PtrInt32(1),
						TotalInstanceCount:    adapterhelpers.PtrInt32(1),
						Weight:                adapterhelpers.PtrFloat64(1),
					},
				},
				State:                  types.CapacityReservationFleetStateActive, // health
				Tenancy:                types.FleetCapacityReservationTenancyDefault,
				TotalFulfilledCapacity: adapterhelpers.PtrFloat64(1),
				TotalTargetCapacity:    adapterhelpers.PtrInt32(1),
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
	tests := adapterhelpers.QueryTests{}

	tests.Execute(t, item)

}

func TestNewEC2CapacityReservationFleetAdapter(t *testing.T) {
	client, account, region := ec2GetAutoConfig(t)

	adapter := NewEC2CapacityReservationFleetAdapter(client, account, region)

	test := adapterhelpers.E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
