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
	integrationTestPGVirtualEndpointServerName = "ovm-integ-test-pg-vep"
	integrationTestPGVirtualEndpointName       = "ovm-integ-test-vep"
)

func TestDBforPostgreSQLFlexibleServerVirtualEndpointIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	adminLogin := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN")
	adminPassword := os.Getenv("AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD")
	if adminLogin == "" || adminPassword == "" {
		t.Skip("AZURE_POSTGRESQL_SERVER_ADMIN_LOGIN and AZURE_POSTGRESQL_SERVER_ADMIN_PASSWORD must be set for PostgreSQL tests")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	postgreSQLServerClient, err := armpostgresqlflexibleservers.NewServersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Flexible Servers client: %v", err)
	}

	virtualEndpointsClient, err := armpostgresqlflexibleservers.NewVirtualEndpointsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Virtual Endpoints client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	pgServerName := generatePostgreSQLServerName(integrationTestPGVirtualEndpointServerName)

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createPostgreSQLFlexibleServerForVirtualEndpoint(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL Flexible Server: %v", err)
		}

		err = waitForPostgreSQLServerAvailable(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL server to be available: %v", err)
		}

		err = createVirtualEndpoint(ctx, virtualEndpointsClient, integrationTestResourceGroup, pgServerName, integrationTestPGVirtualEndpointName)
		if err != nil {
			t.Fatalf("Failed to create virtual endpoint: %v", err)
		}

		err = waitForVirtualEndpointAvailable(ctx, virtualEndpointsClient, integrationTestResourceGroup, pgServerName, integrationTestPGVirtualEndpointName)
		if err != nil {
			t.Fatalf("Failed waiting for virtual endpoint to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetVirtualEndpoint", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(
				clients.NewDBforPostgreSQLFlexibleServerVirtualEndpointClient(virtualEndpointsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, integrationTestPGVirtualEndpointName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(pgServerName, integrationTestPGVirtualEndpointName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved virtual endpoint %s", integrationTestPGVirtualEndpointName)
		})

		t.Run("SearchVirtualEndpoints", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(
				clients.NewDBforPostgreSQLFlexibleServerVirtualEndpointClient(virtualEndpointsClient),
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
				t.Fatalf("Failed to search virtual endpoints: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one virtual endpoint, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					expectedValue := shared.CompositeLookupKey(pgServerName, integrationTestPGVirtualEndpointName)
					if v == expectedValue {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find virtual endpoint %s in the search results", integrationTestPGVirtualEndpointName)
			}

			log.Printf("Found %d virtual endpoints in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(
				clients.NewDBforPostgreSQLFlexibleServerVirtualEndpointClient(virtualEndpointsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, integrationTestPGVirtualEndpointName)
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
				q := liq.GetQuery()
				if q.GetType() == azureshared.DBforPostgreSQLFlexibleServer.String() {
					hasServerLink = true
					if q.GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected linked query method GET, got %s", q.GetMethod())
					}
					if q.GetScope() != scope {
						t.Errorf("Expected linked query scope %s, got %s", scope, q.GetScope())
					}
					break
				}
			}

			if !hasServerLink {
				t.Error("Expected linked query to PostgreSQL Flexible Server, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for virtual endpoint %s", len(linkedQueries), integrationTestPGVirtualEndpointName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(
				clients.NewDBforPostgreSQLFlexibleServerVirtualEndpointClient(virtualEndpointsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, integrationTestPGVirtualEndpointName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint, sdpItem.GetType())
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

		err := deleteVirtualEndpoint(ctx, virtualEndpointsClient, integrationTestResourceGroup, pgServerName, integrationTestPGVirtualEndpointName)
		if err != nil {
			log.Printf("Warning: failed to delete virtual endpoint: %v", err)
		}

		err = deletePostgreSQLFlexibleServer(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName)
		if err != nil {
			t.Fatalf("Failed to delete PostgreSQL Flexible Server: %v", err)
		}
	})
}

func createPostgreSQLFlexibleServerForVirtualEndpoint(ctx context.Context, client *armpostgresqlflexibleservers.ServersClient, resourceGroupName, serverName, location string) error {
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
			"test":    new("dbforpostgresql-virtual-endpoint"),
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

func createVirtualEndpoint(ctx context.Context, client *armpostgresqlflexibleservers.VirtualEndpointsClient, resourceGroupName, serverName, virtualEndpointName string) error {
	_, err := client.Get(ctx, resourceGroupName, serverName, virtualEndpointName, nil)
	if err == nil {
		log.Printf("Virtual endpoint %s already exists, skipping creation", virtualEndpointName)
		return nil
	}

	opCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	endpointType := armpostgresqlflexibleservers.VirtualEndpointTypeReadWrite
	poller, err := client.BeginCreate(opCtx, resourceGroupName, serverName, virtualEndpointName, armpostgresqlflexibleservers.VirtualEndpoint{
		Properties: &armpostgresqlflexibleservers.VirtualEndpointResourceProperties{
			EndpointType: &endpointType,
			Members:      []*string{new(serverName)},
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroupName, serverName, virtualEndpointName, nil); getErr == nil {
				log.Printf("Virtual endpoint %s already exists (conflict), skipping", virtualEndpointName)
				return nil
			}
			return fmt.Errorf("virtual endpoint %s conflict but not retrievable: %w", virtualEndpointName, err)
		}
		return fmt.Errorf("failed to begin creating virtual endpoint: %w", err)
	}

	_, err = poller.PollUntilDone(opCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to create virtual endpoint: %w", err)
	}

	log.Printf("Virtual endpoint %s created successfully", virtualEndpointName)
	return nil
}

func waitForVirtualEndpointAvailable(ctx context.Context, client *armpostgresqlflexibleservers.VirtualEndpointsClient, resourceGroupName, serverName, virtualEndpointName string) error {
	maxAttempts := 30
	pollInterval := 10 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := client.Get(ctx, resourceGroupName, serverName, virtualEndpointName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Virtual endpoint %s not yet available (attempt %d/%d), waiting...", virtualEndpointName, attempt, maxAttempts)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking virtual endpoint availability: %w", err)
		}

		log.Printf("Virtual endpoint %s is available", virtualEndpointName)
		return nil
	}

	return fmt.Errorf("timeout waiting for virtual endpoint %s to be available", virtualEndpointName)
}

func deleteVirtualEndpoint(ctx context.Context, client *armpostgresqlflexibleservers.VirtualEndpointsClient, resourceGroupName, serverName, virtualEndpointName string) error {
	_, err := client.Get(ctx, resourceGroupName, serverName, virtualEndpointName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Virtual endpoint %s does not exist, skipping deletion", virtualEndpointName)
			return nil
		}
		return fmt.Errorf("error checking virtual endpoint existence: %w", err)
	}

	poller, err := client.BeginDelete(ctx, resourceGroupName, serverName, virtualEndpointName, nil)
	if err != nil {
		return fmt.Errorf("failed to begin deleting virtual endpoint: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual endpoint: %w", err)
	}

	log.Printf("Virtual endpoint %s deleted successfully", virtualEndpointName)
	return nil
}
