package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
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
	integrationTestIPConfigNICName      = "ovm-integ-test-nic-for-ipconfig"
	integrationTestIPConfigVNetName     = "ovm-integ-test-vnet-for-ipconfig"
	integrationTestIPConfigSubnetName   = "default"
	integrationTestIPConfigIPConfigName = "ipconfig1"
)

func TestNetworkNetworkInterfaceIPConfigurationIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Networks client: %v", err)
	}

	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Subnets client: %v", err)
	}

	nicClient, err := armnetwork.NewInterfacesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Network Interfaces client: %v", err)
	}

	ipConfigClient, err := armnetwork.NewInterfaceIPConfigurationsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Interface IP Configurations client: %v", err)
	}

	setupCompleted := false

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createVirtualNetworkForIPConfig(ctx, vnetClient, integrationTestResourceGroup, integrationTestIPConfigVNetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		subnetResp, err := subnetClient.Get(ctx, integrationTestResourceGroup, integrationTestIPConfigVNetName, integrationTestIPConfigSubnetName, nil)
		if err != nil {
			t.Fatalf("Failed to get subnet: %v", err)
		}

		err = createNetworkInterfaceForIPConfig(ctx, nicClient, integrationTestResourceGroup, integrationTestIPConfigNICName, integrationTestLocation, *subnetResp.ID)
		if err != nil {
			t.Fatalf("Failed to create network interface: %v", err)
		}

		setupCompleted = true
		log.Printf("Setup completed: Network interface %s created with IP configuration %s", integrationTestIPConfigNICName, integrationTestIPConfigIPConfigName)
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetIPConfiguration", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving IP configuration %s from NIC %s in subscription %s, resource group %s",
				integrationTestIPConfigIPConfigName, integrationTestIPConfigNICName, subscriptionID, integrationTestResourceGroup)

			ipConfigWrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(
				clients.NewInterfaceIPConfigurationsClient(ipConfigClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ipConfigWrapper.Scopes()[0]

			ipConfigAdapter := sources.WrapperToAdapter(ipConfigWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestIPConfigNICName, integrationTestIPConfigIPConfigName)
			sdpItem, qErr := ipConfigAdapter.Get(ctx, scope, query, true)
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

			expectedUniqueValue := shared.CompositeLookupKey(integrationTestIPConfigNICName, integrationTestIPConfigIPConfigName)
			if uniqueAttrValue != expectedUniqueValue {
				t.Fatalf("Expected unique attribute value to be %s, got %s", expectedUniqueValue, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved IP configuration %s", integrationTestIPConfigIPConfigName)
		})

		t.Run("SearchIPConfigurations", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching IP configurations in NIC %s", integrationTestIPConfigNICName)

			ipConfigWrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(
				clients.NewInterfaceIPConfigurationsClient(ipConfigClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ipConfigWrapper.Scopes()[0]

			ipConfigAdapter := sources.WrapperToAdapter(ipConfigWrapper, sdpcache.NewNoOpCache())
			searchable, ok := ipConfigAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestIPConfigNICName, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) == 0 {
				t.Fatalf("Expected at least 1 IP configuration, got: %d", len(sdpItems))
			}

			var found bool
			expectedUniqueValue := shared.CompositeLookupKey(integrationTestIPConfigNICName, integrationTestIPConfigIPConfigName)
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedUniqueValue {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find IP configuration %s in search results", integrationTestIPConfigIPConfigName)
			}

			log.Printf("Successfully found %d IP configurations in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for IP configuration %s", integrationTestIPConfigIPConfigName)

			ipConfigWrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(
				clients.NewInterfaceIPConfigurationsClient(ipConfigClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ipConfigWrapper.Scopes()[0]

			ipConfigAdapter := sources.WrapperToAdapter(ipConfigWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestIPConfigNICName, integrationTestIPConfigIPConfigName)
			sdpItem, qErr := ipConfigAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasNICLink bool
			var hasSubnetLink bool
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query.GetType() == "" {
					t.Error("Linked item query has empty type")
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty scope")
				}

				switch query.GetType() {
				case azureshared.NetworkNetworkInterface.String():
					hasNICLink = true
					if query.GetQuery() != integrationTestIPConfigNICName {
						t.Errorf("Expected linked query to NIC %s, got %s", integrationTestIPConfigNICName, query.GetQuery())
					}
				case azureshared.NetworkSubnet.String():
					hasSubnetLink = true
				}
			}

			if !hasNICLink {
				t.Error("Expected linked query to parent network interface, but didn't find one")
			}

			if !hasSubnetLink {
				t.Error("Expected linked query to subnet, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for IP configuration %s", len(linkedQueries), integrationTestIPConfigIPConfigName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			ipConfigWrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(
				clients.NewInterfaceIPConfigurationsClient(ipConfigClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := ipConfigWrapper.Scopes()[0]

			ipConfigAdapter := sources.WrapperToAdapter(ipConfigWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestIPConfigNICName, integrationTestIPConfigIPConfigName)
			sdpItem, qErr := ipConfigAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkNetworkInterfaceIPConfiguration.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkNetworkInterfaceIPConfiguration, sdpItem.GetType())
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

			log.Printf("Verified item attributes for IP configuration %s", integrationTestIPConfigIPConfigName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteNetworkInterfaceForIPConfig(ctx, nicClient, integrationTestResourceGroup, integrationTestIPConfigNICName)
		if err != nil {
			t.Fatalf("Failed to delete network interface: %v", err)
		}

		err = deleteVirtualNetworkForIPConfig(ctx, vnetClient, integrationTestResourceGroup, integrationTestIPConfigVNetName)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}

func createVirtualNetworkForIPConfig(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, vnetName, nil)
	if err == nil {
		log.Printf("Virtual network %s already exists, skipping creation", vnetName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, armnetwork.VirtualNetwork{
		Location: new(location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{new("10.2.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: new(integrationTestIPConfigSubnetName),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: new("10.2.0.0/24"),
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin creating virtual network: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create virtual network: %w", err)
	}

	log.Printf("Virtual network %s created successfully", vnetName)
	return nil
}

func deleteVirtualNetworkForIPConfig(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, vnetName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Virtual network %s not found, skipping deletion", vnetName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting virtual network: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual network: %w", err)
	}

	log.Printf("Virtual network %s deleted successfully", vnetName)
	return nil
}

func createNetworkInterfaceForIPConfig(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroupName, nicName, location, subnetID string) error {
	_, err := client.Get(ctx, resourceGroupName, nicName, nil)
	if err == nil {
		log.Printf("Network interface %s already exists, skipping creation", nicName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, nicName, armnetwork.Interface{
		Location: new(location),
		Properties: &armnetwork.InterfacePropertiesFormat{
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: new(integrationTestIPConfigIPConfigName),
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						Subnet: &armnetwork.Subnet{
							ID: new(subnetID),
						},
						PrivateIPAllocationMethod: new(armnetwork.IPAllocationMethodDynamic),
						Primary:                   new(true),
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin creating network interface: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create network interface: %w", err)
	}

	log.Printf("Network interface %s created successfully", nicName)
	return nil
}

func deleteNetworkInterfaceForIPConfig(ctx context.Context, client *armnetwork.InterfacesClient, resourceGroupName, nicName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, nicName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Network interface %s not found, skipping deletion", nicName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting network interface: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete network interface: %w", err)
	}

	log.Printf("Network interface %s deleted successfully", nicName)
	return nil
}
