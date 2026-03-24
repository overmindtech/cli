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
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestPgServerName  = "ovm-integ-test-pg-server"
	integrationTestPgReplicaName = "ovm-integ-test-pg-replica"
)

func TestDBforPostgreSQLFlexibleServerReplicaIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	serversClient, err := armpostgresqlflexibleservers.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Flexible Servers client: %v", err)
	}

	replicasClient, err := armpostgresqlflexibleservers.NewReplicasClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Replicas client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Minute)
		defer cancel()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createPostgreSQLServerForReplica(ctx, serversClient, subscriptionID, integrationTestResourceGroup, integrationTestPgServerName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL flexible server: %v", err)
		}

		err = waitForPostgreSQLServerReady(ctx, serversClient, integrationTestResourceGroup, integrationTestPgServerName)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL server to be ready: %v", err)
		}

		err = createPostgreSQLReplica(ctx, serversClient, subscriptionID, integrationTestResourceGroup, integrationTestPgServerName, integrationTestPgReplicaName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL replica: %v", err)
		}

		err = waitForPostgreSQLServerReady(ctx, serversClient, integrationTestResourceGroup, integrationTestPgReplicaName)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL replica to be ready: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetReplica", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving replica %s under server %s", integrationTestPgReplicaName, integrationTestPgServerName)

			wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(
				clients.NewDBforPostgreSQLFlexibleServerReplicaClient(replicasClient, serversClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestPgServerName, integrationTestPgReplicaName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttr := shared.CompositeLookupKey(integrationTestPgServerName, integrationTestPgReplicaName)
			if uniqueAttrValue != expectedUniqueAttr {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttr, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved replica %s", integrationTestPgReplicaName)
		})

		t.Run("SearchReplicas", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching replicas under server %s", integrationTestPgServerName)

			wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(
				clients.NewDBforPostgreSQLFlexibleServerReplicaClient(replicasClient, serversClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestPgServerName, true)
			if err != nil {
				t.Fatalf("Failed to search replicas: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one replica, got %d", len(sdpItems))
			}

			var found bool
			expectedUniqueAttr := shared.CompositeLookupKey(integrationTestPgServerName, integrationTestPgReplicaName)
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedUniqueAttr {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find replica %s in search results", integrationTestPgReplicaName)
			}

			log.Printf("Found %d replicas in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for replica %s", integrationTestPgReplicaName)

			wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(
				clients.NewDBforPostgreSQLFlexibleServerReplicaClient(replicasClient, serversClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestPgServerName, integrationTestPgReplicaName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasSourceServerLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.DBforPostgreSQLFlexibleServer.String() {
					hasSourceServerLink = true
					if liq.GetQuery().GetQuery() != integrationTestPgServerName {
						t.Errorf("Expected linked query to source server %s, got %s", integrationTestPgServerName, liq.GetQuery().GetQuery())
					}
					break
				}
			}

			if !hasSourceServerLink {
				t.Error("Expected linked query to source server, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for replica %s", len(linkedQueries), integrationTestPgReplicaName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(
				clients.NewDBforPostgreSQLFlexibleServerReplicaClient(replicasClient, serversClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestPgServerName, integrationTestPgReplicaName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerReplica.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerReplica.String(), sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Item validation failed: %v", err)
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
		defer cancel()

		err := deletePostgreSQLServer(ctx, serversClient, integrationTestResourceGroup, integrationTestPgReplicaName)
		if err != nil {
			t.Logf("Warning: Failed to delete replica %s: %v", integrationTestPgReplicaName, err)
		}

		err = deletePostgreSQLServer(ctx, serversClient, integrationTestResourceGroup, integrationTestPgServerName)
		if err != nil {
			t.Logf("Warning: Failed to delete server %s: %v", integrationTestPgServerName, err)
		}
	})
}

func createPostgreSQLServerForReplica(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, subscriptionID, resourceGroupName, serverName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, serverName, nil)
	if err == nil {
		log.Printf("PostgreSQL server %s already exists, skipping creation", serverName)
		return nil
	}

	version := armpostgresqlflexibleservers.PostgresMajorVersionSixteen
	createMode := armpostgresqlflexibleservers.CreateModeDefault
	adminLogin := "ovmadmin"
	adminPassword := "TestPassword123!"
	skuName := "Standard_D2ds_v5"
	skuTier := armpostgresqlflexibleservers.SKUTierGeneralPurpose
	storageSizeGB := int32(32)

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, serverName, armpostgresqlflexibleservers.Server{
		Location: &location,
		SKU: &armpostgresqlflexibleservers.SKU{
			Name: &skuName,
			Tier: &skuTier,
		},
		Properties: &armpostgresqlflexibleservers.ServerProperties{
			Version:                    &version,
			CreateMode:                 &createMode,
			AdministratorLogin:         &adminLogin,
			AdministratorLoginPassword: &adminPassword,
			Storage: &armpostgresqlflexibleservers.Storage{
				StorageSizeGB: &storageSizeGB,
			},
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroupName, serverName, nil); getErr == nil {
				log.Printf("PostgreSQL server %s already exists (conflict), skipping creation", serverName)
				return nil
			}
			return fmt.Errorf("server %s conflict but not retrievable: %w", serverName, err)
		}
		return fmt.Errorf("failed to create PostgreSQL server: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL server: %w", err)
	}

	log.Printf("PostgreSQL server %s created successfully", serverName)
	return nil
}

func createPostgreSQLReplica(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, subscriptionID, resourceGroupName, primaryServerName, replicaName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, replicaName, nil)
	if err == nil {
		log.Printf("PostgreSQL replica %s already exists, skipping creation", replicaName)
		return nil
	}

	sourceServerID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DBforPostgreSQL/flexibleServers/%s",
		subscriptionID, resourceGroupName, primaryServerName)

	createMode := armpostgresqlflexibleservers.CreateModeReplica

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, replicaName, armpostgresqlflexibleservers.Server{
		Location: &location,
		Properties: &armpostgresqlflexibleservers.ServerProperties{
			CreateMode:             &createMode,
			SourceServerResourceID: &sourceServerID,
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroupName, replicaName, nil); getErr == nil {
				log.Printf("PostgreSQL replica %s already exists (conflict), skipping creation", replicaName)
				return nil
			}
			return fmt.Errorf("replica %s conflict but not retrievable: %w", replicaName, err)
		}
		return fmt.Errorf("failed to create PostgreSQL replica: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL replica: %w", err)
	}

	log.Printf("PostgreSQL replica %s created successfully", replicaName)
	return nil
}

