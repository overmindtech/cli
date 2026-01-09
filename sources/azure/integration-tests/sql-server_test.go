package integrationtests

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

func TestSQLServerIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	// Initialize Azure credentials using DefaultAzureCredential
	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	// Create Azure SDK clients
	sqlServerClient, err := armsql.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create SQL Servers client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Generate unique SQL server name (must be globally unique, lowercase, no special chars)
	sqlServerName := generateSQLServerName(integrationTestSQLServerName)

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create SQL server
		err = createSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create SQL server: %v", err)
		}

		// Wait for SQL server to be available
		err = waitForSQLServerAvailable(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName)
		if err != nil {
			t.Fatalf("Failed waiting for SQL server to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetSQLServer", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving SQL server %s in subscription %s, resource group %s",
				sqlServerName, subscriptionID, integrationTestResourceGroup)

			sqlServerWrapper := manual.NewSqlServer(
				clients.NewSqlServersClient(sqlServerClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := sqlServerWrapper.Scopes()[0]

			sqlServerAdapter := sources.WrapperToAdapter(sqlServerWrapper)
			sdpItem, qErr := sqlServerAdapter.Get(ctx, scope, sqlServerName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.SQLServer.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServer, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			if uniqueAttrValue != sqlServerName {
				t.Errorf("Expected unique attribute value %s, got %s", sqlServerName, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved SQL server %s", sqlServerName)
		})

		t.Run("ListSQLServers", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing SQL servers in resource group %s", integrationTestResourceGroup)

			sqlServerWrapper := manual.NewSqlServer(
				clients.NewSqlServersClient(sqlServerClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := sqlServerWrapper.Scopes()[0]

			sqlServerAdapter := sources.WrapperToAdapter(sqlServerWrapper)

			// Check if adapter supports list
			listable, ok := sqlServerAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list SQL servers: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one SQL server, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					if v == sqlServerName {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find SQL server %s in the list results", sqlServerName)
			}

			log.Printf("Found %d SQL servers in list results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for SQL server %s", sqlServerName)

			sqlServerWrapper := manual.NewSqlServer(
				clients.NewSqlServersClient(sqlServerClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := sqlServerWrapper.Scopes()[0]

			sqlServerAdapter := sources.WrapperToAdapter(sqlServerWrapper)
			sdpItem, qErr := sqlServerAdapter.Get(ctx, scope, sqlServerName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (SQL server has many child resources)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			// Verify expected child resource links exist
			expectedChildResources := map[string]bool{
				azureshared.SQLDatabase.String():                        false,
				azureshared.SQLElasticPool.String():                     false,
				azureshared.SQLServerFirewallRule.String():              false,
				azureshared.SQLServerVirtualNetworkRule.String():        false,
				azureshared.SQLServerKey.String():                       false,
				azureshared.SQLServerFailoverGroup.String():             false,
				azureshared.SQLServerAdministrator.String():             false,
				azureshared.SQLServerSyncGroup.String():                 false,
				azureshared.SQLServerSyncAgent.String():                 false,
				azureshared.SQLServerPrivateEndpointConnection.String(): false,
				azureshared.SQLServerAuditingSetting.String():           false,
				azureshared.SQLServerSecurityAlertPolicy.String():       false,
				azureshared.SQLServerVulnerabilityAssessment.String():   false,
				azureshared.SQLServerEncryptionProtector.String():       false,
				azureshared.SQLServerBlobAuditingPolicy.String():        false,
				azureshared.SQLServerAutomaticTuning.String():           false,
				azureshared.SQLServerAdvancedThreatProtectionSetting.String(): false,
				azureshared.SQLServerDnsAlias.String():                  false,
				azureshared.SQLServerUsage.String():                     false,
				azureshared.SQLServerOperation.String():                 false,
				azureshared.SQLServerAdvisor.String():                   false,
				azureshared.SQLServerBackupLongTermRetentionPolicy.String(): false,
				azureshared.SQLServerDevOpsAuditSetting.String():        false,
				azureshared.SQLServerTrustGroup.String():                false,
				azureshared.SQLServerOutboundFirewallRule.String():      false,
				azureshared.SQLServerPrivateLinkResource.String():       false,
			}

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				if expectedChildResources[linkedType] {
					t.Errorf("Found duplicate linked query for type %s", linkedType)
				}
				if _, exists := expectedChildResources[linkedType]; exists {
					expectedChildResources[linkedType] = true

					// Verify query method is SEARCH for child resources
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected linked query method SEARCH for %s, got %s", linkedType, liq.GetQuery().GetMethod())
					}

					// Verify query is the server name
					if liq.GetQuery().GetQuery() != sqlServerName {
						t.Errorf("Expected linked query to use server name %s, got %s", sqlServerName, liq.GetQuery().GetQuery())
					}

					// Verify scope matches
					if liq.GetQuery().GetScope() != scope {
						t.Errorf("Expected linked query scope %s, got %s", scope, liq.GetQuery().GetScope())
					}

					// Verify blast propagation is set
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Errorf("Expected BlastPropagation to be set for %s", linkedType)
					}
				}
			}

			// Check that all expected child resources are linked
			for resourceType, found := range expectedChildResources {
				if !found {
					t.Errorf("Expected linked query to %s, but didn't find one", resourceType)
				}
			}

			log.Printf("Verified %d linked item queries for SQL server %s", len(linkedQueries), sqlServerName)
		})

		t.Run("VerifyChildResourceBlastPropagation", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying blast propagation for child resources of SQL server %s", sqlServerName)

			sqlServerWrapper := manual.NewSqlServer(
				clients.NewSqlServersClient(sqlServerClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := sqlServerWrapper.Scopes()[0]

			sqlServerAdapter := sources.WrapperToAdapter(sqlServerWrapper)
			sdpItem, qErr := sqlServerAdapter.Get(ctx, scope, sqlServerName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()

			// Verify specific blast propagation patterns
			blastPropagationTests := map[string]struct {
				in  bool
				out bool
			}{
				// Child resources that depend on server (In: true, Out: false)
				azureshared.SQLDatabase.String():                        {in: true, out: false},
				azureshared.SQLElasticPool.String():                     {in: true, out: false},
				azureshared.SQLServerSyncGroup.String():                 {in: true, out: false},
				azureshared.SQLServerSyncAgent.String():                 {in: true, out: false},
				azureshared.SQLServerUsage.String():                     {in: true, out: false},
				azureshared.SQLServerOperation.String():                 {in: true, out: false},
				azureshared.SQLServerAdvisor.String():                   {in: true, out: false},
				azureshared.SQLServerPrivateLinkResource.String():       {in: true, out: false},
				// Child resources that affect server connectivity/security (In: true, Out: true)
				azureshared.SQLServerFirewallRule.String():              {in: true, out: true},
				azureshared.SQLServerVirtualNetworkRule.String():        {in: true, out: true},
				azureshared.SQLServerKey.String():                       {in: true, out: true},
				azureshared.SQLServerFailoverGroup.String():             {in: true, out: true},
				azureshared.SQLServerAdministrator.String():             {in: true, out: true},
				azureshared.SQLServerPrivateEndpointConnection.String(): {in: true, out: true},
				azureshared.SQLServerAuditingSetting.String():           {in: true, out: true},
				azureshared.SQLServerSecurityAlertPolicy.String():       {in: true, out: true},
				azureshared.SQLServerVulnerabilityAssessment.String():   {in: true, out: true},
				azureshared.SQLServerEncryptionProtector.String():       {in: true, out: true},
				azureshared.SQLServerBlobAuditingPolicy.String():        {in: true, out: true},
				azureshared.SQLServerAutomaticTuning.String():           {in: true, out: true},
				azureshared.SQLServerAdvancedThreatProtectionSetting.String(): {in: true, out: true},
				azureshared.SQLServerDnsAlias.String():                  {in: true, out: true},
				azureshared.SQLServerBackupLongTermRetentionPolicy.String(): {in: true, out: true},
				azureshared.SQLServerDevOpsAuditSetting.String():        {in: true, out: true},
				azureshared.SQLServerTrustGroup.String():                {in: true, out: true},
				azureshared.SQLServerOutboundFirewallRule.String():      {in: true, out: true},
			}

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				if expected, ok := blastPropagationTests[linkedType]; ok {
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Errorf("Expected BlastPropagation to be set for %s", linkedType)
						continue
					}

					if bp.GetIn() != expected.in {
						t.Errorf("Expected BlastPropagation.In=%v for %s, got %v", expected.in, linkedType, bp.GetIn())
					}

					if bp.GetOut() != expected.out {
						t.Errorf("Expected BlastPropagation.Out=%v for %s, got %v", expected.out, linkedType, bp.GetOut())
					}
				}
			}

			log.Printf("Verified blast propagation for all child resources")
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete SQL server
		err := deleteSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName)
		if err != nil {
			t.Fatalf("Failed to delete SQL server: %v", err)
		}

		// Optionally delete the resource group
		// Note: We keep the resource group for faster subsequent test runs
		// Uncomment the following if you want to clean up completely:
		// err = deleteResourceGroup(ctx, rgClient, integrationTestResourceGroup)
		// if err != nil {
		//     t.Fatalf("Failed to delete resource group: %v", err)
		// }
	})
}
