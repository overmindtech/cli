package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
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
	integrationTestPGAdminServerName = "ovm-integ-test-pg-admin"
)

func TestDBforPostgreSQLFlexibleServerAdministratorIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	adminLogin := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD")
	if adminLogin == "" || adminPassword == "" {
		t.Skip("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN and AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD must be set for PostgreSQL tests")
	}

	entraAdminObjectID := os.Getenv("AZURE_POSTGRESQL_ENTRA_ADMIN_OBJECT_ID")
	entraAdminPrincipalName := os.Getenv("AZURE_POSTGRESQL_ENTRA_ADMIN_PRINCIPAL_NAME")
	entraAdminTenantID := os.Getenv("AZURE_POSTGRESQL_ENTRA_ADMIN_TENANT_ID")

	if entraAdminObjectID == "" || entraAdminPrincipalName == "" || entraAdminTenantID == "" {
		t.Skip("AZURE_POSTGRESQL_ENTRA_ADMIN_OBJECT_ID, AZURE_POSTGRESQL_ENTRA_ADMIN_PRINCIPAL_NAME, and AZURE_POSTGRESQL_ENTRA_ADMIN_TENANT_ID must be set for PostgreSQL Administrator tests")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	postgreSQLServerClient, err := armpostgresqlflexibleservers.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Flexible Servers client: %v", err)
	}

	administratorsClient, err := armpostgresqlflexibleservers.NewAdministratorsMicrosoftEntraClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Administrators client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	pgServerName := generatePostgreSQLServerName(integrationTestPGAdminServerName)
	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createPostgreSQLFlexibleServerWithMicrosoftEntraAuth(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL Flexible Server: %v", err)
		}

		err = waitForPostgreSQLServerAvailable(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL server to be available: %v", err)
		}

		err = createPostgreSQLAdministrator(ctx, administratorsClient, integrationTestResourceGroup, pgServerName, entraAdminObjectID, entraAdminPrincipalName, entraAdminTenantID)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL Administrator: %v", err)
		}

		err = waitForPostgreSQLAdministratorAvailable(ctx, administratorsClient, integrationTestResourceGroup, pgServerName, entraAdminObjectID)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL Administrator to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetPostgreSQLFlexibleServerAdministrator", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(
				clients.NewDBforPostgreSQLFlexibleServerAdministratorClient(administratorsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, entraAdminObjectID)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerAdministrator.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerAdministrator, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(pgServerName, entraAdminObjectID)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved administrator %s", entraAdminObjectID)
		})

		t.Run("SearchPostgreSQLFlexibleServerAdministrators", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(
				clients.NewDBforPostgreSQLFlexibleServerAdministratorClient(administratorsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, pgServerName, true)
			if err != nil {
				t.Fatalf("Failed to search administrators: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one administrator, got %d", len(sdpItems))
			}

			var foundAdmin bool
			for _, item := range sdpItems {
				if err := item.Validate(); err != nil {
					t.Fatalf("Item validation failed: %v", err)
				}

				if item.GetType() != azureshared.DBforPostgreSQLFlexibleServerAdministrator.String() {
					t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerAdministrator, item.GetType())
				}

				expectedUniqueValue := shared.CompositeLookupKey(pgServerName, entraAdminObjectID)
				if item.UniqueAttributeValue() == expectedUniqueValue {
					foundAdmin = true
				}
			}

			if !foundAdmin {
				t.Errorf("Expected to find administrator %s in search results", entraAdminObjectID)
			}

			log.Printf("Found %d administrators in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(
				clients.NewDBforPostgreSQLFlexibleServerAdministratorClient(administratorsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, entraAdminObjectID)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == "" {
					t.Error("Expected linked query Type to be non-empty")
				}
				if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET && liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Expected linked query Method to be GET or SEARCH, got %s", liq.GetQuery().GetMethod())
				}
				if liq.GetQuery().GetQuery() == "" {
					t.Error("Expected linked query Query to be non-empty")
				}
				if liq.GetQuery().GetScope() == "" {
					t.Error("Expected linked query Scope to be non-empty")
				}
			}

			var hasServerLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.DBforPostgreSQLFlexibleServer.String() {
					hasServerLink = true
					if liq.GetQuery().GetQuery() != pgServerName {
						t.Errorf("Expected linked query to server %s, got %s", pgServerName, liq.GetQuery().GetQuery())
					}
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected linked query method GET, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetScope() != scope {
						t.Errorf("Expected linked query scope %s, got %s", scope, liq.GetQuery().GetScope())
					}
					break
				}
			}

			if !hasServerLink {
				t.Error("Expected linked query to PostgreSQL Flexible Server, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for administrator %s", len(linkedQueries), entraAdminObjectID)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(
				clients.NewDBforPostgreSQLFlexibleServerAdministratorClient(administratorsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, entraAdminObjectID)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerAdministrator.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerAdministrator, sdpItem.GetType())
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

		err := deletePostgreSQLAdministrator(ctx, administratorsClient, integrationTestResourceGroup, pgServerName, entraAdminObjectID)
		if err != nil {
			log.Printf("Warning: Failed to delete PostgreSQL Administrator: %v", err)
		}

		err = deletePostgreSQLFlexibleServer(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName)
		if err != nil {
			t.Fatalf("Failed to delete PostgreSQL Flexible Server: %v", err)
		}
	})
}

