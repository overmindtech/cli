package manual

import (
	"context"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestExtractPolicyNameAndLocation(t *testing.T) {
	tests := []struct {
		name               string
		url                string
		expectedPolicyName string
		expectedLocation   string
	}{
		{
			name: "Valid URL with policy and location",
			// GET https://networkservices.googleapis.com/v1/{name=projects/*/locations/*/serviceLbPolicies/*}
			url:                "https://networksecurity.googleapis.com/v1/projects/my-project/locations/test-location/clientTlsPolicies/my-policy",
			expectedPolicyName: "my-policy",
			expectedLocation:   "test-location",
		},
		{
			name:               "Invalid URL with missing parts",
			url:                "https://networksecurity.googleapis.com/v1/projects/my-project/locations/",
			expectedPolicyName: "",
			expectedLocation:   "",
		},
		{
			name:               "Empty URL",
			url:                "",
			expectedPolicyName: "",
			expectedLocation:   "",
		},
		{
			name:               "URL with extra slashes",
			url:                "https://networksecurity.googleapis.com/v1/projects/my-project/locations/test-location/clientTlsPolicies/",
			expectedPolicyName: "",
			expectedLocation:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policyName, location := extractNameAndLocation(tt.url)
			if policyName != tt.expectedPolicyName || location != tt.expectedLocation {
				t.Errorf("extractNameAndLocation(%q) = (%q, %q); want (%q, %q)",
					tt.url, policyName, location, tt.expectedPolicyName, tt.expectedLocation)
			}
		})
	}
}

func TestComputeBackendService(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeBackendServiceClient(ctrl)
	projectID := "test-project"

	t.Run("Get", func(t *testing.T) {
		wrapper := NewComputeBackendService(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeBackendService("test-backend-service"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-backend-service", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "network",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   ComputeSecurityPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-edge-security-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   NetworkSecurityClientTlsPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-client-tls-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   NetworkServicesServiceLbPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-location|test-service-lb-policy",
					ExpectedScope:  "test-project",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   NetworkServicesServiceBinding.String(),
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
		wrapper := NewComputeBackendService(mockClient, projectID)

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
			"https://networkservices.googleapis.com/v1alpha1/projects/test-project/locations/test-location/serviceLbPolicies/test-service-binding",
		},
	}
}
