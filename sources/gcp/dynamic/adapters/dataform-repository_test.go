package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	dataform "google.golang.org/api/dataform/v1beta1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestDataformRepository(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	repositoryName := "test-repo"

	repository := &dataform.Repository{
		Name:           fmt.Sprintf("projects/%s/locations/%s/repositories/%s", projectID, location, repositoryName),
		ServiceAccount: "dataform-sa@test-project.iam.gserviceaccount.com",
		KmsKeyName:     "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
	}

	repositoryList := &dataform.ListRepositoriesResponse{
		Repositories: []*dataform.Repository{repository},
	}

	sdpItemType := gcpshared.DataformRepository

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories/%s", projectID, location, repositoryName): {
			StatusCode: http.StatusOK,
			Body:       repository,
		},
		fmt.Sprintf("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       repositoryList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, repositoryName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get dataform repository: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// serviceAccount
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "dataform-sa@test-project.iam.gserviceaccount.com",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// kmsKeyName
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my-keyring", "my-key"),
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

		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search dataform repositories: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 dataform repository, got %d", len(sdpItems))
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

		// Test Terraform format: projects/[project]/locations/[location]/repositories/[repository]
		terraformQuery := fmt.Sprintf("projects/%s/locations/%s/repositories/%s", projectID, location, repositoryName)
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
			fmt.Sprintf("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories/%s", projectID, location, repositoryName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Repository not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, repositoryName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent dataform repository, but got nil")
		}
	})
}
