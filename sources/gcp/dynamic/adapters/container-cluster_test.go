package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/container/apiv1/containerpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestContainerCluster(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1-a"
	linker := gcpshared.NewLinker()
	clusterName := "test-cluster"

	// Create mock protobuf object
	cluster := &containerpb.Cluster{
		Name:        fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, clusterName),
		Description: "Test GKE Cluster",
		Network:     fmt.Sprintf("projects/%s/global/networks/default", projectID),
		Subnetwork:  fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/default", projectID),
		Location:    location,
		NodePools: []*containerpb.NodePool{
			{
				Name: "default-pool",
				Config: &containerpb.NodeConfig{
					ServiceAccount: fmt.Sprintf("test-service-account@%s.iam.gserviceaccount.com", projectID),
					BootDiskKmsKey: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key",
					// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/nodeGroups/{nodeGroup}
					NodeGroup: "projects/test-project/zones/us-central1-a/nodeGroups/test-node-group",
				},
			},
		},
		NotificationConfig: &containerpb.NotificationConfig{
			Pubsub: &containerpb.NotificationConfig_PubSub{
				Topic: fmt.Sprintf("projects/%s/topics/test-topic", projectID),
			},
		},
		UserManagedKeysConfig: &containerpb.UserManagedKeysConfig{
			ServiceAccountSigningKeys: []string{
				"projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key/cryptoKeyVersions/1",
			},
			ServiceAccountVerificationKeys: []string{
				"projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key/cryptoKeyVersions/2",
			},
			ControlPlaneDiskEncryptionKey: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/control-plane-key",
			GkeopsEtcdBackupEncryptionKey: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/etcd-backup-key",
		},
		DatabaseEncryption: &containerpb.DatabaseEncryption{
			KeyName: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/database-encryption-key",
			State:   containerpb.DatabaseEncryption_ENCRYPTED,
		},
		ResourceUsageExportConfig: &containerpb.ResourceUsageExportConfig{
			BigqueryDestination: &containerpb.ResourceUsageExportConfig_BigQueryDestination{
				DatasetId: "gke_usage_export",
			},
			EnableNetworkEgressMetering: true,
		},
		Endpoint: "35.123.45.67",
	}

	// Create second cluster for list testing
	clusterName2 := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, "test-cluster-2")
	cluster2 := &containerpb.Cluster{
		Name:        fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, "test-cluster-2"),
		Description: "Test GKE Cluster 2",
		Network:     fmt.Sprintf("projects/%s/global/networks/default", projectID),
		Location:    location,
	}

	// Create list response with multiple items
	clusterList := &containerpb.ListClustersResponse{
		Clusters: []*containerpb.Cluster{cluster, cluster2},
	}

	sdpItemType := gcpshared.ContainerCluster

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s", projectID, location, clusterName): {
			StatusCode: http.StatusOK,
			Body:       cluster,
		},
		fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s", projectID, location, clusterName2): {
			StatusCode: http.StatusOK,
			Body:       cluster2,
		},
		fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/%s/clusters", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       clusterList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		// For multiple query parameters, use the combined query format
		combinedQuery := shared.CompositeLookupKey(location, clusterName)
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
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		if val != fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, clusterName) {
			t.Errorf("Expected name field to be '%s', got %s", clusterName, val)
		}

		// Include static tests - covers ALL blast propagation links
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Network link
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Subnetwork link
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Service account link
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("test-service-account@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Boot disk KMS key link
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
				// Node group link
				{
					ExpectedType:   gcpshared.ComputeNodeGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-node-group",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, location),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Pub/Sub topic link
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
				// Service account signing key version link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "test-key", "1"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Service account verification key version link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "test-key", "2"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Control plane disk encryption key link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "control-plane-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// ETCD backup encryption key link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "etcd-backup-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Database encryption key link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "database-encryption-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// BigQuery dataset link
				{
					ExpectedType:   gcpshared.BigQueryDataset.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "gke_usage_export",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Master endpoint IP address link
				{
					ExpectedType:   "ip",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "35.123.45.67",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Forward link to node pools (parent to child)
				{
					ExpectedType:   gcpshared.ContainerNodePool.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  combinedQuery,
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

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
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

		// Test Terraform format: projects/[project]/locations/[location]/clusters/[cluster]
		terraformQuery := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, clusterName)
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
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s", projectID, location, clusterName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Cluster not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, clusterName)
		_, err = adapter.Get(ctx, projectID, combinedQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
