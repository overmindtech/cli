package manual_test

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeRegionInstanceGroupManager(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockRegionInstanceGroupManagerClient(ctrl)
	projectID := "test-project-id"
	region := "us-central1"
	instanceTemplateName := "https://www.googleapis.com/compute/v1/projects/test-project-id/global/instanceTemplates/unit-test-template"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeRegionInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createRegionInstanceGroupManager("test-region-instance-group-manager", true, instanceTemplateName), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-region-instance-group-manager", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != gcpshared.ComputeRegionInstanceGroupManager.String() {
			t.Fatalf("Expected type %s, got: %s", gcpshared.ComputeRegionInstanceGroupManager.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			t.Run("GlobalInstanceTemplate", func(t *testing.T) {
				igm := createRegionInstanceGroupManager("test-region-instance-group-manager", true, instanceTemplateName)

				wrapper := manual.NewComputeRegionInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(igm, nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-region-instance-group-manager", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				queryTests := shared.QueryTests{
					{
						ExpectedType:   gcpshared.ComputeInstanceTemplate.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "unit-test-template",
						ExpectedScope:  projectID,
					},
					{
						ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-group",
						ExpectedScope:  "test-project-id.us-central1",
					},
					{
						ExpectedType:   gcpshared.ComputeRegion.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "us-central1",
						ExpectedScope:  projectID,
					},
					{
						ExpectedType:   gcpshared.ComputeResourcePolicy.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-policy",
						ExpectedScope:  "test-project-id.us-central1",
					},
					{
						ExpectedType:   gcpshared.ComputeTargetPool.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-pool",
						ExpectedScope:  "test-project-id.us-central1",
					},
					{
						ExpectedType:   gcpshared.ComputeAutoscaler.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-autoscaler",
						ExpectedScope:  "test-project-id.us-central1",
					},
				}
				shared.RunStaticTests(t, adapter, sdpItem, queryTests)
			})

			t.Run("RegionalInstanceTemplate", func(t *testing.T) {
				regionalInstanceTemplateName := "https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/instanceTemplates/regional-template"
				igm := createRegionInstanceGroupManager("test-region-instance-group-manager", true, regionalInstanceTemplateName)

				wrapper := manual.NewComputeRegionInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(igm, nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-region-instance-group-manager", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				queryTests := shared.QueryTests{
					{
						ExpectedType:   gcpshared.ComputeRegionInstanceTemplate.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "regional-template",
						ExpectedScope:  "test-project-id.us-central1",
					},
					{
						ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-group",
						ExpectedScope:  "test-project-id.us-central1",
					},
					{
						ExpectedType:   gcpshared.ComputeRegion.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "us-central1",
						ExpectedScope:  projectID,
					},
					{
						ExpectedType:   gcpshared.ComputeTargetPool.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-pool",
						ExpectedScope:  "test-project-id.us-central1",
					},
					{
						ExpectedType:   gcpshared.ComputeResourcePolicy.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-policy",
						ExpectedScope:  "test-project-id.us-central1",
					},
					{
						ExpectedType:   gcpshared.ComputeAutoscaler.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-autoscaler",
						ExpectedScope:  "test-project-id.us-central1",
					},
				}
				shared.RunStaticTests(t, adapter, sdpItem, queryTests)
			})
		})
	})

	t.Run("HealthCheck", func(t *testing.T) {
		healthTests := []struct {
			name           string
			isStable       bool
			expectedHealth sdp.Health
		}{
			{
				name:           "Stable",
				isStable:       true,
				expectedHealth: sdp.Health_HEALTH_OK,
			},
			{
				name:           "Unstable",
				isStable:       false,
				expectedHealth: sdp.Health_HEALTH_UNKNOWN,
			},
		}

		for _, tc := range healthTests {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeRegionInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createRegionInstanceGroupManager("test-region-instance-group-manager", tc.isStable, instanceTemplateName), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-region-instance-group-manager", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expectedHealth {
					t.Fatalf("Expected health %v, got: %v", tc.expectedHealth, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeRegionInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		mockIterator := mocks.NewMockRegionInstanceGroupManagerIterator(ctrl)
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)
		mockIterator.EXPECT().Next().Return(createRegionInstanceGroupManager("region-instance-group-manager-1", true, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(createRegionInstanceGroupManager("region-instance-group-manager-2", false, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		items, qErr := wrapper.(sources.ListableWrapper).List(ctx, wrapper.Scopes()[0])
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}

		for i, item := range items {
			expectedName := "region-instance-group-manager-" + fmt.Sprintf("%d", i+1)
			if item.GetAttributes().GetAttrStruct().GetFields()["name"].GetStringValue() != expectedName {
				t.Fatalf("Expected name %s, got: %s", expectedName, item.GetAttributes().GetAttrStruct().GetFields()["name"].GetStringValue())
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeRegionInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})

		mockIterator := mocks.NewMockRegionInstanceGroupManagerIterator(ctrl)
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)
		mockIterator.EXPECT().Next().Return(createRegionInstanceGroupManager("region-instance-group-manager-1", true, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(createRegionInstanceGroupManager("region-instance-group-manager-2", false, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		stream := discovery.NewRecordingQueryResultStream()
		noOpCache := sdpcache.NewNoOpCache()
		emptyCacheKey := sdpcache.CacheKey{}

		wrapper.ListStream(ctx, stream, noOpCache, emptyCacheKey, wrapper.Scopes()[0])

		items := stream.GetItems()
		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("ListCachesNotFoundWithMemoryCache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := mocks.NewMockRegionInstanceGroupManagerClient(ctrl)
		projectID := "cache-test-project"
		region := "us-central1"
		scope := projectID + "." + region

		// "*" path calls List once per region; specific scope calls List once. With 1 region: 2 List calls total.
		mockIter1 := mocks.NewMockRegionInstanceGroupManagerIterator(ctrl)
		mockIter1.EXPECT().Next().Return(nil, iterator.Done)
		mockIter2 := mocks.NewMockRegionInstanceGroupManagerIterator(ctrl)
		mockIter2.EXPECT().Next().Return(nil, iterator.Done)
		mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(mockIter1).Times(1)
		mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(mockIter2).Times(1)

		wrapper := manual.NewComputeRegionInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewRegionalLocation(projectID, region)})
		cache := sdpcache.NewMemoryCache()
		adapter := sources.WrapperToAdapter(wrapper, cache)
		discAdapter := adapter.(discovery.Adapter)
		listable := adapter.(discovery.ListableAdapter)

		// --- Scope "*" ---
		items, err := listable.List(ctx, "*", false)
		if err != nil {
			t.Fatalf("first List(*): %v", err)
		}
		if len(items) != 0 {
			t.Errorf("first List(*): expected 0 items, got %d", len(items))
		}
		cacheHit, _, _, qErr, done := cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_LIST, "*", discAdapter.Type(), "", false)
		done()
		if !cacheHit {
			t.Fatal("expected cache hit for List(*)")
		}
		if qErr == nil || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("expected cached NOTFOUND for List(*), got %v", qErr)
		}
		items, err = listable.List(ctx, "*", false)
		if err != nil {
			t.Fatalf("second List(*): %v", err)
		}
		if len(items) != 0 {
			t.Errorf("second List(*): expected 0 items, got %d", len(items))
		}

		// --- Specific scope ---
		items, err = listable.List(ctx, scope, false)
		if err != nil {
			t.Fatalf("first List(scope): %v", err)
		}
		if len(items) != 0 {
			t.Errorf("first List(scope): expected 0 items, got %d", len(items))
		}
		cacheHit, _, _, qErr, done = cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_LIST, scope, discAdapter.Type(), "", false)
		done()
		if !cacheHit {
			t.Fatal("expected cache hit for List(scope)")
		}
		if qErr == nil || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("expected cached NOTFOUND for List(scope), got %v", qErr)
		}
		items, err = listable.List(ctx, scope, false)
		if err != nil {
			t.Fatalf("second List(scope): %v", err)
		}
		if len(items) != 0 {
			t.Errorf("second List(scope): expected 0 items, got %d", len(items))
		}
	})
}

func createRegionInstanceGroupManager(name string, isStable bool, instanceTemplate string) *computepb.InstanceGroupManager {
	return &computepb.InstanceGroupManager{
		Name: new(name),
		Status: &computepb.InstanceGroupManagerStatus{
			IsStable:   new(isStable),
			Autoscaler: new("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/autoscalers/test-autoscaler"),
		},
		Region:           new("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1"),
		InstanceTemplate: new(instanceTemplate),
		InstanceGroup:    new("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/instanceGroups/test-group"),
		TargetPools:      []string{"https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/targetPools/test-pool"},
		ResourcePolicies: &computepb.InstanceGroupManagerResourcePolicies{
			WorkloadPolicy: new("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/resourcePolicies/test-policy"),
		},
	}
}
