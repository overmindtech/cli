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

func TestBigTableAdminCluster(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	instanceName := "test-instance"
	clusterName := "test-cluster"

	// Create mock protobuf cluster object
	bigTableCluster := &adminpb.Cluster{
		Name:               fmt.Sprintf("projects/%s/instances/%s/clusters/%s", projectID, instanceName, clusterName),
		Location:           fmt.Sprintf("projects/%s/locations/us-central1-a", projectID),
		State:              adminpb.Cluster_READY,
		ServeNodes:         3,
		DefaultStorageType: adminpb.StorageType_SSD,
		Config: &adminpb.Cluster_ClusterConfig_{
			ClusterConfig: &adminpb.Cluster_ClusterConfig{
				ClusterAutoscalingConfig: &adminpb.Cluster_ClusterAutoscalingConfig{
					AutoscalingLimits: &adminpb.AutoscalingLimits{
						MinServeNodes: 1,
						MaxServeNodes: 10,
					},
					AutoscalingTargets: &adminpb.AutoscalingTargets{
						CpuUtilizationPercent:        70,
						StorageUtilizationGibPerNode: 2500,
					},
				},
			},
		},
		EncryptionConfig: &adminpb.Cluster_EncryptionConfig{
			KmsKeyName: fmt.Sprintf("projects/%s/locations/us-central1/keyRings/test-keyring/cryptoKeys/test-key", projectID),
		},
	}

	// Create a second cluster for search testing
	clusterName2 := "test-cluster-2"
	bigTableCluster2 := &adminpb.Cluster{
		Name:               fmt.Sprintf("projects/%s/instances/%s/clusters/%s", projectID, instanceName, clusterName2),
		Location:           fmt.Sprintf("projects/%s/locations/us-east1-b", projectID),
		State:              adminpb.Cluster_CREATING,
		ServeNodes:         5,
		DefaultStorageType: adminpb.StorageType_HDD,
		// No encryption config for this cluster
	}

	// Mock response for search operation (list clusters in an instance)
	bigTableClustersList := &adminpb.ListClustersResponse{
		Clusters: []*adminpb.Cluster{bigTableCluster, bigTableCluster2},
	}

	sdpItemType := gcpshared.BigTableAdminCluster

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s", projectID, instanceName, clusterName): {
			StatusCode: http.StatusOK,
			Body:       bigTableCluster,
		},
		fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters", projectID, instanceName): {
			StatusCode: http.StatusOK,
			Body:       bigTableClustersList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		// Use composite query helper for BigTable Admin Cluster
		getQuery := shared.CompositeLookupKey(instanceName, clusterName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get BigTable Admin Cluster: %v", err)
		}

		// Basic item validation
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", getQuery, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Test specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/instances/%s/clusters/%s", projectID, instanceName, clusterName)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		val, err = sdpItem.GetAttributes().Get("location")
		if err != nil {
			t.Fatalf("Failed to get 'location' attribute: %v", err)
		}
		expectedLocation := fmt.Sprintf("projects/%s/locations/us-central1-a", projectID)
		if val != expectedLocation {
			t.Errorf("Expected location field to be '%s', got %s", expectedLocation, val)
		}

		val, err = sdpItem.GetAttributes().Get("state")
		if err != nil {
			t.Fatalf("Failed to get 'state' attribute: %v", err)
		}
		if val != "READY" {
			t.Errorf("Expected state field to be 'READY', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("serveNodes")
		if err != nil {
			t.Fatalf("Failed to get 'serveNodes' attribute: %v", err)
		}
		// serveNodes comes back as float64 from protojson.Marshal
		converted, ok := val.(float64)
		if !ok {
			t.Fatalf("Expected serveNodes to be a float64, got %T", val)
		}
		if converted != 3 {
			t.Errorf("Expected serveNodes field to be 3, got %f", converted)
		}

		val, err = sdpItem.GetAttributes().Get("defaultStorageType")
		if err != nil {
			t.Fatalf("Failed to get 'defaultStorageType' attribute: %v", err)
		}
		if val != "SSD" {
			t.Errorf("Expected defaultStorageType field to be 'SSD', got %s", val)
		}

		// Test nested attributes from protobuf
		val, err = sdpItem.GetAttributes().Get("encryptionConfig")
		if err != nil {
			t.Fatalf("Failed to get 'encryptionConfig' attribute: %v", err)
		}
		encryptionConfig, ok := val.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected encryptionConfig to be a map[string]interface{}, got %T", val)
		}
		expectedKmsKey := fmt.Sprintf("projects/%s/locations/us-central1/keyRings/test-keyring/cryptoKeys/test-key", projectID)
		if encryptionConfig["kmsKeyName"] != expectedKmsKey {
			t.Errorf("Expected encryptionConfig.kmsKeyName to be '%s', got %s", expectedKmsKey, encryptionConfig["kmsKeyName"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us-central1", "test-keyring", "test-key"),
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a SearchableAdapter")
		}

		// Search by instance name to get all clusters in that instance
		searchQuery := instanceName
		sdpItems, err := searchable.Search(ctx, projectID, searchQuery, true)
		if err != nil {
			t.Fatalf("Failed to search BigTable Admin Clusters: %v", err)
		}

		// Assert there are exactly 2 items
		if len(sdpItems) != 2 {
			t.Fatalf("Expected exactly 2 BigTable Admin Clusters, got %d", len(sdpItems))
		}

		// Validate first cluster
		item1 := sdpItems[0]
		if item1.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), item1.GetType())
		}
		if item1.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, item1.GetScope())
		}
		expectedUAV1 := shared.CompositeLookupKey(instanceName, clusterName)
		if item1.UniqueAttributeValue() != expectedUAV1 {
			t.Errorf("Expected first item unique attribute value '%s', got %s", expectedUAV1, item1.UniqueAttributeValue())
		}

		// Validate second cluster
		item2 := sdpItems[1]
		if item2.GetType() != sdpItemType.String() {
			t.Errorf("Expected second item type %s, got %s", sdpItemType.String(), item2.GetType())
		}
		if item2.GetScope() != projectID {
			t.Errorf("Expected second item scope '%s', got %s", projectID, item2.GetScope())
		}
		expectedUAV2 := shared.CompositeLookupKey(instanceName, clusterName2)
		if item2.UniqueAttributeValue() != expectedUAV2 {
			t.Errorf("Expected second item unique attribute value '%s', got %s", expectedUAV2, item2.UniqueAttributeValue())
		}

		// Validate specific attributes to ensure we have the correct items
		val, err := item2.GetAttributes().Get("state")
		if err != nil {
			t.Fatalf("Failed to get 'state' attribute from second item: %v", err)
		}
		if val != "CREATING" {
			t.Errorf("Expected second cluster state to be 'CREATING', got %s", val)
		}

		val, err = item2.GetAttributes().Get("defaultStorageType")
		if err != nil {
			t.Fatalf("Failed to get 'defaultStorageType' attribute from second item: %v", err)
		}
		if val != "HDD" {
			t.Errorf("Expected second cluster defaultStorageType to be 'HDD', got %s", val)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s", projectID, instanceName, clusterName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Cluster not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(instanceName, clusterName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent BigTable Admin Cluster, but got nil")
		}
	})
}
