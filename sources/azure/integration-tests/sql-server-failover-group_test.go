package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
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

const (
	integrationTestFailoverGroupName   = "ovm-integ-test-failover-group"
	integrationTestPrimaryServerName   = "ovm-integ-test-primary-server"
	integrationTestSecondaryServerName = "ovm-integ-test-secondary-server"
	integrationTestPrimaryLocation     = "westus2"
	integrationTestSecondaryLocation   = "eastus"
	integrationTestFailoverGroupDBName = "ovm-integ-test-fg-database"
)

func TestSQLServerFailoverGroupIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	// SQL server admin credentials are required for creating SQL servers
	adminLogin := os.Getenv("AZURE_SQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_SQL_SERVER_ADMIN_PASSWORD")
	if adminLogin == "" || adminPassword == "" {
		t.Skip("AZURE_SQL_SERVER_ADMIN_LOGIN and AZURE_SQL_SERVER_ADMIN_PASSWORD environment variables must be set for SQL failover group integration tests")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	// Create Azure SDK clients
	sqlServerClient, err := armsql.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create SQL Servers client: %v", err)
	}

	sqlDatabaseClient, err := armsql.NewDatabasesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create SQL Databases client: %v", err)
	}

	sqlFailoverGroupClient, err := armsql.NewFailoverGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create SQL Failover Groups client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Generate unique names for SQL servers (must be globally unique)
	primaryServerName := generateFailoverGroupServerName(integrationTestPrimaryServerName)
	secondaryServerName := generateFailoverGroupServerName(integrationTestSecondaryServerName)

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create primary SQL server
		err = createFailoverGroupSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, primaryServerName, integrationTestPrimaryLocation)
		if err != nil {
			t.Fatalf("Failed to create primary SQL server: %v", err)
		}

		// Wait for primary SQL server to be available
		err = waitForFailoverGroupSQLServerAvailable(ctx, sqlServerClient, integrationTestResourceGroup, primaryServerName)
		if err != nil {
			t.Fatalf("Failed waiting for primary SQL server to be available: %v", err)
		}

		// Create secondary SQL server (in a different region)
		err = createFailoverGroupSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, secondaryServerName, integrationTestSecondaryLocation)
		if err != nil {
			t.Fatalf("Failed to create secondary SQL server: %v", err)
		}

		// Wait for secondary SQL server to be available
		err = waitForFailoverGroupSQLServerAvailable(ctx, sqlServerClient, integrationTestResourceGroup, secondaryServerName)
		if err != nil {
			t.Fatalf("Failed waiting for secondary SQL server to be available: %v", err)
		}

		// Create a database on the primary server (failover groups need at least one database)
		err = createFailoverGroupDatabase(ctx, sqlDatabaseClient, integrationTestResourceGroup, primaryServerName, integrationTestFailoverGroupDBName, integrationTestPrimaryLocation)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		// Wait for database to be available
		err = waitForFailoverGroupDatabaseAvailable(ctx, sqlDatabaseClient, integrationTestResourceGroup, primaryServerName, integrationTestFailoverGroupDBName)
		if err != nil {
			t.Fatalf("Failed waiting for database to be available: %v", err)
		}

		// Create the failover group
		err = createFailoverGroup(ctx, sqlFailoverGroupClient, integrationTestResourceGroup, primaryServerName, secondaryServerName, integrationTestFailoverGroupName, subscriptionID)
		if err != nil {
			t.Fatalf("Failed to create failover group: %v", err)
		}

		// Wait for the failover group to be available
		err = waitForFailoverGroupAvailable(ctx, sqlFailoverGroupClient, integrationTestResourceGroup, primaryServerName, integrationTestFailoverGroupName)
		if err != nil {
			t.Fatalf("Failed waiting for failover group to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetSQLServerFailoverGroup", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving failover group %s in SQL server %s, subscription %s, resource group %s",
				integrationTestFailoverGroupName, primaryServerName, subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewSqlServerFailoverGroup(
				clients.NewSqlFailoverGroupsClient(sqlFailoverGroupClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(primaryServerName, integrationTestFailoverGroupName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.SQLServerFailoverGroup.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServerFailoverGroup, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(primaryServerName, integrationTestFailoverGroupName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved failover group %s", integrationTestFailoverGroupName)
		})

		t.Run("SearchSQLServerFailoverGroups", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching failover groups in SQL server %s", primaryServerName)

			wrapper := manual.NewSqlServerFailoverGroup(
				clients.NewSqlFailoverGroupsClient(sqlFailoverGroupClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, primaryServerName, true)
			if err != nil {
				t.Fatalf("Failed to search failover groups: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one failover group, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					expectedValue := shared.CompositeLookupKey(primaryServerName, integrationTestFailoverGroupName)
					if v == expectedValue {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find failover group %s in the search results", integrationTestFailoverGroupName)
			}

			log.Printf("Found %d failover groups in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for failover group %s", integrationTestFailoverGroupName)

			wrapper := manual.NewSqlServerFailoverGroup(
				clients.NewSqlFailoverGroupsClient(sqlFailoverGroupClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(primaryServerName, integrationTestFailoverGroupName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasPrimaryServerLink bool
			var hasPartnerServerLink bool
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query.GetType() == "" {
					t.Error("Found linked query with empty type")
				}
				if query.GetMethod() != sdp.QueryMethod_GET && query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Found linked query with invalid method: %s", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Found linked query with empty query")
				}
				if query.GetScope() == "" {
					t.Error("Found linked query with empty scope")
				}

				if query.GetType() == azureshared.SQLServer.String() {
					if query.GetQuery() == primaryServerName {
						hasPrimaryServerLink = true
					}
					if query.GetQuery() == secondaryServerName {
						hasPartnerServerLink = true
					}
				}
			}

			if !hasPrimaryServerLink {
				t.Error("Expected linked query to primary SQL server, but didn't find one")
			}

			if !hasPartnerServerLink {
				t.Error("Expected linked query to partner (secondary) SQL server, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for failover group %s", len(linkedQueries), integrationTestFailoverGroupName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewSqlServerFailoverGroup(
				clients.NewSqlFailoverGroupsClient(sqlFailoverGroupClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(primaryServerName, integrationTestFailoverGroupName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.SQLServerFailoverGroup.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServerFailoverGroup, sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete the failover group first
		err := deleteFailoverGroup(ctx, sqlFailoverGroupClient, integrationTestResourceGroup, primaryServerName, integrationTestFailoverGroupName)
		if err != nil {
			t.Logf("Warning: Failed to delete failover group: %v", err)
		}

		// Delete the database
		err = deleteFailoverGroupDatabase(ctx, sqlDatabaseClient, integrationTestResourceGroup, primaryServerName, integrationTestFailoverGroupDBName)
		if err != nil {
			t.Logf("Warning: Failed to delete database: %v", err)
		}

		// Delete secondary SQL server first (since failover group is deleted)
		err = deleteFailoverGroupSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, secondaryServerName)
		if err != nil {
			t.Logf("Warning: Failed to delete secondary SQL server: %v", err)
		}

		// Delete primary SQL server
		err = deleteFailoverGroupSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, primaryServerName)
		if err != nil {
			t.Logf("Warning: Failed to delete primary SQL server: %v", err)
		}
	})
}

// generateFailoverGroupServerName generates a unique SQL server name for failover group tests
func generateFailoverGroupServerName(baseName string) string {
	baseName = strings.ToLower(baseName)
	baseName = strings.ReplaceAll(baseName, "_", "-")
	baseName = strings.ReplaceAll(baseName, " ", "-")

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(os.Getpid())))
	suffix := rng.Intn(10000)
	return fmt.Sprintf("%s-%04d", baseName, suffix)
}

// createFailoverGroupSQLServer creates an Azure SQL server for failover group testing
func createFailoverGroupSQLServer(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName, location string) error {
	_, err := client.Get(ctx, resourceGroup, serverName, nil)
	if err == nil {
		log.Printf("SQL server %s already exists, skipping creation", serverName)
		return nil
	}

	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return fmt.Errorf("failed to check if SQL server exists: %w", err)
	}
	if respErr != nil && respErr.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to check if SQL server exists: %w", err)
	}

	adminLogin := os.Getenv("AZURE_SQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_SQL_SERVER_ADMIN_PASSWORD")

	if adminLogin == "" || adminPassword == "" {
		return fmt.Errorf("AZURE_SQL_SERVER_ADMIN_LOGIN and AZURE_SQL_SERVER_ADMIN_PASSWORD environment variables must be set for integration tests")
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, serverName, armsql.Server{
		Location: &location,
		Properties: &armsql.ServerProperties{
			AdministratorLogin:         &adminLogin,
			AdministratorLoginPassword: &adminPassword,
			Version:                    new("12.0"),
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"managed": new("true"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to start SQL server creation: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create SQL server: %w", err)
	}

	log.Printf("SQL server %s created successfully in location %s", serverName, location)
	return nil
}

// waitForFailoverGroupSQLServerAvailable waits for a SQL server to be available
func waitForFailoverGroupSQLServerAvailable(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName string) error {
	maxAttempts := 60 // Longer timeout for failover group tests
	for range maxAttempts {
		server, err := client.Get(ctx, resourceGroup, serverName, nil)
		if err == nil {
			if server.Properties != nil && server.Properties.State != nil && *server.Properties.State == "Ready" {
				return nil
			}
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("SQL server %s did not become available within expected time", serverName)
}

// createFailoverGroupDatabase creates an Azure SQL database for failover group
func createFailoverGroupDatabase(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName, location string) error {
	_, err := client.Get(ctx, resourceGroup, serverName, databaseName, nil)
	if err == nil {
		log.Printf("SQL database %s already exists, skipping creation", databaseName)
		return nil
	}

	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return fmt.Errorf("failed to check if SQL database exists: %w", err)
	}
	if respErr != nil && respErr.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to check if SQL database exists: %w", err)
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, serverName, databaseName, armsql.Database{
		Location: &location,
		Properties: &armsql.DatabaseProperties{
			RequestedServiceObjectiveName: new("Basic"),
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"managed": new("true"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to start SQL database creation: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create SQL database: %w", err)
	}

	log.Printf("SQL database %s created successfully in server %s", databaseName, serverName)
	return nil
}

// waitForFailoverGroupDatabaseAvailable waits for a SQL database to be available
func waitForFailoverGroupDatabaseAvailable(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName string) error {
	maxAttempts := 60
	for range maxAttempts {
		database, err := client.Get(ctx, resourceGroup, serverName, databaseName, nil)
		if err == nil {
			if database.Properties != nil && database.Properties.Status != nil && *database.Properties.Status == armsql.DatabaseStatusOnline {
				return nil
			}
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("SQL database %s did not become available within expected time", databaseName)
}

// createFailoverGroup creates an Azure SQL Failover Group
func createFailoverGroup(ctx context.Context, client *armsql.FailoverGroupsClient, resourceGroup, primaryServerName, secondaryServerName, failoverGroupName, subscriptionID string) error {
	_, err := client.Get(ctx, resourceGroup, primaryServerName, failoverGroupName, nil)
	if err == nil {
		log.Printf("Failover group %s already exists, skipping creation", failoverGroupName)
		return nil
	}

	var respErr *azcore.ResponseError
	if !errors.As(err, &respErr) {
		return fmt.Errorf("failed to check if failover group exists: %w", err)
	}
	if respErr != nil && respErr.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to check if failover group exists: %w", err)
	}

	secondaryServerID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Sql/servers/%s",
		subscriptionID, resourceGroup, secondaryServerName)

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, primaryServerName, failoverGroupName, armsql.FailoverGroup{
		Properties: &armsql.FailoverGroupProperties{
			PartnerServers: []*armsql.PartnerInfo{
				{
					ID: &secondaryServerID,
				},
			},
			ReadWriteEndpoint: &armsql.FailoverGroupReadWriteEndpoint{
				FailoverPolicy:                         new(armsql.ReadWriteEndpointFailoverPolicyAutomatic),
				FailoverWithDataLossGracePeriodMinutes: new(int32(60)),
			},
			ReadOnlyEndpoint: &armsql.FailoverGroupReadOnlyEndpoint{
				FailoverPolicy: new(armsql.ReadOnlyEndpointFailoverPolicyDisabled),
			},
			Databases: []*string{},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"managed": new("true"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to start failover group creation: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create failover group: %w", err)
	}

	log.Printf("Failover group %s created successfully", failoverGroupName)
	return nil
}

// waitForFailoverGroupAvailable waits for a failover group to be available
func waitForFailoverGroupAvailable(ctx context.Context, client *armsql.FailoverGroupsClient, resourceGroup, serverName, failoverGroupName string) error {
	maxAttempts := 60
	for range maxAttempts {
		fg, err := client.Get(ctx, resourceGroup, serverName, failoverGroupName, nil)
		if err == nil {
			// Replication state can be empty string (ready), "CATCH_UP", "PENDING", "SEEDING", "SUSPENDED"
			if fg.Properties != nil && fg.Properties.ReplicationState != nil {
				state := *fg.Properties.ReplicationState
				if state == "" || state == "CATCH_UP" {
					// Empty string or CATCH_UP indicates the failover group is functional
					return nil
				}
			} else if fg.Properties != nil {
				// ReplicationState is nil, check if properties exist (group created)
				return nil
			}
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("failover group %s did not become available within expected time", failoverGroupName)
}

// deleteFailoverGroup deletes an Azure SQL Failover Group
func deleteFailoverGroup(ctx context.Context, client *armsql.FailoverGroupsClient, resourceGroup, serverName, failoverGroupName string) error {
	_, err := client.Get(ctx, resourceGroup, serverName, failoverGroupName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Failover group %s does not exist, skipping deletion", failoverGroupName)
			return nil
		}
		return fmt.Errorf("failed to check if failover group exists: %w", err)
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, serverName, failoverGroupName, nil)
	if err != nil {
		return fmt.Errorf("failed to start failover group deletion: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete failover group: %w", err)
	}

	log.Printf("Failover group %s deleted successfully", failoverGroupName)
	return nil
}

// deleteFailoverGroupDatabase deletes an Azure SQL database
func deleteFailoverGroupDatabase(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName string) error {
	_, err := client.Get(ctx, resourceGroup, serverName, databaseName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("SQL database %s does not exist, skipping deletion", databaseName)
			return nil
		}
		return fmt.Errorf("failed to check if SQL database exists: %w", err)
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, serverName, databaseName, nil)
	if err != nil {
		return fmt.Errorf("failed to start SQL database deletion: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete SQL database: %w", err)
	}

	log.Printf("SQL database %s deleted successfully", databaseName)
	return nil
}

// deleteFailoverGroupSQLServer deletes an Azure SQL server
func deleteFailoverGroupSQLServer(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName string) error {
	_, err := client.Get(ctx, resourceGroup, serverName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("SQL server %s does not exist, skipping deletion", serverName)
			return nil
		}
		return fmt.Errorf("failed to check if SQL server exists: %w", err)
	}

	poller, err := client.BeginDelete(ctx, resourceGroup, serverName, nil)
	if err != nil {
		return fmt.Errorf("failed to start SQL server deletion: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete SQL server: %w", err)
	}

	log.Printf("SQL server %s deleted successfully", serverName)
	return nil
}
