package adapters_test

import (
	"context"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/adapters"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"
)

func TestComputeAutoscalerWrapper(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeAutoscalerClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Get", func(t *testing.T) {
		// Attach mock client to our wrapper.
		wrapper := adapters.NewComputeAutoscalerWrapper(mockClient, projectID, zone)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createAutoscalerApiFixture("test-autoscaler"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-autoscaler", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// [SPEC] The default scope is a combined zone and project id.
		if sdpItem.GetScope() != "test-project-id.us-central1-a" {
			t.Fatalf("Expected scope to be 'test-project-id.us-central1-a', got: %s", sdpItem.GetScope())
		}

		// [SPEC] Autoscalers have one link: the targeted Instance Group Manager.
		if len(sdpItem.GetLinkedItemQueries()) != 1 {
			t.Fatalf("Expected 1 linked item query, got: %d", len(sdpItem.GetLinkedItemQueries()))
		}

		t.Run("Attributes", func(t *testing.T) {
			// Check for a few attributes from the fixture to make sure they were copied properly.
			// These will not really fail ever unless the underlying shared sources change; so it's more of a sanity check.
			attributes := sdpItem.GetAttributes()

			name, err := attributes.Get("name")
			if err != nil {
				t.Fatalf("Error getting name attribute: %v", err)
			}

			if name.(string) != "test-autoscaler" {
				t.Fatalf("Expected name to be 'test-autoscaler', got: %s", name)
			}

			// Nested attributes.
			minReplicas, err := attributes.Get("autoscaling_policy.min_num_replicas")
			if err != nil {
				t.Fatalf("Error getting MinNumReplicas attribute: %v", err)
			}

			if minReplicas.(float64) != 1 {
				t.Fatalf("Expected minNumReplicas to be 1, got: %d", minReplicas)
			}
		})

		t.Run("StaticTests", func(t *testing.T) {
			// [SPEC] An autoscaler is linked to a instance group manager. The query will
			// match the name of the IGM resource, and the scope is the same zone.
			queryTests := shared.QueryTests{
				{
					ExpectedType:   adapters.ComputeInstanceGroupManager.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance-group",
					ExpectedScope:  "test-project-id.us-central1-a",

					// [SPEC] Autoscalers are tightly coupled with the instance group manager
					// (albeit less strength on the IN direction).
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})

	})

	t.Run("List", func(t *testing.T) {
		wrapper := adapters.NewComputeAutoscalerWrapper(mockClient, projectID, zone)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeAutoscalerIter := mocks.NewMockComputeAutoscalerIterator(ctrl)

		// Mock out items listed from the API.
		mockComputeAutoscalerIter.EXPECT().Next().Return(createAutoscalerApiFixture("test-autoscaler-1"), nil)
		mockComputeAutoscalerIter.EXPECT().Next().Return(createAutoscalerApiFixture("test-autoscaler-2"), nil)
		mockComputeAutoscalerIter.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeAutoscalerIter)

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

// Create an autoscaler fixture (as returned from GCP API).
func createAutoscalerApiFixture(autoscalerName string) *computepb.Autoscaler {
	return &computepb.Autoscaler{
		Name:   ptr.To(autoscalerName),
		Target: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a/instanceGroupManagers/test-instance-group"),
		AutoscalingPolicy: &computepb.AutoscalingPolicy{
			MinNumReplicas: ptr.To(int32(1)),
			MaxNumReplicas: ptr.To(int32(5)),
			CpuUtilization: &computepb.AutoscalingPolicyCpuUtilization{
				UtilizationTarget: ptr.To(float64(0.6)),
			},
		},
		Zone: ptr.To("us-central1-a"),
	}
}
