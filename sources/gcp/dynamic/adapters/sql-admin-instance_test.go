package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/sqladmin/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestSQLAdminInstance(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	instanceName := "test-sql-instance"

	instance := &sqladmin.DatabaseInstance{
		Name: fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName),
		Settings: &sqladmin.Settings{
			IpConfiguration: &sqladmin.IpConfiguration{
				PrivateNetwork: fmt.Sprintf("projects/%s/global/networks/default", projectID),
			},
			SqlServerAuditConfig: &sqladmin.SqlServerAuditConfig{
				Bucket: "audit-logs-bucket",
			},
		},
		DiskEncryptionConfiguration: &sqladmin.DiskEncryptionConfiguration{
			KmsKeyName: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key",
		},
		MasterInstanceName: "master-instance",
		FailoverReplica: &sqladmin.DatabaseInstanceFailoverReplica{
			Name: "failover-replica",
		},
		ReplicaNames:               []string{"replica-1", "replica-2"},
		ServiceAccountEmailAddress: "test-sa@test-project.iam.gserviceaccount.com",
		DnsName:                    "test-sql-instance.database.google.com",
		IpAddresses: []*sqladmin.IpMapping{
			{
				IpAddress: "10.0.0.50",
			},
		},
		Ipv6Address: "2001:db8::1",
	}

	instanceName2 := "test-sql-instance-2"
	instance2 := &sqladmin.DatabaseInstance{
		Name: fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName2),
	}

	instanceList := &sqladmin.InstancesListResponse{
		Items: []*sqladmin.DatabaseInstance{instance, instance2},
	}

	sdpItemType := gcpshared.SQLAdminInstance

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://sqladmin.googleapis.com/sql/v1/projects/%s/instances/%s", projectID, instanceName): {
			StatusCode: http.StatusOK,
			Body:       instance,
		},
		fmt.Sprintf("https://sqladmin.googleapis.com/sql/v1/projects/%s/instances/%s", projectID, instanceName2): {
			StatusCode: http.StatusOK,
			Body:       instance2,
		},
		fmt.Sprintf("https://sqladmin.googleapis.com/sql/v1/projects/%s/instances", projectID): {
			StatusCode: http.StatusOK,
			Body:       instanceList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, instanceName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != instanceName {
			t.Errorf("Expected unique attribute value '%s', got %s", instanceName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		// Include static tests - covers ALL blast propagation links
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// settings.ipConfiguration.privateNetwork
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
				// diskEncryptionConfiguration.kmsKeyName
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
				// settings.sqlServerAuditConfig.bucket
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "audit-logs-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// ipAddresses.ipAddress
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.50",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// ipv6Address
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "2001:db8::1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// serviceAccountEmailAddress
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
				// dnsName
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "test-sql-instance.database.google.com",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// masterInstanceName
				{
					ExpectedType:   gcpshared.SQLAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "master-instance",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// failoverReplica.name
				{
					ExpectedType:   gcpshared.SQLAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "failover-replica",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// replicaNames[0]
				{
					ExpectedType:   gcpshared.SQLAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "replica-1",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				// replicaNames[1]
				{
					ExpectedType:   gcpshared.SQLAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "replica-2",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				// name (parent to child search)
				{
					ExpectedType:   gcpshared.SQLAdminBackupRun.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  instanceName,
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

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
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

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://sqladmin.googleapis.com/sql/v1/projects/%s/instances/%s", projectID, instanceName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Instance not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, instanceName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
