package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/bigtableadmin/v2"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestBigTableAdminAppProfile(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	instanceName := "test-instance"
	linker := gcpshared.NewLinker()
	appProfileID := "test-app-profile"

	appProfile := &bigtableadmin.AppProfile{
		Name: fmt.Sprintf("projects/%s/instances/%s/appProfiles/%s", projectID, instanceName, appProfileID),
		SingleClusterRouting: &bigtableadmin.SingleClusterRouting{
			ClusterId: "test-cluster",
		},
	}

	// Second app profile with multi-cluster routing
	appProfileID2 := "test-app-profile-multi"
	appProfile2 := &bigtableadmin.AppProfile{
		Name: fmt.Sprintf("projects/%s/instances/%s/appProfiles/%s", projectID, instanceName, appProfileID2),
		MultiClusterRoutingUseAny: &bigtableadmin.MultiClusterRoutingUseAny{
			ClusterIds: []string{"cluster-1", "cluster-2"},
		},
	}

	appProfileList := &bigtableadmin.ListAppProfilesResponse{
		AppProfiles: []*bigtableadmin.AppProfile{appProfile},
	}

	sdpItemType := gcpshared.BigTableAdminAppProfile

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles/%s", projectID, instanceName, appProfileID): {
			StatusCode: http.StatusOK,
			Body:       appProfile,
		},
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles/%s", projectID, instanceName, appProfileID2): {
			StatusCode: http.StatusOK,
			Body:       appProfile2,
		},
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles", projectID, instanceName): {
			StatusCode: http.StatusOK,
			Body:       appProfileList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, appProfileID)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get app profile: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// name (parent instance)
				{
					ExpectedType:   gcpshared.BigTableAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  instanceName,
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// TODO: Add test for singleClusterRouting.clusterId → BigTableAdminCluster
				// Requires manual linker to combine instance name with cluster ID
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get with multi-cluster routing", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, appProfileID2)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get app profile: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// name (parent instance)
				{
					ExpectedType:   gcpshared.BigTableAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  instanceName,
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// TODO: Add tests for multiClusterRoutingUseAny.clusterIds → BigTableAdminCluster
				// Requires manual linker to combine instance name with cluster IDs
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, instanceName, true)
		if err != nil {
			t.Fatalf("Failed to search app profiles: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 app profile, got %d", len(sdpItems))
		}
	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project]/instances/[instance]/appProfiles/[app_profile]
		terraformQuery := fmt.Sprintf("projects/%s/instances/%s/appProfiles/%s", projectID, instanceName, appProfileID)
		sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
		if err != nil {
			t.Fatalf("Failed to search resources with Terraform format: %v", err)
		}

		// The search should return only the specific resource matching the Terraform format
		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(sdpItems))
			return
		}

		// Verify the single item returned
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles/%s", projectID, instanceName, appProfileID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "App profile not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, appProfileID)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent app profile, but got nil")
		}
	})
}
