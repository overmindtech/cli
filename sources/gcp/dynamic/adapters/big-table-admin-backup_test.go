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

func TestBigTableAdminBackup(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	instanceName := "test-instance"
	clusterName := "test-cluster"
	linker := gcpshared.NewLinker()
	backupID := "test-backup"

	backup := &bigtableadmin.Backup{
		Name:         fmt.Sprintf("projects/%s/instances/%s/clusters/%s/backups/%s", projectID, instanceName, clusterName, backupID),
		SourceTable:  fmt.Sprintf("projects/%s/instances/%s/tables/source-table", projectID, instanceName),
		SourceBackup: fmt.Sprintf("projects/%s/instances/%s/clusters/%s/backups/source-backup", projectID, instanceName, clusterName),
		EncryptionInfo: &bigtableadmin.EncryptionInfo{
			KmsKeyVersion: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key/cryptoKeyVersions/1",
		},
	}

	backupList := &bigtableadmin.ListBackupsResponse{
		Backups: []*bigtableadmin.Backup{backup},
	}

	sdpItemType := gcpshared.BigTableAdminBackup

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups/%s", projectID, instanceName, clusterName, backupID): {
			StatusCode: http.StatusOK,
			Body:       backup,
		},
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups", projectID, instanceName, clusterName): {
			StatusCode: http.StatusOK,
			Body:       backupList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, clusterName, backupID)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get backup: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// name (BigTableAdminCluster)
					ExpectedType:   gcpshared.BigTableAdminCluster.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(instanceName, clusterName),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// sourceTable
					ExpectedType:   gcpshared.BigTableAdminTable.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(instanceName, "source-table"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// sourceBackup
					ExpectedType:   gcpshared.BigTableAdminBackup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(instanceName, clusterName, "source-backup"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// encryptionInfo.kmsKeyVersion
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "test-key", "1"),
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		searchQuery := shared.CompositeLookupKey(instanceName, clusterName)
		sdpItems, err := searchable.Search(ctx, projectID, searchQuery, true)
		if err != nil {
			t.Fatalf("Failed to search backups: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 backup, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups/%s", projectID, instanceName, clusterName, backupID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Backup not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, clusterName, backupID)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent backup, but got nil")
		}
	})
}
