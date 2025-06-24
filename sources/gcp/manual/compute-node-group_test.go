package manual_test

import (
	"context"
	"strings"
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

func TestComputeNodeGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeNodeGroupClient(ctrl)
	projectID := "test-project-id"
	zone := "us-central1-a"

	testTemplateUrl := "https://www.googleapis.com/compute/v1/projects/test-project/regions/northamerica-northeast1/nodeTemplates/node-template-1"
	testTemplateUrl2 := "https://www.googleapis.com/compute/v1/projects/test-project/regions/northamerica-northeast1/nodeTemplates/node-template-2"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeNodeGroup(mockClient, projectID, zone)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeNodeGroup("test-node-group", testTemplateUrl, computepb.NodeGroup_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-node-group", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNodeTemplate.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "node-template-1",
					ExpectedScope:  "test-project-id.northamerica-northeast1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.NodeGroup_Status
			expected sdp.Health
		}

		testCases := []testCase{
			{
				name:     "Ready status",
				input:    computepb.NodeGroup_READY,
				expected: sdp.Health_HEALTH_OK,
			},
			{
				name:     "Invalid status",
				input:    computepb.NodeGroup_INVALID,
				expected: sdp.Health_HEALTH_ERROR,
			},
			{
				name:     "Creating status",
				input:    computepb.NodeGroup_CREATING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Deleting status",
				input:    computepb.NodeGroup_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeNodeGroup(mockClient, projectID, zone)
				adapter := sources.WrapperToAdapter(wrapper)

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeNodeGroup("test-ng", "test-temp", tc.input), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-node-group", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expected {
					t.Errorf("Expected health: %v, got: %v", tc.expected, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeNodeGroup(mockClient, projectID, zone)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeNodeGroupIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeNodeGroup("test-node-group-1", testTemplateUrl, computepb.NodeGroup_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeNodeGroup("test-node-group-2", testTemplateUrl2, computepb.NodeGroup_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
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

			query := item.GetLinkedItemQueries()[0].GetQuery().GetQuery()
			if !strings.Contains(query, "node-template") {
				t.Fatalf("Expected node-template in query, got: %s", query)
			}
		}
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewComputeNodeGroup(mockClient, projectID, zone)
		adapter := sources.WrapperToAdapter(wrapper)

		filterBy := testTemplateUrl

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, req *computepb.ListNodeGroupsRequest, opts ...any) *mocks.MockComputeNodeGroupIterator {
			fullList := []*computepb.NodeGroup{
				createComputeNodeGroup("test-node-group-1", testTemplateUrl, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-2", testTemplateUrl2, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-3", testTemplateUrl, computepb.NodeGroup_READY),
				createComputeNodeGroup("test-node-group-4", testTemplateUrl, computepb.NodeGroup_READY),
			}

			expectedFilter := "nodeTemplate = " + filterBy
			if req.GetFilter() != expectedFilter {
				t.Fatalf("Expected filter to be %s, got: %s", expectedFilter, req.GetFilter())
			}

			mockComputeIterator := mocks.NewMockComputeNodeGroupIterator(ctrl)
			for _, nodeGroup := range fullList {
				if nodeGroup.GetNodeTemplate() == filterBy {
					mockComputeIterator.EXPECT().Next().Return(nodeGroup, nil)
				}
			}

			mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

			return mockComputeIterator
		})

		// [SPEC] Search filters by the node template URL. It will list and filter out
		// any node groups that are not using the given URL.

		sdpItems, err := adapter.Search(ctx, wrapper.Scopes()[0], testTemplateUrl, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 1 of 4 are filtered out.
		if len(sdpItems) != 3 {
			t.Fatalf("Expected 3 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}

			attributes := item.GetAttributes()
			nodeTemplate, err := attributes.Get("node_template")
			if err != nil {
				t.Fatalf("Failed to get node_template attribute: %v", err)
			}

			if nodeTemplate != testTemplateUrl {
				t.Fatalf("Expected node_template to be %s, got: %s", testTemplateUrl, nodeTemplate)
			}
		}
	})
}

func createComputeNodeGroup(name, templateUrl string, status computepb.NodeGroup_Status) *computepb.NodeGroup {
	return &computepb.NodeGroup{
		Name:         ptr.To(name),
		NodeTemplate: ptr.To(templateUrl),
		Status:       ptr.To(status.String()),
	}
}
