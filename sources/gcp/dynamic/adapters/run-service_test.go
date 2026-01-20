package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/run/apiv2/runpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestRunService(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	serviceName := "test-service"

	service := &runpb.Service{
		Name: fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, location, serviceName),
		Template: &runpb.RevisionTemplate{
			ServiceAccount: "test-sa@test-project.iam.gserviceaccount.com",
			VpcAccess: &runpb.VpcAccess{
				Connector: fmt.Sprintf("projects/%s/locations/%s/connectors/test-connector", projectID, location),
				NetworkInterfaces: []*runpb.VpcAccess_NetworkInterface{
					{
						Network:    fmt.Sprintf("projects/%s/global/networks/default", projectID),
						Subnetwork: fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", projectID, location),
					},
				},
			},
			Containers: []*runpb.Container{
				{
					Image: fmt.Sprintf("%s-docker.pkg.dev/%s/repo/image:latest", location, projectID),
					Env: []*runpb.EnvVar{
						{
							Values: &runpb.EnvVar_ValueSource{
								ValueSource: &runpb.EnvVarSource{
									SecretKeyRef: &runpb.SecretKeySelector{
										Secret: fmt.Sprintf("projects/%s/secrets/api-key", projectID),
									},
								},
							},
						},
					},
				},
			},
			Volumes: []*runpb.Volume{
				{
					VolumeType: &runpb.Volume_Secret{
						Secret: &runpb.SecretVolumeSource{
							Secret: fmt.Sprintf("projects/%s/secrets/db-creds", projectID),
						},
					},
				},
				{
					VolumeType: &runpb.Volume_CloudSqlInstance{
						CloudSqlInstance: &runpb.CloudSqlInstance{
							Instances: []string{fmt.Sprintf("projects/%s/instances/test-db", projectID)},
						},
					},
				},
				{
					VolumeType: &runpb.Volume_Gcs{
						Gcs: &runpb.GCSVolumeSource{
							Bucket: "test-bucket",
						},
					},
				},
			},
			EncryptionKey: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key",
		},
		LatestReadyRevision:   fmt.Sprintf("projects/%s/locations/%s/services/%s/revisions/rev-1", projectID, location, serviceName),
		LatestCreatedRevision: fmt.Sprintf("projects/%s/locations/%s/services/%s/revisions/rev-2", projectID, location, serviceName),
		Traffic: []*runpb.TrafficTarget{
			{
				Revision: fmt.Sprintf("projects/%s/locations/%s/services/%s/revisions/rev-3", projectID, location, serviceName),
			},
		},
	}

	serviceName2 := "test-service-2"
	service2 := &runpb.Service{
		Name: fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, location, serviceName2),
	}

	serviceList := &runpb.ListServicesResponse{
		Services: []*runpb.Service{service, service2},
	}

	sdpItemType := gcpshared.RunService

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s", projectID, location, serviceName): {
			StatusCode: http.StatusOK,
			Body:       service,
		},
		fmt.Sprintf("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s", projectID, location, serviceName2): {
			StatusCode: http.StatusOK,
			Body:       service2,
		},
		fmt.Sprintf("https://run.googleapis.com/v2/projects/%s/locations/%s/services", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       serviceList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		// For multiple query parameters, use the combined query format
		combinedQuery := shared.CompositeLookupKey(location, serviceName)
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
		expectedName := fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, location, serviceName)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// template.serviceAccount
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-sa@test-project.iam.gserviceaccount.com",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// template.vpcAccess.networkInterfaces.network
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
				// template.vpcAccess.networkInterfaces.subnetwork
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, location),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// template.containers.env.valueSource.secretKeyRef.secret
				{
					ExpectedType:   gcpshared.SecretManagerSecret.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "api-key",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// template.volumes.secret.secret
				{
					ExpectedType:   gcpshared.SecretManagerSecret.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "db-creds",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// template.volumes.cloudSqlInstance.instances
				{
					ExpectedType:   gcpshared.SQLAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-db",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// template.volumes.gcs.bucket
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// template.encryptionKey
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
				// latestReadyRevision
				{
					ExpectedType:   gcpshared.RunRevision.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, serviceName, "rev-1"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				// latestCreatedRevision
				{
					ExpectedType:   gcpshared.RunRevision.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, serviceName, "rev-2"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				// traffic.revision
				{
					ExpectedType:   gcpshared.RunRevision.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, serviceName, "rev-3"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				// name (parent to child search)
				{
					ExpectedType:   gcpshared.RunRevision.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  shared.CompositeLookupKey(location, serviceName),
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

		// Test Terraform format: projects/[project_id]/locations/[location]/services/[service]
		terraformQuery := fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, location, serviceName)
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
			fmt.Sprintf("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s", projectID, location, serviceName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Service not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, serviceName)
		_, err = adapter.Get(ctx, projectID, combinedQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
