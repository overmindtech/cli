package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestSecretManagerSecret(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	secretID := "test-secret"

	// Create mock protobuf object with automatic replication
	secret := &secretmanagerpb.Secret{
		Name: fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID),
		Replication: &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{
					CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{
						KmsKeyName: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key",
					},
				},
			},
		},
		Topics: []*secretmanagerpb.Topic{
			{
				Name: fmt.Sprintf("projects/%s/topics/secret-events", projectID),
			},
		},
	}

	// Create second secret with user-managed replication
	secretID2 := "test-secret-2"
	secret2 := &secretmanagerpb.Secret{
		Name: fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID2),
		Replication: &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_UserManaged_{
				UserManaged: &secretmanagerpb.Replication_UserManaged{
					Replicas: []*secretmanagerpb.Replication_UserManaged_Replica{
						{
							Location: "us-central1",
							CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{
								KmsKeyName: "projects/test-project/locations/us-central1/keyRings/region-ring/cryptoKeys/region-key",
							},
						},
					},
				},
			},
		},
	}

	// Create third secret for list testing (minimal)
	secretID3 := "test-secret-3"
	secret3 := &secretmanagerpb.Secret{
		Name: fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID3),
	}

	// Create list response with multiple items
	secretList := &secretmanagerpb.ListSecretsResponse{
		Secrets: []*secretmanagerpb.Secret{secret, secret2, secret3},
	}

	sdpItemType := gcpshared.SecretManagerSecret

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://secretmanager.googleapis.com/v1/projects/%s/secrets/%s", projectID, secretID): {
			StatusCode: http.StatusOK,
			Body:       secret,
		},
		fmt.Sprintf("https://secretmanager.googleapis.com/v1/projects/%s/secrets/%s", projectID, secretID2): {
			StatusCode: http.StatusOK,
			Body:       secret2,
		},
		fmt.Sprintf("https://secretmanager.googleapis.com/v1/projects/%s/secrets/%s", projectID, secretID3): {
			StatusCode: http.StatusOK,
			Body:       secret3,
		},
		fmt.Sprintf("https://secretmanager.googleapis.com/v1/projects/%s/secrets", projectID): {
			StatusCode: http.StatusOK,
			Body:       secretList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, secretID, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != secretID {
			t.Errorf("Expected unique attribute value '%s', got %s", secretID, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// replication.automatic.customerManagedEncryption.kmsKeyName
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "test-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// topics.name
				{
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "secret-events",
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

	t.Run("Get with UserManaged Replication", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, secretID2, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != secretID2 {
			t.Errorf("Expected unique attribute value '%s', got %s", secretID2, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// replication.userManaged.replicas.customerManagedEncryption.kmsKeyName
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us-central1", "region-ring", "region-key"),
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

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
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

		if len(sdpItems) != 3 {
			t.Errorf("Expected 3 resources, got %d", len(sdpItems))
		}

		// Validate first item
		if len(sdpItems) > 0 {
			firstItem := sdpItems[0]
			if firstItem.GetType() != sdpItemType.String() {
				t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
			}
			if firstItem.GetScope() != projectID {
				t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
			}
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://secretmanager.googleapis.com/v1/projects/%s/secrets/%s", projectID, secretID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Secret not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, secretID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
