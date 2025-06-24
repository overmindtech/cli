package manual_test

import (
	"context"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeSecurityPolicy(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeSecurityPolicyClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeSecurityPolicy(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeSecurityPolicy("test-security-policy"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-security-policy", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-security-policy|1000",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeSecurityPolicy(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeSecurityPolicyIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeSecurityPolicy("test-security-policy-1"), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeSecurityPolicy("test-security-policy-2"), nil)
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

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}
		}
	})
}

func createComputeSecurityPolicy(policyName string) *computepb.SecurityPolicy {
	return &computepb.SecurityPolicy{
		Name:   ptr.To(policyName),
		Labels: map[string]string{"env": "test"},
		Rules: []*computepb.SecurityPolicyRule{
			{
				Priority: ptr.To(int32(1000)),
			},
		},
		Region: ptr.To("us-central1"),
	}
}
