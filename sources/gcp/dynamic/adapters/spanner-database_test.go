package adapters

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/spanner/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestSpannerDatabase(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()

	databaseName := "test-database"
	instanceName := "test-instance"
	spannerDatabase := &spanner.Database{
		Name: fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceName, databaseName),
		EncryptionConfig: &spanner.EncryptionConfig{
			KmsKeyName: "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
			KmsKeyNames: []string{
				"projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/array-key-1",
			},
		},
		RestoreInfo: &spanner.RestoreInfo{
			BackupInfo: &spanner.BackupInfo{
				Backup: "projects/test-project/instances/test-instance/backups/my-backup",
			},
		},
		EncryptionInfo: []*spanner.EncryptionInfo{
			&spanner.EncryptionInfo{
				KmsKeyVersion: "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key/cryptoKeyVersions/1",
			},
		},
	}

	spannerDatabases := &spanner.ListDatabasesResponse{
		Databases: []*spanner.Database{spannerDatabase},
	}

	sdpItemType := gcpshared.SpannerDatabase

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://spanner.googleapis.com/v1/projects/%s/instances/%s/databases/%s", projectID, instanceName, databaseName): {
			StatusCode: http.StatusOK,
			Body:       spannerDatabase,
		},
		fmt.Sprintf("https://spanner.googleapis.com/v1/projects/%s/instances/%s/databases", projectID, instanceName): {
			StatusCode: http.StatusOK,
			Body:       spannerDatabases,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, databaseName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get Spanner database: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", databaseName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.SpannerBackup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-instance", "my-backup"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
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
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my-keyring", "array-key-1"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
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
					// name field creates a backlink to the Spanner instance
					ExpectedType:   gcpshared.SpannerInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  instanceName,
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
		// This is a project level adapter, so we pass the project
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, instanceName, true)
		if err != nil {
			t.Fatalf("Failed to list databases images: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 database, got %d", len(sdpItems))
		}
	})
}
