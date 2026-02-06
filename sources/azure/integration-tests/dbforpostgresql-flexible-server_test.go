package integrationtests

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

const (
	integrationTestPostgreSQLFlexibleServerName = "ovm-integ-test-pg-server"
)

func TestDBforPostgreSQLFlexibleServerIntegration(t *testing.T) {
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

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Generate unique PostgreSQL server name (must be globally unique, lowercase, no special chars)
	postgreSQLServerName := generatePostgreSQLServerName(integrationTestPostgreSQLFlexibleServerName)

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
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetPostgreSQLFlexibleServer", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving PostgreSQL Flexible Server %s in subscription %s, resource group %s",
				postgreSQLServerName, subscriptionID, integrationTestResourceGroup)

			pgServerWrapper := manual.NewDBforPostgreSQLFlexibleServer(
				clients.NewPostgreSQLFlexibleServersClient(postgreSQLServerClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pgServerWrapper.Scopes()[0]

			pgServerAdapter := sources.WrapperToAdapter(pgServerWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := pgServerAdapter.Get(ctx, scope, postgreSQLServerName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServer.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServer, sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			if uniqueAttrValue != postgreSQLServerName {
				t.Errorf("Expected unique attribute value %s, got %s", postgreSQLServerName, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved PostgreSQL Flexible Server %s", postgreSQLServerName)
		})

		t.Run("ListPostgreSQLFlexibleServers", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing PostgreSQL Flexible Servers in resource group %s", integrationTestResourceGroup)

			pgServerWrapper := manual.NewDBforPostgreSQLFlexibleServer(
				clients.NewPostgreSQLFlexibleServersClient(postgreSQLServerClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pgServerWrapper.Scopes()[0]

			pgServerAdapter := sources.WrapperToAdapter(pgServerWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports list
			listable, ok := pgServerAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list PostgreSQL Flexible Servers: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one PostgreSQL Flexible Server, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					if v == postgreSQLServerName {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find PostgreSQL Flexible Server %s in the list results", postgreSQLServerName)
			}

			log.Printf("Found %d PostgreSQL Flexible Servers in list results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for PostgreSQL Flexible Server %s", postgreSQLServerName)

			pgServerWrapper := manual.NewDBforPostgreSQLFlexibleServer(
				clients.NewPostgreSQLFlexibleServersClient(postgreSQLServerClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pgServerWrapper.Scopes()[0]

			pgServerAdapter := sources.WrapperToAdapter(pgServerWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := pgServerAdapter.Get(ctx, scope, postgreSQLServerName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (PostgreSQL Flexible Server has many child resources)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			// Verify expected child resource links exist
			expectedChildResources := map[string]bool{
				azureshared.DBforPostgreSQLDatabase.String():                    false,
				azureshared.DBforPostgreSQLFlexibleServerFirewallRule.String():  false,
				azureshared.DBforPostgreSQLFlexibleServerConfiguration.String(): false,
			}

			// These are conditional links (only present if server uses private networking or has FQDN)
			hasSubnetLink := false
			hasVirtualNetworkLink := false
			hasDNSLink := false

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				if expectedChildResources[linkedType] {
					t.Errorf("Found duplicate linked query for type %s", linkedType)
				}
				if _, exists := expectedChildResources[linkedType]; exists {
					expectedChildResources[linkedType] = true

					// Verify query method is SEARCH for child resources
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected linked query method SEARCH for %s, got %s", linkedType, liq.GetQuery().GetMethod())
					}

					// Verify query is the server name
					if liq.GetQuery().GetQuery() != postgreSQLServerName {
						t.Errorf("Expected linked query to use server name %s, got %s", postgreSQLServerName, liq.GetQuery().GetQuery())
					}

					// Verify scope matches
					if liq.GetQuery().GetScope() != scope {
						t.Errorf("Expected linked query scope %s, got %s", scope, liq.GetQuery().GetScope())
					}

					// Verify blast propagation is set
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Errorf("Expected BlastPropagation to be set for %s", linkedType)
					}
				}

				// Check for conditional links
				if linkedType == azureshared.NetworkSubnet.String() {
					hasSubnetLink = true
				}
				if linkedType == azureshared.NetworkVirtualNetwork.String() {
					hasVirtualNetworkLink = true
				}
				if linkedType == stdlib.NetworkDNS.String() {
					hasDNSLink = true
				}
			}

			// Check that all expected child resources are linked
			for resourceType, found := range expectedChildResources {
				if !found {
					t.Errorf("Expected linked query to %s, but didn't find one", resourceType)
				}
			}

			log.Printf("Verified %d linked item queries for PostgreSQL Flexible Server %s (hasSubnet: %v, hasVNet: %v, hasDNS: %v)",
				len(linkedQueries), postgreSQLServerName, hasSubnetLink, hasVirtualNetworkLink, hasDNSLink)
		})

		t.Run("VerifyChildResourceBlastPropagation", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying blast propagation for child resources of PostgreSQL Flexible Server %s", postgreSQLServerName)

			pgServerWrapper := manual.NewDBforPostgreSQLFlexibleServer(
				clients.NewPostgreSQLFlexibleServersClient(postgreSQLServerClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pgServerWrapper.Scopes()[0]

			pgServerAdapter := sources.WrapperToAdapter(pgServerWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := pgServerAdapter.Get(ctx, scope, postgreSQLServerName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()

			// Verify specific blast propagation patterns
			blastPropagationTests := map[string]struct {
				in  bool
				out bool
			}{
				// Child resources that depend on server (In: true, Out: false)
				azureshared.DBforPostgreSQLDatabase.String(): {in: true, out: false},
				// Child resources that affect server connectivity/configuration (In: true, Out: true)
				azureshared.DBforPostgreSQLFlexibleServerFirewallRule.String():  {in: true, out: true},
				azureshared.DBforPostgreSQLFlexibleServerConfiguration.String(): {in: true, out: true},
			}

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				if expected, ok := blastPropagationTests[linkedType]; ok {
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Errorf("Expected BlastPropagation to be set for %s", linkedType)
						continue
					}

					if bp.GetIn() != expected.in {
						t.Errorf("Expected BlastPropagation.In=%v for %s, got %v", expected.in, linkedType, bp.GetIn())
					}

					if bp.GetOut() != expected.out {
						t.Errorf("Expected BlastPropagation.Out=%v for %s, got %v", expected.out, linkedType, bp.GetOut())
					}
				}
			}

			log.Printf("Verified blast propagation for all child resources")
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete PostgreSQL Flexible Server
		err := deletePostgreSQLFlexibleServer(ctx, postgreSQLServerClient, integrationTestResourceGroup, postgreSQLServerName)
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
