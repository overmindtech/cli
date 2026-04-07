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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestVPNConnectionName = "ovm-integ-test-vpn-conn"
	integrationTestVPNVNetName       = "ovm-integ-test-vpn-vnet"
	integrationTestVPNSubnetName     = "GatewaySubnet"
	integrationTestVPNGatewayName    = "ovm-integ-test-vpn-gw"
	integrationTestVPNPublicIPName   = "ovm-integ-test-vpn-pip"
	integrationTestVPNLocalGWName    = "ovm-integ-test-vpn-lgw"
)

func TestNetworkVirtualNetworkGatewayConnectionIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	vpnConnectionsClient, err := armnetwork.NewVirtualNetworkGatewayConnectionsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create VPN Connections client: %v", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Networks client: %v", err)
	}

	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Subnets client: %v", err)
	}

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Public IP Addresses client: %v", err)
	}

	vpnGatewayClient, err := armnetwork.NewVirtualNetworkGatewaysClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create VPN Gateways client: %v", err)
	}

	localGatewayClient, err := armnetwork.NewLocalNetworkGatewaysClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Local Network Gateways client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 55*time.Minute)
		defer cancel()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createVPNTestVNet(ctx, vnetClient, integrationTestResourceGroup, integrationTestVPNVNetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create VNet: %v", err)
		}

		err = createVPNGatewaySubnet(ctx, subnetClient, integrationTestResourceGroup, integrationTestVPNVNetName, integrationTestVPNSubnetName)
		if err != nil {
			t.Fatalf("Failed to create GatewaySubnet: %v", err)
		}

		err = createVPNPublicIP(ctx, publicIPClient, integrationTestResourceGroup, integrationTestVPNPublicIPName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create public IP: %v", err)
		}

		err = waitForVPNPublicIPAvailable(ctx, publicIPClient, integrationTestResourceGroup, integrationTestVPNPublicIPName)
		if err != nil {
			t.Fatalf("Failed waiting for public IP: %v", err)
		}

		err = createVPNLocalGateway(ctx, localGatewayClient, integrationTestResourceGroup, integrationTestVPNLocalGWName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create local network gateway: %v", err)
		}

		err = waitForVPNLocalGatewayAvailable(ctx, localGatewayClient, integrationTestResourceGroup, integrationTestVPNLocalGWName)
		if err != nil {
			t.Fatalf("Failed waiting for local gateway: %v", err)
		}

		err = createVPNGateway(ctx, vpnGatewayClient, subscriptionID, integrationTestResourceGroup, integrationTestVPNGatewayName, integrationTestVPNVNetName, integrationTestVPNSubnetName, integrationTestVPNPublicIPName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create VPN gateway: %v", err)
		}

		err = waitForVPNGatewayAvailable(ctx, vpnGatewayClient, integrationTestResourceGroup, integrationTestVPNGatewayName)
		if err != nil {
			t.Fatalf("Failed waiting for VPN gateway: %v", err)
		}

		err = createVPNConnection(ctx, vpnConnectionsClient, subscriptionID, integrationTestResourceGroup, integrationTestVPNConnectionName, integrationTestVPNGatewayName, integrationTestVPNLocalGWName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create VPN connection: %v", err)
		}

		err = waitForVPNConnectionAvailable(ctx, vpnConnectionsClient, integrationTestResourceGroup, integrationTestVPNConnectionName)
		if err != nil {
			t.Fatalf("Failed waiting for VPN connection: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetVPNConnection", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving VPN connection %s in subscription %s, resource group %s",
				integrationTestVPNConnectionName, subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(
				clients.NewVirtualNetworkGatewayConnectionsClient(vpnConnectionsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestVPNConnectionName, true)
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

			if uniqueAttrValue != integrationTestVPNConnectionName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestVPNConnectionName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved VPN connection %s", integrationTestVPNConnectionName)
		})

		t.Run("ListVPNConnections", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing VPN connections in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(
				clients.NewVirtualNetworkGatewayConnectionsClient(vpnConnectionsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			listable, ok := adapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list VPN connections: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one VPN connection, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestVPNConnectionName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find VPN connection %s in the list", integrationTestVPNConnectionName)
			}

			log.Printf("Found %d VPN connections in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for VPN connection %s", integrationTestVPNConnectionName)

			wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(
				clients.NewVirtualNetworkGatewayConnectionsClient(vpnConnectionsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestVPNConnectionName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkVirtualNetworkGatewayConnection.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.NetworkVirtualNetworkGatewayConnection, sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for VPN connection %s", integrationTestVPNConnectionName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for VPN connection %s", integrationTestVPNConnectionName)

			wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(
				clients.NewVirtualNetworkGatewayConnectionsClient(vpnConnectionsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestVPNConnectionName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for VPN connection %s", len(linkedQueries), integrationTestVPNConnectionName)

			if len(linkedQueries) < 1 {
				t.Error("Expected at least one linked item query (VirtualNetworkGateway1)")
			}

			var hasVNGLink, hasLNGLink bool
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if query.GetMethod() != sdp.QueryMethod_GET && query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Linked item query has unexpected Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}

				if query.GetType() == azureshared.NetworkVirtualNetworkGateway.String() {
					hasVNGLink = true
				}
				if query.GetType() == azureshared.NetworkLocalNetworkGateway.String() {
					hasLNGLink = true
				}

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope())
			}

			if !hasVNGLink {
				t.Error("Expected a linked item query for VirtualNetworkGateway")
			}
			if !hasLNGLink {
				t.Error("Expected a linked item query for LocalNetworkGateway")
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Minute)
		defer cancel()

		err := deleteVPNConnection(ctx, vpnConnectionsClient, integrationTestResourceGroup, integrationTestVPNConnectionName)
		if err != nil {
			log.Printf("Warning: Failed to delete VPN connection: %v", err)
		}

		err = deleteVPNGateway(ctx, vpnGatewayClient, integrationTestResourceGroup, integrationTestVPNGatewayName)
		if err != nil {
			log.Printf("Warning: Failed to delete VPN gateway: %v", err)
		}

		err = deleteVPNLocalGateway(ctx, localGatewayClient, integrationTestResourceGroup, integrationTestVPNLocalGWName)
		if err != nil {
			log.Printf("Warning: Failed to delete local gateway: %v", err)
		}

		err = deleteVPNPublicIP(ctx, publicIPClient, integrationTestResourceGroup, integrationTestVPNPublicIPName)
		if err != nil {
			log.Printf("Warning: Failed to delete public IP: %v", err)
		}

		err = deleteVPNVNet(ctx, vnetClient, integrationTestResourceGroup, integrationTestVPNVNetName)
		if err != nil {
			log.Printf("Warning: Failed to delete VNet: %v", err)
		}
	})
}

