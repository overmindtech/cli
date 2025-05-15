package adapters_test

import (
	"context"
	"fmt"
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
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeForwardingRule(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeForwardingRuleClient(ctrl)
	projectID := "test-project-id"
	region := "us-central1"

	t.Run("Get", func(t *testing.T) {
		wrapper := adapters.NewComputeForwardingRule(mockClient, projectID, region)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createForwardingRule("test-rule", projectID, region, "192.168.1.1"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-rule", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   adapters.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnetwork",
					ExpectedScope:  "test-project-id.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   adapters.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   adapters.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backend-service",
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

	t.Run("List", func(t *testing.T) {
		wrapper := adapters.NewComputeForwardingRule(mockClient, projectID, region)

		adapter := sources.WrapperToAdapter(wrapper)

		mockIterator := mocks.NewMockForwardingRuleIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createForwardingRule("test-rule-1", projectID, region, "192.168.1.1"), nil)
		mockIterator.EXPECT().Next().Return(createForwardingRule("test-rule-2", projectID, region, "192.168.1.2"), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

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

func createForwardingRule(name, projectID, region, ipAddress string) *computepb.ForwardingRule {
	return &computepb.ForwardingRule{
		Name:           ptr.To(name),
		IPAddress:      ptr.To(ipAddress),
		Labels:         map[string]string{"env": "test"},
		Network:        ptr.To("global/networks/test-network"),
		Subnetwork:     ptr.To(fmt.Sprintf("regions/%s/subnetworks/test-subnetwork", region)),
		BackendService: ptr.To(fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/backendServices/backend-service", projectID, region)),
	}
}
