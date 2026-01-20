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

func TestBigTableAdminTable(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	instanceName := "test-instance"
	linker := gcpshared.NewLinker()
	tableName := "test-table"

	table := &bigtableadmin.Table{
		Name: fmt.Sprintf("projects/%s/instances/%s/tables/%s", projectID, instanceName, tableName),
		RestoreInfo: &bigtableadmin.RestoreInfo{
			BackupInfo: &bigtableadmin.BackupInfo{
				SourceTable:  fmt.Sprintf("projects/%s/instances/%s/tables/source-table", projectID, instanceName),
				SourceBackup: fmt.Sprintf("projects/%s/instances/%s/clusters/test-cluster/backups/test-backup", projectID, instanceName),
			},
		},
	}

	tableList := &bigtableadmin.ListTablesResponse{
		Tables: []*bigtableadmin.Table{table},
	}

	sdpItemType := gcpshared.BigTableAdminTable

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables/%s", projectID, instanceName, tableName): {
			StatusCode: http.StatusOK,
			Body:       table,
		},
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables", projectID, instanceName): {
			StatusCode: http.StatusOK,
			Body:       tableList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, tableName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get table: %v", err)
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
				// restoreInfo.backupInfo.sourceTable
				{
					ExpectedType:   gcpshared.BigTableAdminTable.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(instanceName, "source-table"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// restoreInfo.backupInfo.sourceBackup
				{
					ExpectedType:   gcpshared.BigTableAdminBackup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(instanceName, "test-cluster", "test-backup"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, instanceName, true)
		if err != nil {
			t.Fatalf("Failed to search tables: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 table, got %d", len(sdpItems))
		}
	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project]/instances/[instance]/tables/[table]
		terraformQuery := fmt.Sprintf("projects/%s/instances/%s/tables/%s", projectID, instanceName, tableName)
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
			fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables/%s", projectID, instanceName, tableName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Table not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, tableName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent table, but got nil")
		}
	})
}