func createVPNTestVNet(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
	existingVNet, err := client.Get(ctx, resourceGroupName, vnetName, nil)
	if err == nil && existingVNet.Properties != nil {
		log.Printf("VNet %s already exists, skipping creation", vnetName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, armnetwork.VirtualNetwork{
		Location: &location,
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{new("10.200.0.0/16")},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("network-virtual-network-gateway-connection"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("VNet %s already exists (conflict), skipping creation", vnetName)
			return nil
		}
		return fmt.Errorf("failed to begin creating VNet: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create VNet: %w", err)
	}

	log.Printf("VNet %s created successfully", vnetName)
	return nil
}

func createVPNGatewaySubnet(ctx context.Context, client *armnetwork.SubnetsClient, resourceGroupName, vnetName, subnetName string) error {
	existingSubnet, err := client.Get(ctx, resourceGroupName, vnetName, subnetName, nil)
	if err == nil && existingSubnet.Properties != nil {
		log.Printf("Subnet %s already exists, skipping creation", subnetName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, subnetName, armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: new("10.200.255.0/27"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Subnet %s already exists (conflict), skipping creation", subnetName)
			return nil
		}
		return fmt.Errorf("failed to begin creating subnet: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create subnet: %w", err)
	}

	log.Printf("Subnet %s created successfully", subnetName)
	return nil
}

func createVPNPublicIP(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, pipName, location string) error {
	existingPIP, err := client.Get(ctx, resourceGroupName, pipName, nil)
	if err == nil && existingPIP.Properties != nil {
		log.Printf("Public IP %s already exists, skipping creation", pipName)
		return nil
	}

	allocMethodStatic := armnetwork.IPAllocationMethodStatic
	skuNameStandard := armnetwork.PublicIPAddressSKUNameStandard
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, pipName, armnetwork.PublicIPAddress{
		Location: &location,
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: &allocMethodStatic,
		},
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: &skuNameStandard,
		},
		Zones: []*string{new("1"), new("2"), new("3")},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("network-virtual-network-gateway-connection"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Public IP %s already exists (conflict), skipping creation", pipName)
			return nil
		}
		return fmt.Errorf("failed to begin creating public IP: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create public IP: %w", err)
	}

	log.Printf("Public IP %s created successfully", pipName)
	return nil
}

func waitForVPNPublicIPAvailable(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, pipName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, pipName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Public IP %s not yet available (attempt %d/%d)", pipName, attempt, maxAttempts)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking public IP: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armnetwork.ProvisioningStateSucceeded {
			log.Printf("Public IP %s is available", pipName)
			return nil
		}
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for public IP %s", pipName)
}

