package integrationtests

import (
	"fmt"
	"os"
	"testing"

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
	integrationTestPGConfigServerName = "ovm-integ-test-pg-config"
)

func TestDBforPostgreSQLFlexibleServerConfigurationIntegration(t *testing.T) {
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

	configurationsClient, err := armpostgresqlflexibleservers.NewConfigurationsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL Configurations client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	pgServerName := generatePostgreSQLServerName(integrationTestPGConfigServerName)
	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createPostgreSQLFlexibleServer(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create PostgreSQL Flexible Server: %v", err)
		}

		err = waitForPostgreSQLServerAvailable(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName)
		if err != nil {
			t.Fatalf("Failed waiting for PostgreSQL server to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetPostgreSQLFlexibleServerConfiguration", func(t *testing.T) {
			ctx := t.Context()

			pager := configurationsClient.NewListByServerPager(integrationTestResourceGroup, pgServerName, nil)
			var configName string
			if pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					t.Fatalf("Failed to list configurations: %v", err)
				}
				if len(page.Value) > 0 && page.Value[0].Name != nil {
					configName = *page.Value[0].Name
				}
			}

			if configName == "" {
				t.Skip("No configurations found on server")
			}

			log.Printf("Testing with configuration: %s", configName)

			wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(
				clients.NewPostgreSQLConfigurationsClient(configurationsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, configName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerConfiguration.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerConfiguration, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(pgServerName, configName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved configuration %s", configName)
		})

		t.Run("SearchPostgreSQLFlexibleServerConfigurations", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(
				clients.NewPostgreSQLConfigurationsClient(configurationsClient),
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
				t.Fatalf("Failed to search configurations: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one configuration, got %d", len(sdpItems))
			}

			for _, item := range sdpItems {
				if err := item.Validate(); err != nil {
					t.Fatalf("Item validation failed: %v", err)
				}

				if item.GetType() != azureshared.DBforPostgreSQLFlexibleServerConfiguration.String() {
					t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerConfiguration, item.GetType())
				}
			}

			log.Printf("Found %d configurations in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			pager := configurationsClient.NewListByServerPager(integrationTestResourceGroup, pgServerName, nil)
			var configName string
			if pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					t.Fatalf("Failed to list configurations: %v", err)
				}
				if len(page.Value) > 0 && page.Value[0].Name != nil {
					configName = *page.Value[0].Name
				}
			}

			if configName == "" {
				t.Skip("No configurations found on server")
			}

			wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(
				clients.NewPostgreSQLConfigurationsClient(configurationsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, configName)
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

			log.Printf("Verified %d linked item queries for configuration %s", len(linkedQueries), configName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			pager := configurationsClient.NewListByServerPager(integrationTestResourceGroup, pgServerName, nil)
			var configName string
			if pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					t.Fatalf("Failed to list configurations: %v", err)
				}
				if len(page.Value) > 0 && page.Value[0].Name != nil {
					configName = *page.Value[0].Name
				}
			}

			if configName == "" {
				t.Skip("No configurations found on server")
			}

			wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(
				clients.NewPostgreSQLConfigurationsClient(configurationsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(pgServerName, configName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerConfiguration.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerConfiguration, sdpItem.GetType())
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

		err := deletePostgreSQLFlexibleServer(ctx, postgreSQLServerClient, integrationTestResourceGroup, pgServerName)
		if err != nil {
			t.Fatalf("Failed to delete PostgreSQL Flexible Server: %v", err)
		}
	})
}
