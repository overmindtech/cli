package integrationtests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// findExistingSQLServer searches for an existing SQL server in the resource group
// Returns the server name if found, empty string otherwise
func findExistingSQLServer(ctx context.Context, client *armsql.ServersClient, resourceGroup string) string {
	pager := client.NewListByResourceGroupPager(resourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			log.Printf("Failed to list SQL servers: %v", err)
			return ""
		}
		for _, server := range page.Value {
			if server.Name != nil && *server.Name != "" {
				log.Printf("Found existing SQL server: %s", *server.Name)
				return *server.Name
			}
		}
	}
	return ""
}

func TestSQLServerKeyIntegration(t *testing.T) {
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

	serverKeysClient, err := armsql.NewServerKeysClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create SQL Server Keys client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Track setup completion for skipping Run if Setup fails
	setupCompleted := false

	// Track if we created the server (for cleanup)
	serverCreated := false

	// SQL server name - will be set in Setup
	var sqlServerName string

	// The ServiceManaged key name is always "ServiceManaged"
	const serviceManagedKeyName = "ServiceManaged"

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// First, try to find an existing SQL server to reuse
		// This helps when admin credentials are not available
		sqlServerName = findExistingSQLServer(ctx, sqlServerClient, integrationTestResourceGroup)

		if sqlServerName == "" {
			// No existing server found, try to create one
			sqlServerName = generateSQLServerName(integrationTestSQLServerName)
			err = createSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName, integrationTestLocation)
			if err != nil {
				t.Skipf("Skipping test: Failed to create SQL server (admin credentials may be missing): %v", err)
			}
			serverCreated = true

			// Wait for SQL server to be available
			err = waitForSQLServerAvailable(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName)
			if err != nil {
				t.Fatalf("Failed waiting for SQL server to be available: %v", err)
			}
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetSQLServerKey", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving SQL server key %s for server %s in subscription %s, resource group %s",
				serviceManagedKeyName, sqlServerName, subscriptionID, integrationTestResourceGroup)

			serverKeyWrapper := manual.NewSqlServerKey(
				clients.NewSqlServerKeysClient(serverKeysClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := serverKeyWrapper.Scopes()[0]

			serverKeyAdapter := sources.WrapperToAdapter(serverKeyWrapper, sdpcache.NewNoOpCache())
			// Get requires serverName and keyName as query parts
			query := shared.CompositeLookupKey(sqlServerName, serviceManagedKeyName)
			sdpItem, qErr := serverKeyAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.SQLServerKey.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServerKey, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(sqlServerName, serviceManagedKeyName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved SQL server key %s", serviceManagedKeyName)
		})

		t.Run("SearchSQLServerKeys", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching SQL server keys for server %s", sqlServerName)

			serverKeyWrapper := manual.NewSqlServerKey(
				clients.NewSqlServerKeysClient(serverKeysClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := serverKeyWrapper.Scopes()[0]

			serverKeyAdapter := sources.WrapperToAdapter(serverKeyWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports search
			searchable, ok := serverKeyAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, sqlServerName, true)
			if err != nil {
				t.Fatalf("Failed to search SQL server keys: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one SQL server key, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					expectedValue := shared.CompositeLookupKey(sqlServerName, serviceManagedKeyName)
					if v == expectedValue {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find key %s in the search results", serviceManagedKeyName)
			}

			log.Printf("Found %d SQL server keys in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for SQL server key %s", serviceManagedKeyName)

			serverKeyWrapper := manual.NewSqlServerKey(
				clients.NewSqlServerKeysClient(serverKeysClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := serverKeyWrapper.Scopes()[0]

			serverKeyAdapter := sources.WrapperToAdapter(serverKeyWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(sqlServerName, serviceManagedKeyName)
			sdpItem, qErr := serverKeyAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (SQL server should be linked as parent)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			// Verify each linked item query has required fields
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET && liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Linked item query has invalid Method: %v", liq.GetQuery().GetMethod())
				}
				if liq.GetQuery().GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if liq.GetQuery().GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}
			}

			// Verify parent SQL Server link exists
			var hasSQLServerLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.SQLServer.String() {
					hasSQLServerLink = true
					if liq.GetQuery().GetQuery() != sqlServerName {
						t.Errorf("Expected linked query to SQL server %s, got %s", sqlServerName, liq.GetQuery().GetQuery())
					}
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected linked query method GET for SQL server, got %v", liq.GetQuery().GetMethod())
					}
					break
				}
			}

			if !hasSQLServerLink {
				t.Error("Expected linked query to parent SQL server, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for SQL server key %s", len(linkedQueries), serviceManagedKeyName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for SQL server key %s", serviceManagedKeyName)

			serverKeyWrapper := manual.NewSqlServerKey(
				clients.NewSqlServerKeysClient(serverKeysClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := serverKeyWrapper.Scopes()[0]

			serverKeyAdapter := sources.WrapperToAdapter(serverKeyWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(sqlServerName, serviceManagedKeyName)
			sdpItem, qErr := serverKeyAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify GetType returns the expected item type
			if sdpItem.GetType() != azureshared.SQLServerKey.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServerKey, sdpItem.GetType())
			}

			// Verify GetScope returns the expected scope
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify GetUniqueAttribute returns the correct attribute
			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			// Verify Validate passes
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for SQL server key %s", serviceManagedKeyName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Only delete the SQL server if we created it
		if serverCreated && sqlServerName != "" {
			err := deleteSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName)
			if err != nil {
				t.Fatalf("Failed to delete SQL server: %v", err)
			}
		} else {
			log.Printf("Skipping SQL server deletion (using pre-existing server)")
		}

		// We don't delete the resource group to allow faster subsequent test runs
	})
}
