package adapters_test

import (
	"context"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/adapters"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeReservation(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeReservationClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Get", func(t *testing.T) {
		wrapper := adapters.NewComputeReservation(mockClient, projectID, zone)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeReservation("test-reservation", computepb.Reservation_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-reservation", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   adapters.ComputeRegionCommitment.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-commitment",
					ExpectedScope:  "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   adapters.ComputeMachineType.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "n1-standard-1",
					ExpectedScope:  "test-project-id.us-central1-a",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   adapters.ComputeAcceleratorType.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "nvidia-tesla-k80",
					ExpectedScope:  "test-project-id.us-central1-a",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   adapters.ComputeResourcePolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-policy",
					ExpectedScope:  "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.Reservation_Status
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Ready",
				input:    computepb.Reservation_READY,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Creating",
				input:    computepb.Reservation_CREATING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Deleting",
				input:    computepb.Reservation_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Updating",
				input:    computepb.Reservation_UPDATING,
				expected: sdp.Health_HEALTH_PENDING,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := adapters.NewComputeReservation(mockClient, projectID, zone)
				adapter := sources.WrapperToAdapter(wrapper)

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeReservation("test-reservation", tc.input), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-reservation", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expected {
					t.Fatalf("Expected health %s, got: %s", tc.expected, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := adapters.NewComputeReservation(mockClient, projectID, zone)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeReservationIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeReservation("test-reservation-1", computepb.Reservation_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeReservation("test-reservation-2", computepb.Reservation_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

		sdpItems, err := adapter.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})
}

func createComputeReservation(reservationName string, status computepb.Reservation_Status) *computepb.Reservation {
	return &computepb.Reservation{
		Name: ptr.To(reservationName),
		Commitment: ptr.To(
			"https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/commitments/test-commitment",
		),
		SpecificReservation: &computepb.AllocationSpecificSKUReservation{
			InstanceProperties: &computepb.AllocationSpecificSKUAllocationReservedInstanceProperties{
				MachineType: ptr.To(
					"https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a/machineTypes/n1-standard-1",
				),
				GuestAccelerators: []*computepb.AcceleratorConfig{
					{
						AcceleratorType: ptr.To(
							"https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a/acceleratorTypes/nvidia-tesla-k80",
						),
					},
				},
			},
		},
		ResourcePolicies: map[string]string{
			"policy1": "https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/resourcePolicies/test-policy",
		},
		Status: ptr.To(status.String()),
	}
}
