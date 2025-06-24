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

func TestComputeNodeTemplate(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeNodeTemplateClient(ctrl)
	projectID := "test-project-id"
	region := "us-central1"

	t.Run("Get", func(t *testing.T) {
		// Attach mock client to our wrapper.
		wrapper := manual.NewComputeNodeTemplate(mockClient, projectID, region)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createNodeTemplateApiFixture("test-node-template"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-node-template", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// [SPEC] The default scope is a combined region and project id.
		if sdpItem.GetScope() != "test-project-id.us-central1" {
			t.Fatalf("Expected scope to be 'test-project-id.us-central1', got: %s", sdpItem.GetScope())
		}

		// [SPEC] Node templates are linked to one or more node groups.
		// TODO - this is not currently implemented in the adapter.
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

			if name.(string) != "test-node-template" {
				t.Fatalf("Expected name to be 'test-node-template', got: %s", name)
			}

			// [SPEC] Nested attributes are visible under attribute_parent.attribute_child
			serverBindingType, err := attributes.Get("server_binding.type")
			if err != nil {
				t.Fatalf("Error getting serverBindingType attribute: %v", err)
			}

			if serverBindingType.(string) != "RESTART_NODE_ON_ANY_SERVER" {
				t.Fatalf("Expected serverBindingType to be RESTART_NODE_ON_ANY_SERVER, got: %v", serverBindingType)
			}
		})

		t.Run("StaticTests", func(t *testing.T) {
			// [SPEC] A node template is linked to one or more node groups.
			// The query will be a SEARCH query against the node template URL.
			// The query uses all scopes as the scope of the node group is not the same as the template.
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNodeGroup.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "test-node-template",
					ExpectedScope:  "*",

					// [SPEC] The node groups does not affect the node template.
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
		wrapper := manual.NewComputeNodeTemplate(mockClient, projectID, region)
		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeNodeTemplateIter := mocks.NewMockComputeNodeTemplateIterator(ctrl)

		// Mock out items listed from the API.
		mockComputeNodeTemplateIter.EXPECT().Next().Return(createNodeTemplateApiFixture("test-node-template-1"), nil)
		mockComputeNodeTemplateIter.EXPECT().Next().Return(createNodeTemplateApiFixture("test-node-template-2"), nil)
		mockComputeNodeTemplateIter.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeNodeTemplateIter)

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

// Create an node template fixture (as returned from GCP API).
func createNodeTemplateApiFixture(nodeTemplateName string) *computepb.NodeTemplate {
	return &computepb.NodeTemplate{
		Name:     ptr.To(nodeTemplateName),
		NodeType: ptr.To("c2-node-60-240"),
		ServerBinding: &computepb.ServerBinding{
			Type: ptr.To("RESTART_NODE_ON_ANY_SERVER"),
		},
		SelfLink: ptr.To("test-self-link"),
		Region:   ptr.To("us-central1"),
	}
}
