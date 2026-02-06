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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestPostgreSQLServerName   = "ovm-integ-test-pg-server"
	integrationTestPostgreSQLDatabaseName = "ovm-integ-test-database"
)

func TestDBforPostgreSQLDatabaseIntegration(t *testing.T) {
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
	postgreSQLServerClient, err := armpostgresqlflexibleservers.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Flexible Servers client: %v", err)
	}

	postgreSQLDatabaseClient, err := armpostgresqlflexibleservers.NewDatabasesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Databases client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Generate unique PostgreSQL server name (must be globally unique, lowercase, no special chars)
	postgreSQLServerName := generatePostgreSQLServerName(integrationTestPostgreSQLServerName)

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create PostgreSQL Flexible Server
		err = createPostgreSQLFlexibleServer(ctx, postgreSQLServerClient, integrationTestResourceGroup, postgreSQLServerName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL Flexible Server: %v", err)
		}

		// Wait for PostgreSQL server to be available
		err = waitForPostgreSQLServerAvailable(ctx, postgreSQLServerClient, integrationTestResourceGroup, postgreSQLServerName)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL server to be available: %v", err)
		}

		// Create PostgreSQL database
		err = createPostgreSQLDatabase(ctx, postgreSQLDatabaseClient, integrationTestResourceGroup, postgreSQLServerName, integrationTestPostgreSQLDatabaseName)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL database: %v", err)
		}

		// Wait for PostgreSQL database to be available
		err = waitForPostgreSQLDatabaseAvailable(ctx, postgreSQLDatabaseClient, integrationTestResourceGroup, postgreSQLServerName, integrationTestPostgreSQLDatabaseName)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL database to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetPostgreSQLDatabase", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving PostgreSQL database %s in server %s, subscription %s, resource group %s",
				integrationTestPostgreSQLDatabaseName, postgreSQLServerName, subscriptionID, integrationTestResourceGroup)

			pgDbWrapper := manual.NewDBforPostgreSQLDatabase(
				clients.NewPostgreSQLDatabasesClient(postgreSQLDatabaseClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pgDbWrapper.Scopes()[0]

			pgDbAdapter := sources.WrapperToAdapter(pgDbWrapper, sdpcache.NewNoOpCache())
			// Get requires serverName and databaseName as query parts
			query := shared.CompositeLookupKey(postgreSQLServerName, integrationTestPostgreSQLDatabaseName)
			sdpItem, qErr := pgDbAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLDatabase.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLDatabase, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(postgreSQLServerName, integrationTestPostgreSQLDatabaseName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved PostgreSQL database %s", integrationTestPostgreSQLDatabaseName)
		})

		t.Run("SearchPostgreSQLDatabases", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching PostgreSQL databases in server %s", postgreSQLServerName)

			pgDbWrapper := manual.NewDBforPostgreSQLDatabase(
				clients.NewPostgreSQLDatabasesClient(postgreSQLDatabaseClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pgDbWrapper.Scopes()[0]

			pgDbAdapter := sources.WrapperToAdapter(pgDbWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports search
			searchable, ok := pgDbAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, postgreSQLServerName, true)
			if err != nil {
				t.Fatalf("Failed to search PostgreSQL databases: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one PostgreSQL database, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					expectedValue := shared.CompositeLookupKey(postgreSQLServerName, integrationTestPostgreSQLDatabaseName)
					if v == expectedValue {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find database %s in the search results", integrationTestPostgreSQLDatabaseName)
			}

			log.Printf("Found %d PostgreSQL databases in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for PostgreSQL database %s", integrationTestPostgreSQLDatabaseName)

			pgDbWrapper := manual.NewDBforPostgreSQLDatabase(
				clients.NewPostgreSQLDatabasesClient(postgreSQLDatabaseClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pgDbWrapper.Scopes()[0]

			pgDbAdapter := sources.WrapperToAdapter(pgDbWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(postgreSQLServerName, integrationTestPostgreSQLDatabaseName)
			sdpItem, qErr := pgDbAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (PostgreSQL Flexible Server should be linked)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasPostgreSQLServerLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.DBforPostgreSQLFlexibleServer.String() {
					hasPostgreSQLServerLink = true
					if liq.GetQuery().GetQuery() != postgreSQLServerName {
						t.Errorf("Expected linked query to PostgreSQL server %s, got %s", postgreSQLServerName, liq.GetQuery().GetQuery())
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
						t.Error("Expected BlastPropagation to be set for PostgreSQL server link")
					} else {
						if !bp.GetIn() {
							t.Error("Expected BlastPropagation.In to be true for PostgreSQL server link")
						}
						if bp.GetOut() {
							t.Error("Expected BlastPropagation.Out to be false for PostgreSQL server link")
						}
					}
					break
				}
			}

			if !hasPostgreSQLServerLink {
				t.Error("Expected linked query to PostgreSQL Flexible Server, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for PostgreSQL database %s", len(linkedQueries), integrationTestPostgreSQLDatabaseName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete PostgreSQL database
		err := deletePostgreSQLDatabase(ctx, postgreSQLDatabaseClient, integrationTestResourceGroup, postgreSQLServerName, integrationTestPostgreSQLDatabaseName)
		if err != nil {
			t.Fatalf("Failed to delete PostgreSQL database: %v", err)
		}

		// Delete PostgreSQL Flexible Server
		err = deletePostgreSQLFlexibleServer(ctx, postgreSQLServerClient, integrationTestResourceGroup, postgreSQLServerName)
		if err != nil {
			t.Fatalf("Failed to delete PostgreSQL Flexible Server: %v", err)
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

// generatePostgreSQLServerName generates a unique PostgreSQL Flexible Server name
// PostgreSQL server names must be globally unique, 1-63 characters, lowercase letters, numbers, and hyphens
func generatePostgreSQLServerName(baseName string) string {
	// Ensure base name is lowercase and valid
	baseName = strings.ToLower(baseName)
	// Remove any invalid characters (only alphanumeric and hyphens allowed)
	baseName = strings.ReplaceAll(baseName, "_", "-")
	// Remove any invalid characters
	baseName = strings.ReplaceAll(baseName, " ", "-")
	// Add random suffix to ensure uniqueness
	suffix := rand.Intn(10000)
	return fmt.Sprintf("%s-%d", baseName, suffix)
}

// createPostgreSQLFlexibleServer creates an Azure PostgreSQL Flexible Server (idempotent)
func createPostgreSQLFlexibleServer(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, resourceGroupName, serverName, location string) error {
	// Check if PostgreSQL server already exists
	_, err := client.Get(ctx, resourceGroupName, serverName, nil)
	if err == nil {
		log.Printf("PostgreSQL Flexible Server %s already exists, skipping creation", serverName)
		return nil
	}

	// Get administrator credentials from environment variables
	// Note: PostgreSQL Flexible Servers require administrator login credentials
	// Credentials are read from environment variables to avoid committing secrets to source control
	adminLogin := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD")

	if adminLogin == "" || adminPassword == "" {
		return fmt.Errorf("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN and AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD environment variables must be set for integration tests")
	}

	// Create the PostgreSQL Flexible Server
	// Using Burstable tier for cost-effective testing
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, serverName, armpostgresqlflexibleservers.Server{
		Location: ptr.To(location),
		Properties: &armpostgresqlflexibleservers.ServerProperties{
			AdministratorLogin:         ptr.To(adminLogin),
			AdministratorLoginPassword: ptr.To(adminPassword),
			Version:                    ptr.To(armpostgresqlflexibleservers.PostgresMajorVersion("14")),
			Storage:                    &armpostgresqlflexibleservers.Storage{StorageSizeGB: ptr.To[int32](32)},
			Backup:                     &armpostgresqlflexibleservers.Backup{BackupRetentionDays: ptr.To[int32](7), GeoRedundantBackup: ptr.To(armpostgresqlflexibleservers.GeographicallyRedundantBackupDisabled)},
			Network:                    &armpostgresqlflexibleservers.Network{PublicNetworkAccess: ptr.To(armpostgresqlflexibleservers.ServerPublicNetworkAccessStateEnabled)},
			HighAvailability:           nil, // High availability disabled by not setting it
		},
		SKU: &armpostgresqlflexibleservers.SKU{
			Name: ptr.To("Standard_B1ms"), // Burstable tier, 1 vCore, 2GB RAM
			Tier: ptr.To(armpostgresqlflexibleservers.SKUTierBurstable),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("dbforpostgresql-database"),
		},
	}, nil)
	if err != nil {
		// Check if PostgreSQL server already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("PostgreSQL Flexible Server %s already exists, skipping creation", serverName)
			return nil
		}
		return fmt.Errorf("failed to begin creating PostgreSQL Flexible Server: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL Flexible Server: %w", err)
	}

	// Verify the PostgreSQL server was created successfully
	if resp.Properties == nil {
		return fmt.Errorf("PostgreSQL Flexible Server created but properties are nil")
	}

	log.Printf("PostgreSQL Flexible Server %s created successfully", serverName)
	return nil
}

// waitForPostgreSQLServerAvailable waits for a PostgreSQL Flexible Server to be fully available
func waitForPostgreSQLServerAvailable(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, resourceGroupName, serverName string) error {
	maxAttempts := 20
	pollInterval := 10 * time.Second

	log.Printf("Waiting for PostgreSQL Flexible Server %s to be available via API...", serverName)

	for attempt := range maxAttempts {
		resp, err := client.Get(ctx, resourceGroupName, serverName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("PostgreSQL Flexible Server %s not yet available (attempt %d/%d), waiting %v...", serverName, attempt+1, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking PostgreSQL Flexible Server availability: %w", err)
		}

		// Check if server is ready (State should be "Ready")
		if resp.Properties != nil && resp.Properties.State != nil {
			state := *resp.Properties.State
			if state == armpostgresqlflexibleservers.ServerStateReady {
				log.Printf("PostgreSQL Flexible Server %s is available with state: %s", serverName, state)
				return nil
			}
			if state == armpostgresqlflexibleservers.ServerStateDisabled || state == armpostgresqlflexibleservers.ServerStateDropping {
				return fmt.Errorf("PostgreSQL Flexible Server provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("PostgreSQL Flexible Server %s state: %s (attempt %d/%d), waiting...", serverName, state, attempt+1, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// PostgreSQL server exists but no state - consider it available
		log.Printf("PostgreSQL Flexible Server %s is available", serverName)
		return nil
	}

	return fmt.Errorf("timeout waiting for PostgreSQL Flexible Server %s to be available after %d attempts", serverName, maxAttempts)
}

// createPostgreSQLDatabase creates an Azure PostgreSQL Database (idempotent)
func createPostgreSQLDatabase(ctx context.Context, client *armpostgresqlflexibleservers.DatabasesClient, resourceGroupName, serverName, databaseName string) error {
	// Check if PostgreSQL database already exists
	_, err := client.Get(ctx, resourceGroupName, serverName, databaseName, nil)
	if err == nil {
		log.Printf("PostgreSQL database %s already exists, skipping creation", databaseName)
		return nil
	}

	// Create the PostgreSQL database
	poller, err := client.BeginCreate(ctx, resourceGroupName, serverName, databaseName, armpostgresqlflexibleservers.Database{
		Properties: &armpostgresqlflexibleservers.DatabaseProperties{
			Charset:   ptr.To("UTF8"),
			Collation: ptr.To("en_US.utf8"),
		},
	}, nil)
	if err != nil {
		// Check if PostgreSQL database already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("PostgreSQL database %s already exists, skipping creation", databaseName)
			return nil
		}
		return fmt.Errorf("failed to begin creating PostgreSQL database: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL database: %w", err)
	}

	log.Printf("PostgreSQL database %s created successfully", databaseName)
	return nil
}

// waitForPostgreSQLDatabaseAvailable waits for a PostgreSQL Database to be fully available
func waitForPostgreSQLDatabaseAvailable(ctx context.Context, client *armpostgresqlflexibleservers.DatabasesClient, resourceGroupName, serverName, databaseName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for PostgreSQL database %s to be available via API...", databaseName)

	for attempt := range maxAttempts {
		_, err := client.Get(ctx, resourceGroupName, serverName, databaseName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("PostgreSQL database %s not yet available (attempt %d/%d), waiting %v...", databaseName, attempt+1, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking PostgreSQL database availability: %w", err)
		}

		// If we can get the database, it's available
		log.Printf("PostgreSQL database %s is available", databaseName)
		return nil
	}

	return fmt.Errorf("timeout waiting for PostgreSQL database %s to be available after %d attempts", databaseName, maxAttempts)
}

// deletePostgreSQLDatabase deletes an Azure PostgreSQL Database
func deletePostgreSQLDatabase(ctx context.Context, client *armpostgresqlflexibleservers.DatabasesClient, resourceGroupName, serverName, databaseName string) error {
	// Check if PostgreSQL database exists
	_, err := client.Get(ctx, resourceGroupName, serverName, databaseName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("PostgreSQL database %s does not exist, skipping deletion", databaseName)
			return nil
		}
		return fmt.Errorf("error checking PostgreSQL database existence: %w", err)
	}

	// Delete the PostgreSQL database
	poller, err := client.BeginDelete(ctx, resourceGroupName, serverName, databaseName, nil)
	if err != nil {
		return fmt.Errorf("failed to begin deleting PostgreSQL database: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete PostgreSQL database: %w", err)
	}

	log.Printf("PostgreSQL database %s deleted successfully", databaseName)
	return nil
}

// deletePostgreSQLFlexibleServer deletes an Azure PostgreSQL Flexible Server
func deletePostgreSQLFlexibleServer(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, resourceGroupName, serverName string) error {
	// Check if PostgreSQL server exists
	_, err := client.Get(ctx, resourceGroupName, serverName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("PostgreSQL Flexible Server %s does not exist, skipping deletion", serverName)
			return nil
		}
		return fmt.Errorf("error checking PostgreSQL Flexible Server existence: %w", err)
	}

	// Delete the PostgreSQL Flexible Server
	poller, err := client.BeginDelete(ctx, resourceGroupName, serverName, nil)
	if err != nil {
		return fmt.Errorf("failed to begin deleting PostgreSQL Flexible Server: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete PostgreSQL Flexible Server: %w", err)
	}

	log.Printf("PostgreSQL Flexible Server %s deleted successfully", serverName)
	return nil
}
