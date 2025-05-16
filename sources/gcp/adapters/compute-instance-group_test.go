package adapters_test

import (
	"context"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/adapters"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
)

func TestComputeInstanceGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeInstanceGroupsClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Get", func(t *testing.T) {
		wrapper := adapters.NewComputeInstanceGroup(mockClient, projectID, zone)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeInstanceGroup("test-ig", "test-network", "test-subnetwork"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-ig", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		nameAttrValue, err := sdpItem.GetAttributes().Get("name")
		if err != nil || nameAttrValue != "test-ig" {
			t.Fatalf("Expected name 'test-ig', got: %s. Error: %v", nameAttrValue, err)
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := adapters.NewComputeInstanceGroup(mockClient, projectID, zone)
		adapter := sources.WrapperToAdapter(wrapper)

		mockIterator := mocks.NewMockComputeInstanceGroupIterator(ctrl)
		mockIterator.EXPECT().Next().Return(createComputeInstanceGroup("test-ig-1", "net-1", "subnet-1"), nil)
		mockIterator.EXPECT().Next().Return(createComputeInstanceGroup("test-ig-2", "net-2", "subnet-2"), nil)
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

func createComputeInstanceGroup(name, network, subnetwork string) *computepb.InstanceGroup {
	return &computepb.InstanceGroup{
		Name:       ptr.To(name),
		Network:    ptr.To("projects/test-project/global/networks/" + network),
		Subnetwork: ptr.To("projects/test-project/regions/us-central1/subnetworks/" + subnetwork),
	}
}
