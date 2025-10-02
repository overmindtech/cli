package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/bigquery/datatransfer/apiv1/datatransferpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// Helper functions for creating pointers
func stringValuePtr(s string) *wrapperspb.StringValue {
	return &wrapperspb.StringValue{Value: s}
}

func TestBigQueryDataTransferTransferConfig(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	location := "us-central1"
	transferConfigName := "test-transfer-config"
	transferConfigName2 := "test-transfer-config-2"
	destinationDatasetId := "test-dataset"
	dataSourceId := "test-data-source"
	notificationPubsubTopic := "projects/test-project/topics/test-topic"
	kmsKeyName := "projects/test-project/locations/us-central1/keyRings/test-ring/cryptoKeys/test-key"

	// Create mock protobuf objects
	transferConfig := &datatransferpb.TransferConfig{
		Name:         fmt.Sprintf("projects/%s/locations/%s/transferConfigs/%s", projectID, location, transferConfigName),
		DisplayName:  "Test Transfer Config",
		DataSourceId: dataSourceId,
		Destination: &datatransferpb.TransferConfig_DestinationDatasetId{
			DestinationDatasetId: destinationDatasetId,
		},
		Schedule:                "0 9 * * *", // Daily at 9 AM
		NotificationPubsubTopic: notificationPubsubTopic,
		EncryptionConfiguration: &datatransferpb.EncryptionConfiguration{
			KmsKeyName: stringValuePtr(kmsKeyName),
		},
	}

	transferConfig2 := &datatransferpb.TransferConfig{
		Name:         fmt.Sprintf("projects/%s/locations/%s/transferConfigs/%s", projectID, location, transferConfigName2),
		DisplayName:  "Test Transfer Config 2",
		DataSourceId: dataSourceId,
		Destination: &datatransferpb.TransferConfig_DestinationDatasetId{
			DestinationDatasetId: destinationDatasetId,
		},
		Schedule:                "0 12 * * *", // Daily at 12 PM
		NotificationPubsubTopic: notificationPubsubTopic,
	}

	// Create list response with multiple items
	transferConfigList := &datatransferpb.ListTransferConfigsResponse{
		TransferConfigs: []*datatransferpb.TransferConfig{transferConfig, transferConfig2},
	}

	sdpItemType := gcpshared.BigQueryDataTransferTransferConfig

	// Mock HTTP responses for location-based resources
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://bigquerydatatransfer.googleapis.com/v1/projects/%s/locations/%s/transferConfigs/%s", projectID, location, transferConfigName): {
			StatusCode: http.StatusOK,
			Body:       transferConfig,
		},
		fmt.Sprintf("https://bigquerydatatransfer.googleapis.com/v1/projects/%s/locations/%s/transferConfigs/%s", projectID, location, transferConfigName2): {
			StatusCode: http.StatusOK,
			Body:       transferConfig2,
		},
		fmt.Sprintf("https://bigquerydatatransfer.googleapis.com/v1/projects/%s/locations/%s/transferConfigs", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       transferConfigList,
		},
	}

	// Test Get with location + resource name
	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		// For multiple query parameters, use the combined query format
		combinedQuery := fmt.Sprintf("%s|%s", location, transferConfigName)
		sdpItem, err := adapter.Get(ctx, projectID, combinedQuery, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != combinedQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", combinedQuery, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific attributes
		name, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		if name != transferConfig.GetName() {
			t.Errorf("Expected name field to be '%s', got %s", transferConfig.GetName(), name)
		}

		displayName, err := sdpItem.GetAttributes().Get("displayName")
		if err != nil {
			t.Fatalf("Failed to get 'displayName' attribute: %v", err)
		}
		if displayName != transferConfig.GetDisplayName() {
			t.Errorf("Expected displayName field to be '%s', got %s", transferConfig.GetDisplayName(), displayName)
		}

		dataSourceIdAttr, err := sdpItem.GetAttributes().Get("dataSourceId")
		if err != nil {
			t.Fatalf("Failed to get 'dataSourceId' attribute: %v", err)
		}
		if dataSourceIdAttr != dataSourceId {
			t.Errorf("Expected dataSourceId field to be '%s', got %s", dataSourceId, dataSourceIdAttr)
		}

		destinationDatasetIdAttr, err := sdpItem.GetAttributes().Get("destinationDatasetId")
		if err != nil {
			t.Fatalf("Failed to get 'destinationDatasetId' attribute: %v", err)
		}
		if destinationDatasetIdAttr != destinationDatasetId {
			t.Errorf("Expected destinationDatasetId field to be '%s', got %s", destinationDatasetId, destinationDatasetIdAttr)
		}

		notificationTopic, err := sdpItem.GetAttributes().Get("notificationPubsubTopic")
		if err != nil {
			t.Fatalf("Failed to get 'notificationPubsubTopic' attribute: %v", err)
		}
		if notificationTopic != notificationPubsubTopic {
			t.Errorf("Expected notificationPubsubTopic field to be '%s', got %s", notificationPubsubTopic, notificationTopic)
		}

		// Include static tests - MUST cover ALL blast propagation links
		t.Run("StaticTests", func(t *testing.T) {
			// CRITICAL: Review the adapter's blast propagation configuration and create
			// test cases for EVERY linked resource defined in the adapter's blastPropagation map
			queryTests := shared.QueryTests{
				// destinationDatasetId link
				{
					ExpectedType:   gcpshared.BigQueryDataset.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  destinationDatasetId,
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// dataSourceId link - NOTE: BigQueryDataTransferDataSource adapter doesn't exist yet
				// TODO: Add test case when BigQueryDataTransferDataSource adapter is created
				// notificationPubsubTopic link
				{
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-topic",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// encryptionConfiguration.kmsKeyName link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us-central1", "test-ring", "test-key"),
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

	// Test Search (location-based resources typically use Search instead of List)
	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test location-based search
		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}

		// Verify first item
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
		}

		// Verify second item
		secondItem := sdpItems[1]
		if secondItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected second item type %s, got %s", sdpItemType.String(), secondItem.GetType())
		}
		if secondItem.GetScope() != projectID {
			t.Errorf("Expected second item scope '%s', got %s", projectID, secondItem.GetScope())
		}
	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project_id]/locations/[location]/transferConfigs/[transfer_config_id]
		// The adapter should extract the location from this format and search in that location
		terraformQuery := fmt.Sprintf("projects/%s/locations/%s/transferConfigs/%s", projectID, location, transferConfigName)
		sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
		if err != nil {
			t.Fatalf("Failed to search resources with Terraform format: %v", err)
		}

		// The search should return all resources in the location extracted from the Terraform format
		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(sdpItems))
			return
		}

		// Verify first item
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://bigquerydatatransfer.googleapis.com/v1/projects/%s/locations/%s/transferConfigs/%s", projectID, location, transferConfigName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Resource not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := fmt.Sprintf("%s|%s", location, transferConfigName)
		_, err = adapter.Get(ctx, projectID, combinedQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
