package manual_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeInstanceGroupManager(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeInstanceGroupManagerClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"
	region := "us-central1"
	instanceTemplateName := "https://www.googleapis.com/compute/v1/projects/test-project-id/global/instanceTemplates/unit-test-template"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeInstanceGroupManager(mockClient, projectID, zone)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createInstanceGroupManager("test-instance-group-manager", true, instanceTemplateName), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance-group-manager", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != gcpshared.ComputeInstanceGroupManager.String() {
			t.Fatalf("Expected type %s, got: %s", gcpshared.ComputeInstanceGroupManager.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			t.Run("GlobalInstanceTemplate", func(t *testing.T) {
				igm := createInstanceGroupManager("test-instance-group-manager", true, instanceTemplateName)

				wrapper := manual.NewComputeInstanceGroupManager(mockClient, projectID, zone)
				adapter := sources.WrapperToAdapter(wrapper)
				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(igm, nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance-group-manager", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				queryTests := shared.QueryTests{
					{
						ExpectedType:   gcpshared.ComputeInstanceTemplate.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "unit-test-template",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-group",
						ExpectedScope:  "test-project-id.us-central1-a",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeResourcePolicy.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-policy",
						ExpectedScope:  "test-project-id.us-central1",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeTargetPool.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-pool",
						ExpectedScope:  "test-project-id.us-central1",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
				}

				shared.RunStaticTests(t, adapter, sdpItem, queryTests)
			})

			t.Run("RegionalInstanceTemplate", func(t *testing.T) {
				regionalInstanceTemplateName := "https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/instanceTemplates/unit-test-template"
				igm := createInstanceGroupManager("test-instance-group-manager", true, regionalInstanceTemplateName)

				wrapper := manual.NewComputeInstanceGroupManager(mockClient, projectID, zone)
				adapter := sources.WrapperToAdapter(wrapper)
				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(igm, nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance-group-manager", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				queryTests := shared.QueryTests{
					{
						ExpectedType:   gcpshared.ComputeRegionInstanceTemplate.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "unit-test-template",
						ExpectedScope:  gcpshared.RegionalScope(projectID, region),
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-group",
						ExpectedScope:  "test-project-id.us-central1-a",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeResourcePolicy.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-policy",
						ExpectedScope:  "test-project-id.us-central1",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeTargetPool.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-pool",
						ExpectedScope:  "test-project-id.us-central1",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
				}

				shared.RunStaticTests(t, adapter, sdpItem, queryTests)
			})
		})

	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			isStable bool
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Healthy",
				isStable: true,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Unhealthy",
				isStable: false,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeInstanceGroupManager(mockClient, projectID, zone)
				adapter := sources.WrapperToAdapter(wrapper)

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createInstanceGroupManager("test-instance-group-manager", tc.isStable, instanceTemplateName), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance-group-manager", true)
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
		wrapper := manual.NewComputeInstanceGroupManager(mockClient, projectID, zone)
		adapter := sources.WrapperToAdapter(wrapper)

		mockIterator := mocks.NewMockComputeInstanceGroupManagerIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createInstanceGroupManager("instance-group-manager-1", true, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(createInstanceGroupManager("instance-group-manager-2", false, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

		sdpItems, err := adapter.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for i, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
			expectedName := "instance-group-manager-" + fmt.Sprintf("%d", i+1)
			if item.UniqueAttributeValue() != expectedName {
				t.Fatalf("Expected name %s, got: %s", expectedName, item.UniqueAttributeValue())
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeInstanceGroupManager(mockClient, projectID, zone)
		adapter := sources.WrapperToAdapter(wrapper)

		mockIterator := mocks.NewMockComputeInstanceGroupManagerIterator(ctrl)
		mockIterator.EXPECT().Next().Return(createInstanceGroupManager("instance-group-manager-1", true, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(createInstanceGroupManager("instance-group-manager-2", false, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		var errs []error
		mockItemHandler := func(item *sdp.Item) { items = append(items, item); wg.Done() }
		mockErrorHandler := func(err error) { errs = append(errs, err) }

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)
		adapter.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}
		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
		for _, item := range items {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})
}

func createInstanceGroupManager(name string, isStable bool, instanceTemplate string) *computepb.InstanceGroupManager {
	return &computepb.InstanceGroupManager{
		Name: ptr.To(name),
		Status: &computepb.InstanceGroupManagerStatus{
			IsStable: ptr.To(isStable),
		},
		InstanceTemplate: ptr.To(instanceTemplate),
		InstanceGroup:    ptr.To("projects/test-project-id/zones/us-central1-a/instanceGroups/test-group"),
		TargetPools: []string{
			"https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/targetPools/test-pool",
		},
		ResourcePolicies: &computepb.InstanceGroupManagerResourcePolicies{
			WorkloadPolicy: ptr.To("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/resourcePolicies/test-policy"),
		},
	}
}