func waitForPostgreSQLServerReady(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, resourceGroupName, serverName string) error {
	maxAttempts := 60
	pollInterval := 30 * time.Second
	maxNotFoundAttempts := 5
	notFoundCount := 0

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, serverName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				notFoundCount++
				if notFoundCount >= maxNotFoundAttempts {
					return fmt.Errorf("server %s not found after %d attempts", serverName, notFoundCount)
				}
				log.Printf("Server %s not found yet (attempt %d/%d), waiting...", serverName, attempt, maxAttempts)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking server: %w", err)
		}
		notFoundCount = 0

		if resp.Properties != nil && resp.Properties.State != nil {
			state := *resp.Properties.State
			log.Printf("Server %s state: %s (attempt %d/%d)", serverName, state, attempt, maxAttempts)
			if state == armpostgresqlflexibleservers.ServerStateReady {
				return nil
			}
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for server %s to be ready", serverName)
}

func deletePostgreSQLServer(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, resourceGroupName, serverName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, serverName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("PostgreSQL server %s not found, skipping deletion", serverName)
			return nil
		}
		return fmt.Errorf("failed to delete PostgreSQL server: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete PostgreSQL server: %w", err)
	}

	log.Printf("PostgreSQL server %s deleted successfully", serverName)
	return nil
}