func createVPNLocalGateway(ctx context.Context, client *armnetwork.LocalNetworkGatewaysClient, resourceGroupName, gatewayName, location string) error {
	existingGW, err := client.Get(ctx, resourceGroupName, gatewayName, nil)
	if err == nil && existingGW.Properties != nil {
		log.Printf("Local network gateway %s already exists, skipping creation", gatewayName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, gatewayName, armnetwork.LocalNetworkGateway{
		Location: &location,
		Properties: &armnetwork.LocalNetworkGatewayPropertiesFormat{
			GatewayIPAddress: new("203.0.113.1"),
			LocalNetworkAddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{new("10.100.0.0/16")},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("network-virtual-network-gateway-connection"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Local network gateway %s already exists (conflict), skipping creation", gatewayName)
			return nil
		}
		return fmt.Errorf("failed to begin creating local network gateway: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create local network gateway: %w", err)
	}

	log.Printf("Local network gateway %s created successfully", gatewayName)
	return nil
}

func waitForVPNLocalGatewayAvailable(ctx context.Context, client *armnetwork.LocalNetworkGatewaysClient, resourceGroupName, gatewayName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, gatewayName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Local network gateway %s not yet available (attempt %d/%d)", gatewayName, attempt, maxAttempts)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking local network gateway: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armnetwork.ProvisioningStateSucceeded {
			log.Printf("Local network gateway %s is available", gatewayName)
			return nil
		}
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for local network gateway %s", gatewayName)
}

func createVPNGateway(ctx context.Context, client *armnetwork.VirtualNetworkGatewaysClient, subscriptionID, resourceGroupName, gatewayName, vnetName, subnetName, pipName, location string) error {
	existingGW, err := client.Get(ctx, resourceGroupName, gatewayName, nil)
	if err == nil && existingGW.Properties != nil {
		log.Printf("VPN gateway %s already exists, skipping creation", gatewayName)
		return nil
	}

	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		subscriptionID, resourceGroupName, vnetName, subnetName)
	pipID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s",
		subscriptionID, resourceGroupName, pipName)

	gatewayTypeVPN := armnetwork.VirtualNetworkGatewayTypeVPN
	vpnTypeRouteBased := armnetwork.VPNTypeRouteBased
	skuNameVPNGw1AZ := armnetwork.VirtualNetworkGatewaySKUNameVPNGw1AZ
	skuTierVPNGw1AZ := armnetwork.VirtualNetworkGatewaySKUTierVPNGw1AZ
	allocMethodDynamic := armnetwork.IPAllocationMethodDynamic
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, gatewayName, armnetwork.VirtualNetworkGateway{
		Location: &location,
		Properties: &armnetwork.VirtualNetworkGatewayPropertiesFormat{
			GatewayType: &gatewayTypeVPN,
			VPNType:     &vpnTypeRouteBased,
			SKU: &armnetwork.VirtualNetworkGatewaySKU{
				Name: &skuNameVPNGw1AZ,
				Tier: &skuTierVPNGw1AZ,
			},
			IPConfigurations: []*armnetwork.VirtualNetworkGatewayIPConfiguration{
				{
					Name: new("default"),
					Properties: &armnetwork.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: &allocMethodDynamic,
						Subnet: &armnetwork.SubResource{
							ID: &subnetID,
						},
						PublicIPAddress: &armnetwork.SubResource{
							ID: &pipID,
						},
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("network-virtual-network-gateway-connection"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("VPN gateway %s already exists (conflict), skipping creation", gatewayName)
			return nil
		}
		return fmt.Errorf("failed to begin creating VPN gateway: %w", err)
	}

	log.Printf("VPN gateway %s creation started, this may take 20-45 minutes...", gatewayName)
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create VPN gateway: %w", err)
	}

	log.Printf("VPN gateway %s created successfully", gatewayName)
	return nil
}

func waitForVPNGatewayAvailable(ctx context.Context, client *armnetwork.VirtualNetworkGatewaysClient, resourceGroupName, gatewayName string) error {
	maxAttempts := 60
	pollInterval := 30 * time.Second

	log.Printf("Waiting for VPN gateway %s to be available (this may take 20-45 minutes)...", gatewayName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, gatewayName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("VPN gateway %s not yet available (attempt %d/%d)", gatewayName, attempt, maxAttempts)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking VPN gateway: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == armnetwork.ProvisioningStateSucceeded {
				log.Printf("VPN gateway %s is available", gatewayName)
				return nil
			}
			if state == armnetwork.ProvisioningStateFailed {
				return fmt.Errorf("VPN gateway %s provisioning failed", gatewayName)
			}
			log.Printf("VPN gateway %s state: %s (attempt %d/%d)", gatewayName, state, attempt, maxAttempts)
		}
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for VPN gateway %s", gatewayName)
}

func createVPNConnection(ctx context.Context, client *armnetwork.VirtualNetworkGatewayConnectionsClient, subscriptionID, resourceGroupName, connectionName, gatewayName, localGatewayName, location string) error {
	existingConn, err := client.Get(ctx, resourceGroupName, connectionName, nil)
	if err == nil && existingConn.Properties != nil {
		log.Printf("VPN connection %s already exists, skipping creation", connectionName)
		return nil
	}

	vpnGatewayID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworkGateways/%s",
		subscriptionID, resourceGroupName, gatewayName)
	localGatewayID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/localNetworkGateways/%s",
		subscriptionID, resourceGroupName, localGatewayName)

	connTypeIPsec := armnetwork.VirtualNetworkGatewayConnectionTypeIPsec
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, connectionName, armnetwork.VirtualNetworkGatewayConnection{
		Location: &location,
		Properties: &armnetwork.VirtualNetworkGatewayConnectionPropertiesFormat{
			ConnectionType: &connTypeIPsec,
			VirtualNetworkGateway1: &armnetwork.VirtualNetworkGateway{
				ID: &vpnGatewayID,
			},
			LocalNetworkGateway2: &armnetwork.LocalNetworkGateway{
				ID: &localGatewayID,
			},
			SharedKey: new("overmind-test-key-12345"),
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("network-virtual-network-gateway-connection"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("VPN connection %s already exists (conflict), skipping creation", connectionName)
			return nil
		}
		return fmt.Errorf("failed to begin creating VPN connection: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create VPN connection: %w", err)
	}

	log.Printf("VPN connection %s created successfully", connectionName)
	return nil
}

func waitForVPNConnectionAvailable(ctx context.Context, client *armnetwork.VirtualNetworkGatewayConnectionsClient, resourceGroupName, connectionName string) error {
	maxAttempts := 30
	pollInterval := 10 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, connectionName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("VPN connection %s not yet available (attempt %d/%d)", connectionName, attempt, maxAttempts)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking VPN connection: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == armnetwork.ProvisioningStateSucceeded {
				log.Printf("VPN connection %s is available", connectionName)
				return nil
			}
			if state == armnetwork.ProvisioningStateFailed {
				return fmt.Errorf("VPN connection %s provisioning failed", connectionName)
			}
			log.Printf("VPN connection %s state: %s (attempt %d/%d)", connectionName, state, attempt, maxAttempts)
		}
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for VPN connection %s", connectionName)
}

