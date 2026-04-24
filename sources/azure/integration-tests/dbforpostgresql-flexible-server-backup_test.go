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
	integrationTestPGBackupServerName = "ovm-integ-test-pg-backup"
	integrationTestPGBackupName       = "ovm-integ-test-backup"
)

func TestDBforPostgreSQLFlexibleServerBackupIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	postgreSQLServerClient, err := armpostgresqlflexibleservers.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Flexible Servers client: %v", err)
	}

	backupsClient, err := armpostgresqlflexibleservers.NewBackupsAutomaticAndOnDemandClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Backups client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	pgServerName := generatePostgreSQLServerName(integrationTestPGBackupServerName)

	setupCompleted := false

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createPostgreSQLFlexibleServerForBackup(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL Flexible Server: %v", err)
		}

		err = waitForPostgreSQLServerAvailable(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL server to be available: %v", err)
		}

		err = createOnDemandBackup(ctx, backupsClient, integrationTestResourceGroup, pgServerName, integrationTestPGBackupName)
		if err != nil {
			t.Fatalf("Failed to create on-demand backup: %v", err)
		}

		err = waitForBackupAvailable(ctx, backupsClient, integrationTestResourceGroup, pgServerName, integrationTestPGBackupName)
		if err != nil {
			t.Fatalf("Failed waiting for backup to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetPostgreSQLFlexibleServerBackup", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(
				clients.NewDBforPostgreSQLFlexibleServerBackupClient(backupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, integrationTestPGBackupName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerBackup.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerBackup, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(pgServerName, integrationTestPGBackupName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved backup %s", integrationTestPGBackupName)
		})

		t.Run("SearchPostgreSQLFlexibleServerBackups", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(
				clients.NewDBforPostgreSQLFlexibleServerBackupClient(backupsClient),
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
				t.Fatalf("Failed to search backups: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one backup, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					expectedValue := shared.CompositeLookupKey(pgServerName, integrationTestPGBackupName)
					if v == expectedValue {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find backup %s in the search results", integrationTestPGBackupName)
			}

			log.Printf("Found %d backups in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(
				clients.NewDBforPostgreSQLFlexibleServerBackupClient(backupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, integrationTestPGBackupName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
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

			log.Printf("Verified %d linked item queries for backup %s", len(linkedQueries), integrationTestPGBackupName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(
				clients.NewDBforPostgreSQLFlexibleServerBackupClient(backupsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, integrationTestPGBackupName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerBackup.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerBackup, sdpItem.GetType())
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

		err := deleteOnDemandBackup(ctx, backupsClient, integrationTestResourceGroup, pgServerName, integrationTestPGBackupName)
		if err != nil {
			log.Printf("Warning: failed to delete backup (may have been auto-cleaned): %v", err)
		}

		err = deletePostgreSQLFlexibleServer(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName)
		if err != nil {
			t.Fatalf("Failed to delete PostgreSQL Flexible Server: %v", err)
		}
	})
}

// createPostgreSQLFlexibleServerForBackup creates a GeneralPurpose-tier server
// because Azure does not allow on-demand backups on Burstable-tier servers.
func createPostgreSQLFlexibleServerForBackup(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, resourceGroupName, serverName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, serverName, nil)
	if err == nil {
		log.Printf("PostgreSQL Flexible Server %s already exists, skipping creation", serverName)
		return nil
	}

	adminLogin := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD")
	if adminLogin == "" || adminPassword == "" {
		return fmt.Errorf("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN and AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD must be set")
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
		},
		SKU: &armpostgresqlflexibleservers.SKU{
			Name: new("Standard_D2s_v3"),
			Tier: new(armpostgresqlflexibleservers.SKUTierGeneralPurpose),
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("dbforpostgresql-backup"),
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

	_, err = poller.PollUntilDone(opCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL Flexible Server: %w", err)
	}

	log.Printf("PostgreSQL Flexible Server %s (GeneralPurpose) created successfully", serverName)
	return nil
}

func createOnDemandBackup(ctx context.Context, client *armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClient, resourceGroupName, serverName, backupName string) error {
	_, err := client.Get(ctx, resourceGroupName, serverName, backupName, nil)
	if err == nil {
		log.Printf("Backup %s already exists, skipping creation", backupName)
		return nil
	}

	poller, err := client.BeginCreate(ctx, resourceGroupName, serverName, backupName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Backup %s already exists (conflict), skipping", backupName)
			return nil
		}
		return fmt.Errorf("failed to begin creating backup: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	log.Printf("Backup %s created successfully", backupName)
	return nil
}

func waitForBackupAvailable(ctx context.Context, client *armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClient, resourceGroupName, serverName, backupName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := client.Get(ctx, resourceGroupName, serverName, backupName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Backup %s not yet available (attempt %d/%d), waiting...", backupName, attempt, maxAttempts)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking backup availability: %w", err)
		}

		log.Printf("Backup %s is available", backupName)
		return nil
	}

	return fmt.Errorf("timeout waiting for backup %s to be available", backupName)
}

func deleteOnDemandBackup(ctx context.Context, client *armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClient, resourceGroupName, serverName, backupName string) error {
	_, err := client.Get(ctx, resourceGroupName, serverName, backupName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Backup %s does not exist, skipping deletion", backupName)
			return nil
		}
		return fmt.Errorf("error checking backup existence: %w", err)
	}

	poller, err := client.BeginDelete(ctx, resourceGroupName, serverName, backupName, nil)
	if err != nil {
		return fmt.Errorf("failed to begin deleting backup: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	log.Printf("Backup %s deleted successfully", backupName)
	return nil
}
