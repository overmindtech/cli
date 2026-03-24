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
	integrationTestSQLSchemaServerName   = "ovm-integ-test-schema-svr"
	integrationTestSQLSchemaDatabaseName = "ovm-integ-test-schema-db"
)

func TestSQLDatabaseSchemaIntegration(t *testing.T) {
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

	sqlDatabaseSchemasClient, err := armsql.NewDatabaseSchemasClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create SQL Database Schemas client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Generate unique SQL server name (must be globally unique, lowercase, no special chars)
	sqlServerName := generateSQLServerNameForSchemaTest(integrationTestSQLSchemaServerName)

	// Track if setup completed successfully
	setupCompleted := false

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create SQL server
		err = createSQLServerForSchemaTest(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName, integrationTestLocation)
		if err != nil {
			if errors.Is(err, errMissingSQLCredentials) {
				t.Skip("Skipping: SQL server admin credentials not configured")
			}
			t.Fatalf("Failed to create SQL server: %v", err)
		}

		// Wait for SQL server to be available
		err = waitForSQLServerAvailableForSchemaTest(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName)
		if err != nil {
			t.Fatalf("Failed waiting for SQL server to be available: %v", err)
		}

		// Create SQL database
		err = createSQLDatabaseForSchemaTest(ctx, sqlDatabaseClient, integrationTestResourceGroup, sqlServerName, integrationTestSQLSchemaDatabaseName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create SQL database: %v", err)
		}

		// Wait for SQL database to be available
		err = waitForSQLDatabaseAvailableForSchemaTest(ctx, sqlDatabaseClient, integrationTestResourceGroup, sqlServerName, integrationTestSQLSchemaDatabaseName)
		if err != nil {
			t.Fatalf("Failed waiting for SQL database to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		// First discover available schemas from the database (schemas are auto-created like dbo, sys, etc.)
		var testSchemaName string

		t.Run("DiscoverSchemas", func(t *testing.T) {
			ctx := t.Context()

			// List schemas to find an available one (dbo is standard in SQL Server databases)
			pager := sqlDatabaseSchemasClient.NewListByDatabasePager(integrationTestResourceGroup, sqlServerName, integrationTestSQLSchemaDatabaseName, nil)
			for pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					t.Fatalf("Failed to list schemas: %v", err)
				}
				if len(page.Value) > 0 && page.Value[0].Name != nil {
					testSchemaName = *page.Value[0].Name
					log.Printf("Discovered schema: %s", testSchemaName)
					break
				}
			}

			if testSchemaName == "" {
				t.Fatalf("No schemas found in database %s", integrationTestSQLSchemaDatabaseName)
			}
		})

		t.Run("GetSQLDatabaseSchema", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving SQL database schema %s in database %s, server %s",
				testSchemaName, integrationTestSQLSchemaDatabaseName, sqlServerName)

			schemaWrapper := manual.NewSqlDatabaseSchema(
				clients.NewSqlDatabaseSchemasClient(sqlDatabaseSchemasClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := schemaWrapper.Scopes()[0]

			schemaAdapter := sources.WrapperToAdapter(schemaWrapper, sdpcache.NewNoOpCache())
			// Get requires serverName, databaseName, and schemaName as query parts
			query := shared.CompositeLookupKey(sqlServerName, integrationTestSQLSchemaDatabaseName, testSchemaName)
			sdpItem, qErr := schemaAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.SQLDatabaseSchema.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLDatabaseSchema, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(sqlServerName, integrationTestSQLSchemaDatabaseName, testSchemaName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved SQL database schema %s", testSchemaName)
		})

		t.Run("SearchSQLDatabaseSchemas", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching SQL database schemas in database %s", integrationTestSQLSchemaDatabaseName)

			schemaWrapper := manual.NewSqlDatabaseSchema(
				clients.NewSqlDatabaseSchemasClient(sqlDatabaseSchemasClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := schemaWrapper.Scopes()[0]

			schemaAdapter := sources.WrapperToAdapter(schemaWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports search
			searchable, ok := schemaAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, shared.CompositeLookupKey(sqlServerName, integrationTestSQLSchemaDatabaseName), true)
			if err != nil {
				t.Fatalf("Failed to search SQL database schemas: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one SQL database schema, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					expectedValue := shared.CompositeLookupKey(sqlServerName, integrationTestSQLSchemaDatabaseName, testSchemaName)
					if v == expectedValue {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find schema %s in the search results", testSchemaName)
			}

			log.Printf("Found %d SQL database schemas in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for SQL database schema %s", testSchemaName)

			schemaWrapper := manual.NewSqlDatabaseSchema(
				clients.NewSqlDatabaseSchemasClient(sqlDatabaseSchemasClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := schemaWrapper.Scopes()[0]

			schemaAdapter := sources.WrapperToAdapter(schemaWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(sqlServerName, integrationTestSQLSchemaDatabaseName, testSchemaName)
			sdpItem, qErr := schemaAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (SQL database should be linked)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasSQLDatabaseLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() != "" {
					// Verify query structure
					if liq.GetQuery().GetQuery() == "" {
						t.Errorf("LinkedItemQuery has empty query")
					}
					if liq.GetQuery().GetScope() == "" {
						t.Errorf("LinkedItemQuery has empty scope")
					}
				}

				if liq.GetQuery().GetType() == azureshared.SQLDatabase.String() {
					hasSQLDatabaseLink = true
					expectedQuery := shared.CompositeLookupKey(sqlServerName, integrationTestSQLSchemaDatabaseName)
					if liq.GetQuery().GetQuery() != expectedQuery {
						t.Errorf("Expected linked query to SQL database %s, got %s", expectedQuery, liq.GetQuery().GetQuery())
					}
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected linked query method GET, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetScope() != scope {
						t.Errorf("Expected linked query scope %s, got %s", scope, liq.GetQuery().GetScope())
					}
				}
			}

			if !hasSQLDatabaseLink {
				t.Error("Expected linked query to SQL database, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for SQL database schema %s", len(linkedQueries), testSchemaName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			schemaWrapper := manual.NewSqlDatabaseSchema(
				clients.NewSqlDatabaseSchemasClient(sqlDatabaseSchemasClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := schemaWrapper.Scopes()[0]

			schemaAdapter := sources.WrapperToAdapter(schemaWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(sqlServerName, integrationTestSQLSchemaDatabaseName, testSchemaName)
			sdpItem, qErr := schemaAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.SQLDatabaseSchema.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLDatabaseSchema.String(), sdpItem.GetType())
			}

			// Verify scope
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			// Validate the item
			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for SQL database schema %s", testSchemaName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete SQL database
		err := deleteSQLDatabaseForSchemaTest(ctx, sqlDatabaseClient, integrationTestResourceGroup, sqlServerName, integrationTestSQLSchemaDatabaseName)
		if err != nil {
			t.Logf("Warning: Failed to delete SQL database: %v", err)
		}

		// Delete SQL server
		err = deleteSQLServerForSchemaTest(ctx, sqlServerClient, integrationTestResourceGroup, sqlServerName)
		if err != nil {
			t.Logf("Warning: Failed to delete SQL server: %v", err)
		}
	})
}

// errMissingSQLCredentials is a sentinel error for missing SQL credentials
var errMissingSQLCredentials = errors.New("AZURE_SQL_SERVER_ADMIN_LOGIN and AZURE_SQL_SERVER_ADMIN_PASSWORD environment variables must be set for integration tests")

// createSQLServerForSchemaTest creates an Azure SQL server for schema tests
func createSQLServerForSchemaTest(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName, location string) error {
	// Check if SQL server already exists
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

	// Get credentials from environment
	adminLogin := os.Getenv("AZURE_SQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_SQL_SERVER_ADMIN_PASSWORD")

	if adminLogin == "" || adminPassword == "" {
		return errMissingSQLCredentials
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, serverName, armsql.Server{
		Location: new(location),
		Properties: &armsql.ServerProperties{
			AdministratorLogin:         new(adminLogin),
			AdministratorLoginPassword: new(adminPassword),
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

// waitForSQLServerAvailableForSchemaTest waits for a SQL server to be available
func waitForSQLServerAvailableForSchemaTest(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName string) error {
	maxAttempts := 30
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

// createSQLDatabaseForSchemaTest creates an Azure SQL database for schema tests
func createSQLDatabaseForSchemaTest(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName, location string) error {
	// Check if SQL database already exists
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
		Location: new(location),
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

// waitForSQLDatabaseAvailableForSchemaTest waits for a SQL database to be available
func waitForSQLDatabaseAvailableForSchemaTest(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName string) error {
	maxAttempts := 30
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

// deleteSQLDatabaseForSchemaTest deletes an Azure SQL database
func deleteSQLDatabaseForSchemaTest(ctx context.Context, client *armsql.DatabasesClient, resourceGroup, serverName, databaseName string) error {
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

// deleteSQLServerForSchemaTest deletes an Azure SQL server
func deleteSQLServerForSchemaTest(ctx context.Context, client *armsql.ServersClient, resourceGroup, serverName string) error {
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

// generateSQLServerNameForSchemaTest generates a unique SQL server name
// SQL server names must be globally unique, 1-63 characters, lowercase letters, numbers, and hyphens
func generateSQLServerNameForSchemaTest(baseName string) string {
	baseName = strings.ToLower(baseName)
	baseName = strings.ReplaceAll(baseName, "_", "-")
	baseName = strings.ReplaceAll(baseName, " ", "-")

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(os.Getpid())))
	suffix := rng.Intn(10000)
	return fmt.Sprintf("%s-%04d", baseName, suffix)
}
