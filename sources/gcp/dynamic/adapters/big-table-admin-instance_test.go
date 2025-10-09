package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/bigtable/admin/apiv2/adminpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestBigTableAdminInstance(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	instanceName := "test-instance"

	instance := &adminpb.Instance{
		Name: fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName),
	}

	instanceName2 := "test-instance-2"
	instance2 := &adminpb.Instance{
		Name: fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName2),
	}

	instanceList := &adminpb.ListInstancesResponse{
		Instances: []*adminpb.Instance{instance, instance2},
	}

	sdpItemType := gcpshared.BigTableAdminInstance

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s", projectID, instanceName): {
			StatusCode: http.StatusOK,
			Body:       instance,
		},
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s", projectID, instanceName2): {
			StatusCode: http.StatusOK,
			Body:       instance2,
		},
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances", projectID): {
			StatusCode: http.StatusOK,
			Body:       instanceList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, instanceName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != instanceName {
			t.Errorf("Expected unique attribute value '%s', got %s", instanceName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.BigTableAdminCluster.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  instanceName,
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
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s", projectID, instanceName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Instance not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, instanceName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
