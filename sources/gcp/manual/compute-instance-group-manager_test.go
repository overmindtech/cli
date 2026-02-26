package manual_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
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
		wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createInstanceGroupManager("test-instance-group-manager", true, instanceTemplateName), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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

				wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
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
					},
					{
						ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-group",
						ExpectedScope:  "test-project-id.us-central1-a",
					},
					{
						ExpectedType:   gcpshared.ComputeZone.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "us-central1-a",
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
				}

				shared.RunStaticTests(t, adapter, sdpItem, queryTests)
			})

			t.Run("RegionalInstanceTemplate", func(t *testing.T) {
				regionalInstanceTemplateName := "https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/instanceTemplates/unit-test-template"
				igm := createInstanceGroupManager("test-instance-group-manager", true, regionalInstanceTemplateName)

				wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
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
					},
					{
						ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-group",
						ExpectedScope:  "test-project-id.us-central1-a",
					},
					{
						ExpectedType:   gcpshared.ComputeZone.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "us-central1-a",
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
				}

				shared.RunStaticTests(t, adapter, sdpItem, queryTests)
			})

			t.Run("VersionsWithInstanceTemplates", func(t *testing.T) {
				// Create IGM with versions array containing multiple templates
				igm := &computepb.InstanceGroupManager{
					Name: new("test-instance-group-manager"),
					Status: &computepb.InstanceGroupManagerStatus{
						IsStable: new(true),
					},
					Versions: []*computepb.InstanceGroupManagerVersion{
						{
							Name:             new("canary"),
							InstanceTemplate: new("https://www.googleapis.com/compute/v1/projects/test-project-id/global/instanceTemplates/canary-template"),
						},
						{
							Name:             new("stable"),
							InstanceTemplate: new("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/instanceTemplates/stable-template"),
						},
					},
					InstanceGroup: new("projects/test-project-id/zones/us-central1-a/instanceGroups/test-group"),
					TargetPools: []string{
						"https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/targetPools/test-pool",
					},
					ResourcePolicies: &computepb.InstanceGroupManagerResourcePolicies{
						WorkloadPolicy: new("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/resourcePolicies/test-policy"),
					},
				}

				wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(igm, nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance-group-manager", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				queryTests := shared.QueryTests{
					// Canary version template (global)
					{
						ExpectedType:   gcpshared.ComputeInstanceTemplate.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "canary-template",
						ExpectedScope:  projectID,
					},
					// Stable version template (regional)
					{
						ExpectedType:   gcpshared.ComputeRegionInstanceTemplate.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "stable-template",
						ExpectedScope:  gcpshared.RegionalScope(projectID, region),
					},
					{
						ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-group",
						ExpectedScope:  "test-project-id.us-central1-a",
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
				}

				shared.RunStaticTests(t, adapter, sdpItem, queryTests)
			})

			t.Run("AutoHealingPoliciesWithHealthCheck", func(t *testing.T) {
				// Create IGM with auto-healing policy containing health check
				igm := &computepb.InstanceGroupManager{
					Name: new("test-instance-group-manager"),
					Status: &computepb.InstanceGroupManagerStatus{
						IsStable: new(true),
					},
					Zone:             new("https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a"),
					InstanceTemplate: new(instanceTemplateName),
					InstanceGroup:    new("projects/test-project-id/zones/us-central1-a/instanceGroups/test-group"),
					AutoHealingPolicies: []*computepb.InstanceGroupManagerAutoHealingPolicy{
						{
							HealthCheck:     new("https://www.googleapis.com/compute/v1/projects/test-project-id/global/healthChecks/test-health-check"),
							InitialDelaySec: new(int32(300)),
						},
					},
					TargetPools: []string{
						"https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/targetPools/test-pool",
					},
					ResourcePolicies: &computepb.InstanceGroupManagerResourcePolicies{
						WorkloadPolicy: new("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/resourcePolicies/test-policy"),
					},
				}

				wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
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
					},
					// Health check from auto-healing policy
					{
						ExpectedType:   gcpshared.ComputeHealthCheck.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-health-check",
						ExpectedScope:  projectID,
					},
					{
						ExpectedType:   gcpshared.ComputeInstanceGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-group",
						ExpectedScope:  "test-project-id.us-central1-a",
					},
					{
						ExpectedType:   gcpshared.ComputeZone.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "us-central1-a",
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
				wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockIterator := mocks.NewMockComputeInstanceGroupManagerIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createInstanceGroupManager("instance-group-manager-1", true, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(createInstanceGroupManager("instance-group-manager-2", false, instanceTemplateName), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

		// Check if adapter supports listing
		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
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

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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
		// Check if adapter supports list streaming
		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
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

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListCachesNotFoundWithMemoryCache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := mocks.NewMockComputeInstanceGroupManagerClient(ctrl)
		projectID := "cache-test-project"
		zone := "us-central1-a"
		scope := projectID + "." + zone

		mockAggIter := mocks.NewMockInstanceGroupManagersScopedListPairIterator(ctrl)
		mockAggIter.EXPECT().Next().Return(compute.InstanceGroupManagersScopedListPair{}, iterator.Done)
		mockListIter := mocks.NewMockComputeInstanceGroupManagerIterator(ctrl)
		mockListIter.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().AggregatedList(gomock.Any(), gomock.Any()).Return(mockAggIter).Times(1)
		mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(mockListIter).Times(1)

		wrapper := manual.NewComputeInstanceGroupManager(mockClient, []gcpshared.LocationInfo{gcpshared.NewZonalLocation(projectID, zone)})
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

func createInstanceGroupManager(name string, isStable bool, instanceTemplate string) *computepb.InstanceGroupManager {
	return &computepb.InstanceGroupManager{
		Name: new(name),
		Status: &computepb.InstanceGroupManagerStatus{
			IsStable: new(isStable),
		},
		Zone:             new("https://www.googleapis.com/compute/v1/projects/test-project-id/zones/us-central1-a"),
		InstanceTemplate: new(instanceTemplate),
		InstanceGroup:    new("projects/test-project-id/zones/us-central1-a/instanceGroups/test-group"),
		TargetPools: []string{
			"https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/targetPools/test-pool",
		},
		ResourcePolicies: &computepb.InstanceGroupManagerResourcePolicies{
			WorkloadPolicy: new("https://www.googleapis.com/compute/v1/projects/test-project-id/regions/us-central1/resourcePolicies/test-policy"),
		},
	}
}
