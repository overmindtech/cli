package manual_test

import (
	"context"
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

func TestComputeBackendService(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeBackendServiceClient(ctrl)
	projectID := "test-project"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeBackendService("test-backend-service"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-edge-security-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkSecurityClientTlsPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-client-tls-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceLbPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-lb-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.NetworkServicesServiceBinding.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-binding",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})

	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockBackendServiceIterator := mocks.NewMockComputeBackendServiceIterator(ctrl)

		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-backend-service"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-backend-service"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockBackendServiceIterator)

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

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeBackendService(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockBackendServiceIterator := mocks.NewMockComputeBackendServiceIterator(ctrl)

		// add mock implementation here
		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-backend-service-1"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(createComputeBackendService("test-backend-service-2"), nil)
		mockBackendServiceIterator.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockBackendServiceIterator)

		wg := &sync.WaitGroup{}
		wg.Add(2) // we added two items

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done() // signal that we processed an item
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)
		adapter.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})
}

func createComputeBackendService(name string) *computepb.BackendService {
	return &computepb.BackendService{
		Name:               ptr.To(name),
		Network:            ptr.To("global/networks/network"),
		SecurityPolicy:     ptr.To("https://compute.googleapis.com/compute/v1/projects/test-project/global/securityPolicies/test-security-policy"),
		EdgeSecurityPolicy: ptr.To("https://compute.googleapis.com/compute/v1/projects/test-project/global/securityPolicies/test-edge-security-policy"),
		SecuritySettings: &computepb.SecuritySettings{
			ClientTlsPolicy: ptr.To("https://networksecurity.googleapis.com/v1/projects/test-project/locations/test-location/clientTlsPolicies/test-client-tls-policy"),
		},
		ServiceLbPolicy: ptr.To(" https://networkservices.googleapis.com/v1alpha1/name=projects/test-project/locations/test-location/serviceLbPolicies/test-service-lb-policy"),
		ServiceBindings: []string{
			"https://networkservices.googleapis.com/v1alpha1/projects/test-project/locations/test-location/serviceBindings/test-service-binding",
		},
	}
}
