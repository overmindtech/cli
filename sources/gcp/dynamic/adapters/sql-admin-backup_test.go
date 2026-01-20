package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/sqladmin/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestSQLAdminBackup(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	backupName := "test-backup"

	backup := &sqladmin.Backup{
		Name:          backupName,
		Instance:      "test-instance",
		KmsKey:        "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
		KmsKeyVersion: "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key/cryptoKeyVersions/1",
		BackupRun:     "1234567890",
		InstanceSettings: &sqladmin.DatabaseInstance{
			Settings: &sqladmin.Settings{
				IpConfiguration: &sqladmin.IpConfiguration{
					PrivateNetwork: "projects/test-project/global/networks/test-network",
					AuthorizedNetworks: []*sqladmin.AclEntry{
						{
							Value: "203.0.113.0/24",
							Name:  "office-range",
						},
						{
							Value: "198.51.100.5/32",
							Name:  "admin-ip",
						},
					},
					AllocatedIpRange: "projects/test-project/locations/us-central1/internalRanges/test-range",
				},
			},
		},
	}

	backupList := &sqladmin.ListBackupsResponse{
		Backups: []*sqladmin.Backup{backup},
	}

	sdpItemType := gcpshared.SQLAdminBackup

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://sqladmin.googleapis.com/v1/projects/%s/backups/%s", projectID, backupName): {
			StatusCode: http.StatusOK,
			Body:       backup,
		},
		fmt.Sprintf("https://sqladmin.googleapis.com/v1/projects/%s/backups", projectID): {
			StatusCode: http.StatusOK,
			Body:       backupList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, backupName, true)
		if err != nil {
			t.Fatalf("Failed to get backup: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// instance
					ExpectedType:   gcpshared.SQLAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// kmsKey
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my-keyring", "my-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// kmsKeyVersion
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my-keyring", "my-key", "1"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// instanceSettings.settings.ipConfiguration.privateNetwork
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// instanceSettings.settings.ipConfiguration.authorizedNetworks.value (first entry)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.0/24",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// instanceSettings.settings.ipConfiguration.authorizedNetworks.value (second entry)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "198.51.100.5/32",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Note: allocatedIpRange link is not tested here because the NetworkConnectivityInternalRange adapter doesn't exist yet.
				// The blast propagation is defined in the adapter so it will work automatically when the adapter is created.
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list backups: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 backup, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://sqladmin.googleapis.com/v1/projects/%s/backups/%s", projectID, backupName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Backup not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, backupName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent backup, but got nil")
		}
	})
}