func deleteVPNConnection(ctx context.Context, client *armnetwork.VirtualNetworkGatewayConnectionsClient, resourceGroupName, connectionName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, connectionName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("VPN connection %s not found, skipping deletion", connectionName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting VPN connection: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete VPN connection: %w", err)
	}

	log.Printf("VPN connection %s deleted successfully", connectionName)
	return nil
}

func deleteVPNGateway(ctx context.Context, client *armnetwork.VirtualNetworkGatewaysClient, resourceGroupName, gatewayName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, gatewayName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("VPN gateway %s not found, skipping deletion", gatewayName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting VPN gateway: %w", err)
	}

	log.Printf("VPN gateway %s deletion started, this may take several minutes...", gatewayName)
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete VPN gateway: %w", err)
	}

	log.Printf("VPN gateway %s deleted successfully", gatewayName)
	return nil
}

func deleteVPNLocalGateway(ctx context.Context, client *armnetwork.LocalNetworkGatewaysClient, resourceGroupName, gatewayName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, gatewayName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Local network gateway %s not found, skipping deletion", gatewayName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting local network gateway: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete local network gateway: %w", err)
	}

	log.Printf("Local network gateway %s deleted successfully", gatewayName)
	return nil
}

func deleteVPNPublicIP(ctx context.Context, client *armnetwork.PublicIPAddressesClient, resourceGroupName, pipName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, pipName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Public IP %s not found, skipping deletion", pipName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting public IP: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete public IP: %w", err)
	}

	log.Printf("Public IP %s deleted successfully", pipName)
	return nil
}

func deleteVPNVNet(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, vnetName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("VNet %s not found, skipping deletion", vnetName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting VNet: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete VNet: %w", err)
	}

	log.Printf("VNet %s deleted successfully", vnetName)
	return nil
}
