package integrationtests

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

func TestNetworkVirtualNetworkIntegration(t *testing.T) {
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
	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Networks client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create virtual network
		err = createVirtualNetwork(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetVirtualNetwork", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving virtual network %s in subscription %s, resource group %s",
				integrationTestVNetName, subscriptionID, integrationTestResourceGroup)

			vnetWrapper := manual.NewNetworkVirtualNetwork(
				clients.NewVirtualNetworksClient(vnetClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vnetWrapper.Scopes()[0]

			vnetAdapter := sources.WrapperToAdapter(vnetWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := vnetAdapter.Get(ctx, scope, integrationTestVNetName, true)
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

			if uniqueAttrValue != integrationTestVNetName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestVNetName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved virtual network %s", integrationTestVNetName)
		})

		t.Run("ListVirtualNetworks", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing virtual networks in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			vnetWrapper := manual.NewNetworkVirtualNetwork(
				clients.NewVirtualNetworksClient(vnetClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vnetWrapper.Scopes()[0]

			vnetAdapter := sources.WrapperToAdapter(vnetWrapper, sdpcache.NewNoOpCache())
			listable, ok := vnetAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) == 0 {
				t.Fatalf("Expected at least 1 virtual network, got: %d", len(sdpItems))
			}

			// Find our test VNet
			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestVNetName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find virtual network %s in list, but didn't", integrationTestVNetName)
			}

			log.Printf("Successfully listed %d virtual networks", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			vnetWrapper := manual.NewNetworkVirtualNetwork(
				clients.NewVirtualNetworksClient(vnetClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := vnetWrapper.Scopes()[0]

			vnetAdapter := sources.WrapperToAdapter(vnetWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := vnetAdapter.Get(ctx, scope, integrationTestVNetName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify type
			if sdpItem.GetType() != azureshared.NetworkVirtualNetwork.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkVirtualNetwork.String(), sdpItem.GetType())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}

			// Verify scope
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify linked item queries
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected at least one linked item query, got: %d", len(linkedQueries))
			}

			// Verify subnet link
			var hasSubnetLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.NetworkSubnet.String() {
					hasSubnetLink = true
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected subnet link method to be SEARCH, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetQuery() != integrationTestVNetName {
						t.Errorf("Expected subnet link query to be %s, got %s", integrationTestVNetName, liq.GetQuery().GetQuery())
					}
					if liq.GetBlastPropagation().GetIn() != false {
						t.Error("Expected subnet blast propagation In=false, got true")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected subnet blast propagation Out=true, got false")
					}
					break
				}
			}
			if !hasSubnetLink {
				t.Error("Expected linked query to subnet, but didn't find one")
			}

			// Verify peering link
			var hasPeeringLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.NetworkVirtualNetworkPeering.String() {
					hasPeeringLink = true
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected peering link method to be SEARCH, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetQuery() != integrationTestVNetName {
						t.Errorf("Expected peering link query to be %s, got %s", integrationTestVNetName, liq.GetQuery().GetQuery())
					}
					if liq.GetBlastPropagation().GetIn() != false {
						t.Error("Expected peering blast propagation In=false, got true")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected peering blast propagation Out=true, got false")
					}
					break
				}
			}
			if !hasPeeringLink {
				t.Error("Expected linked query to virtual network peering, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for VNet %s", len(linkedQueries), integrationTestVNetName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete VNet (this also deletes the subnet)
		// Note: deleteVirtualNetwork is already defined in compute-virtual-machine_test.go
		err := deleteVirtualNetwork(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetName)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
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
