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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestSQLServerName   = "ovm-integ-test-sql-server"
	integrationTestSQLDatabaseName = "ovm-integ-test-database"
)

func TestSQLDatabaseIntegration(t *testing.T) {
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

	sqlDatabaseClient, err := armsql.NewDatabasesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create SQL Databases client: %v", err)
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

		// Create SQL database
		err = createSQLDatabase(ctx, sqlDatabaseClient, integrationTestResourceGroup, sqlServerName, integrationTestSQLDatabaseName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create SQL database: %v", err)
		}

		// Wait for SQL database to be available
		err = waitForSQLDatabaseAvailable(ctx, sqlDatabaseClient, integrationTestResourceGroup, sqlServerName, integrationTestSQLDatabaseName)
		if err != nil {
			t.Fatalf("Failed waiting for SQL database to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetSQLDatabase", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving SQL database %s in SQL server %s, subscription %s, resource group %s",
				integrationTestSQLDatabaseName, sqlServerName, subscriptionID, integrationTestResourceGroup)

			sqlDbWrapper := manual.NewSqlDatabase(
				clients.NewSqlDatabasesClient(sqlDatabaseClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := sqlDbWrapper.Scopes()[0]

			sqlDbAdapter := sources.WrapperToAdapter(sqlDbWrapper)
			// Get requires serverName and databaseName as query parts
			query := sqlServerName + shared.QuerySeparator + integrationTestSQLDatabaseName
			sdpItem, qErr := sqlDbAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.SQLDatabase.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLDatabase, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(sqlServerName, integrationTestSQLDatabaseName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved SQL database %s", integrationTestSQLDatabaseName)
		})

		t.Run("SearchSQLDatabases", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching SQL databases in SQL server %s", sqlServerName)

			sqlDbWrapper := manual.NewSqlDatabase(
				clients.NewSqlDatabasesClient(sqlDatabaseClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := sqlDbWrapper.Scopes()[0]

			sqlDbAdapter := sources.WrapperToAdapter(sqlDbWrapper)

			// Check if adapter supports search
			searchable, ok := sqlDbAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, sqlServerName, true)
			if err != nil {
				t.Fatalf("Failed to search SQL databases: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one SQL database, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					expectedValue := shared.CompositeLookupKey(sqlServerName, integrationTestSQLDatabaseName)
					if v == expectedValue {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find database %s in the search results", integrationTestSQLDatabaseName)
			}

			log.Printf("Found %d SQL databases in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for SQL database %s", integrationTestSQLDatabaseName)

			sqlDbWrapper := manual.NewSqlDatabase(
				clients.NewSqlDatabasesClient(sqlDatabaseClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := sqlDbWrapper.Scopes()[0]

			sqlDbAdapter := sources.WrapperToAdapter(sqlDbWrapper)
			query := sqlServerName + shared.QuerySeparator + integrationTestSQLDatabaseName
			sdpItem, qErr := sqlDbAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (SQL server should be linked)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasSQLServerLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.SQLServer.String() {
					hasSQLServerLink = true
					if liq.GetQuery().GetQuery() != sqlServerName {
						t.Errorf("Expected linked query to SQL server %s, got %s", sqlServerName, liq.GetQuery().GetQuery())
					}
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected linked query method GET, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetScope() != scope {
						t.Errorf("Expected linked query scope %s, got %s", scope, liq.GetQuery().GetScope())
					}
					// Verify blast propagation
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Error("Expected BlastPropagation to be set for SQL server link")
					} else {
						if !bp.GetIn() {
							t.Error("Expected BlastPropagation.In to be true for SQL server link")
						}
						if bp.GetOut() {
							t.Error("Expected BlastPropagation.Out to be false for SQL server link")
						}
					}
					break
				}
			}

			if !hasSQLServerLink {
				t.Error("Expected linked query to SQL server, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for SQL database %s", len(linkedQueries), integrationTestSQLDatabaseName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete SQL database
		err := deleteSQLDatabase(ctx, sqlDatabaseClient, integrationTestResourceGroup, sqlServerName, integrationTestSQLDatabaseName)
		if err != nil {
			t.Fatalf("Failed to delete SQL database: %v", err)
		}

		// Delete SQL server
		err = deleteSQLServer(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName)
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

// generateSQLServerName generates a unique SQL server name
// SQL server names must be globally unique, 1-63 characters, lowercase letters, numbers, and hyphens
func generateSQLServerName(baseName string) string {
	// Ensure base name is lowercase and valid
	baseName = strings.ToLower(baseName)
	// Remove any invalid characters (only alphanumeric and hyphens allowed)
	baseName = strings.ReplaceAll(baseName, "_", "-")
	// Remove any invalid characters
	baseName = strings.ReplaceAll(baseName, " ", "-")

	// Add random suffix for uniqueness (4 characters)
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(os.Getpid())))
	suffix := rng.Intn(10000)
	return fmt.Sprintf("%s-%04d", baseName, suffix)
}

// createSQLServer creates an Azure SQL server
func createSQLServer(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName, location string) error {
	// Check if SQL server already exists
	_, err := client.Get(ctx, resourceGroup, serverName, nil)
	if err == nil {
		log.Printf("SQL server %s already exists, skipping creation", serverName)
		return nil
	}

	var respErr *azcore.ResponseError
	if err != nil && !errors.As(err, &respErr) {
		// Some other error occurred
		return fmt.Errorf("failed to check if SQL server exists: %w", err)
	}
	if respErr != nil && respErr.StatusCode != http.StatusNotFound {
		// Server exists or other error
		if respErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("failed to check if SQL server exists: %w", err)
		}
	}

	// Create the SQL server
	// Note: SQL servers require administrator login credentials
	// Credentials are read from environment variables to avoid committing secrets to source control
	adminLogin := os.Getenv("AZURE_SQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_SQL_SERVER_ADMIN_PASSWORD")

	if adminLogin == "" || adminPassword == "" {
		return fmt.Errorf("AZURE_SQL_SERVER_ADMIN_LOGIN and AZURE_SQL_SERVER_ADMIN_PASSWORD environment variables must be set for integration tests")
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, serverName, armsql.Server{
		Location: ptr.To(location),
		Properties: &armsql.ServerProperties{
			AdministratorLogin:         ptr.To(adminLogin),
			AdministratorLoginPassword: ptr.To(adminPassword),
			Version:                    ptr.To("12.0"),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"managed": ptr.To("true"),
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

// waitForSQLServerAvailable waits for a SQL server to be available
func waitForSQLServerAvailable(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName string) error {
	maxAttempts := 30
	for range maxAttempts {
		server, err := client.Get(ctx, resourceGroup, serverName, nil)
		if err == nil {
			// Server exists, check if it's ready (state should be "Ready")
			if server.Properties != nil && server.Properties.State != nil && *server.Properties.State == "Ready" {
				return nil
			}
		}

		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("SQL server %s did not become available within expected time", serverName)
}

// createSQLDatabase creates an Azure SQL database
func createSQLDatabase(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName, location string) error {
	// Check if SQL database already exists
	_, err := client.Get(ctx, resourceGroup, serverName, databaseName, nil)
	if err == nil {
		log.Printf("SQL database %s already exists, skipping creation", databaseName)
		return nil
	}

	var respErr *azcore.ResponseError
	if err != nil && !errors.As(err, &respErr) {
		// Some other error occurred
		return fmt.Errorf("failed to check if SQL database exists: %w", err)
	}
	if respErr != nil && respErr.StatusCode != http.StatusNotFound {
		// Database exists or other error
		if respErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("failed to check if SQL database exists: %w", err)
		}
	}

	// Create the SQL database
	// Using Basic tier for integration tests (cheaper)
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, serverName, databaseName, armsql.Database{
		Location: ptr.To(location),
		Properties: &armsql.DatabaseProperties{
			RequestedServiceObjectiveName: ptr.To("Basic"),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"managed": ptr.To("true"),
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

// waitForSQLDatabaseAvailable waits for a SQL database to be available
func waitForSQLDatabaseAvailable(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName string) error {
	maxAttempts := 30
	for range maxAttempts {
		database, err := client.Get(ctx, resourceGroup, serverName, databaseName, nil)
		if err == nil {
			// Database exists, check if it's ready (status should be "Online")
			if database.Properties != nil && database.Properties.Status != nil && *database.Properties.Status == armsql.DatabaseStatusOnline {
				return nil
			}
		}

		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("SQL database %s did not become available within expected time", databaseName)
}

// deleteSQLDatabase deletes an Azure SQL database
func deleteSQLDatabase(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName string) error {
	// Check if database exists before attempting to delete
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

// deleteSQLServer deletes an Azure SQL server
func deleteSQLServer(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName string) error {
	// Check if server exists before attempting to delete
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