// createPostgreSQLFlexibleServerWithMicrosoftEntraAuth creates a PostgreSQL Flexible Server with Microsoft Entra authentication enabled
func createPostgreSQLFlexibleServerWithMicrosoftEntraAuth(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, resourceGroupName, serverName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, serverName, nil)
	if err == nil {
		log.Printf("PostgreSQL Flexible Server %s already exists, skipping creation", serverName)
		return nil
	}

	adminLogin := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD")

	if adminLogin == "" || adminPassword == "" {
		return fmt.Errorf("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN and AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD environment variables must be set for integration tests")
	}

	opCtx, cancel := context.WithTimeout(ctx, 25*time.Minute)
	defer cancel()

	poller, err := client.BeginCreateOrUpdate(opCtx, resourceGroupName, serverName, armpostgresqlflexibleservers.Server{
		Location: new(location),
		Properties: &armpostgresqlflexibleservers.ServerProperties{
			AdministratorLogin:         new(adminLogin),
			AdministratorLoginPassword: new(adminPassword),
			Version:                    new(armpostgresqlflexibleservers.PostgresMajorVersion("14")),
			Storage:                    &armpostgresqlflexibleservers.Storage{StorageSizeGB: new(int32(32))},
			Backup:                     &armpostgresqlflexibleservers.Backup{BackupRetentionDays: new(int32(7)), GeoRedundantBackup: new(armpostgresqlflexibleservers.GeographicallyRedundantBackupDisabled)},
			Network:                    &armpostgresqlflexibleservers.Network{PublicNetworkAccess: new(armpostgresqlflexibleservers.ServerPublicNetworkAccessStateEnabled)},
			HighAvailability:           nil,
			AuthConfig: &armpostgresqlflexibleservers.AuthConfig{
				ActiveDirectoryAuth: new(armpostgresqlflexibleservers.MicrosoftEntraAuthEnabled),
				PasswordAuth:        new(armpostgresqlflexibleservers.PasswordBasedAuthEnabled),
			},
		},
		SKU: &armpostgresqlflexibleservers.SKU{
			Name: new("Standard_B1ms"),
			Tier: new(armpostgresqlflexibleservers.SKUTierBurstable),
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("dbforpostgresql-administrator"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("PostgreSQL Flexible Server %s already exists, skipping creation", serverName)
			return nil
		}
		return fmt.Errorf("failed to begin creating PostgreSQL Flexible Server: %w", err)
	}

	resp, err := poller.PollUntilDone(opCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL Flexible Server: %w", err)
	}

	if resp.Properties == nil {
		return fmt.Errorf("PostgreSQL Flexible Server created but properties are nil")
	}

	log.Printf("PostgreSQL Flexible Server %s created successfully with Microsoft Entra authentication enabled", serverName)
	return nil
}

// createPostgreSQLAdministrator creates a Microsoft Entra administrator for a PostgreSQL Flexible Server
func createPostgreSQLAdministrator(ctx context.Context, client *armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClient, resourceGroupName, serverName, objectID, principalName, tenantID string) error {
	_, err := client.Get(ctx, resourceGroupName, serverName, objectID, nil)
	if err == nil {
		log.Printf("PostgreSQL Administrator %s already exists on server %s, skipping creation", objectID, serverName)
		return nil
	}

	opCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	principalType := armpostgresqlflexibleservers.PrincipalTypeServicePrincipal

	poller, err := client.BeginCreateOrUpdate(opCtx, resourceGroupName, serverName, objectID, armpostgresqlflexibleservers.AdministratorMicrosoftEntraAdd{
		Properties: &armpostgresqlflexibleservers.AdministratorMicrosoftEntraPropertiesForAdd{
			PrincipalName: new(principalName),
			PrincipalType: &principalType,
			TenantID:      new(tenantID),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("PostgreSQL Administrator %s already exists on server %s, skipping creation", objectID, serverName)
			return nil
		}
		return fmt.Errorf("failed to begin creating PostgreSQL Administrator: %w", err)
	}

	_, err = poller.PollUntilDone(opCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL Administrator: %w", err)
	}

	log.Printf("PostgreSQL Administrator %s created successfully on server %s", objectID, serverName)
	return nil
}

// waitForPostgreSQLAdministratorAvailable waits for a PostgreSQL Administrator to be fully available
func waitForPostgreSQLAdministratorAvailable(ctx context.Context, client *armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClient, resourceGroupName, serverName, objectID string) error {
	maxAttempts := 30
	pollInterval := 10 * time.Second

	log.Printf("Waiting for PostgreSQL Administrator %s to be available on server %s...", objectID, serverName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := client.Get(ctx, resourceGroupName, serverName, objectID, nil)
		if err == nil {
			log.Printf("PostgreSQL Administrator %s is available on server %s", objectID, serverName)
			return nil
		}

		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("PostgreSQL Administrator %s not yet available (attempt %d/%d), waiting %v...", objectID, attempt, maxAttempts, pollInterval)
			time.Sleep(pollInterval)
			continue
		}

		return fmt.Errorf("error checking PostgreSQL Administrator availability: %w", err)
	}

	return fmt.Errorf("timeout waiting for PostgreSQL Administrator %s to be available on server %s", objectID, serverName)
}

// deletePostgreSQLAdministrator deletes a Microsoft Entra administrator from a PostgreSQL Flexible Server
func deletePostgreSQLAdministrator(ctx context.Context, client *armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClient, resourceGroupName, serverName, objectID string) error {
	opCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	poller, err := client.BeginDelete(opCtx, resourceGroupName, serverName, objectID, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("PostgreSQL Administrator %s already deleted or does not exist on server %s", objectID, serverName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting PostgreSQL Administrator: %w", err)
	}

	_, err = poller.PollUntilDone(opCtx, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("PostgreSQL Administrator %s already deleted", objectID)
			return nil
		}
		return fmt.Errorf("failed to delete PostgreSQL Administrator: %w", err)
	}

	log.Printf("PostgreSQL Administrator %s deleted successfully from server %s", objectID, serverName)
	return nil
}
