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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
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
	integrationTestDNSVNetLinkName = "ovm-integ-test-dns-vnet-link"
	integrationTestPrivateZoneName = "ovm-integ-test.private.zone"
	integrationTestVNetForDNSName  = "ovm-integ-test-vnet-dns"
)

func TestNetworkDNSVirtualNetworkLinkIntegration(t *testing.T) {
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

	privateDNSZonesClient, err := armprivatedns.NewPrivateZonesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Private DNS Zones client: %v", err)
	}

	vnetLinksClient, err := armprivatedns.NewVirtualNetworkLinksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Network Links client: %v", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Networks client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createVNetForDNS(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetForDNSName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		err = createPrivateDNSZoneForLink(ctx, privateDNSZonesClient, integrationTestResourceGroup, integrationTestPrivateZoneName)
		if err != nil {
			t.Fatalf("Failed to create private DNS zone: %v", err)
		}

		err = waitForPrivateDNSZoneAvailable(ctx, privateDNSZonesClient, integrationTestResourceGroup, integrationTestPrivateZoneName)
		if err != nil {
			t.Fatalf("Failed waiting for private DNS zone: %v", err)
		}

		vnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s",
			subscriptionID, integrationTestResourceGroup, integrationTestVNetForDNSName)

		err = createVirtualNetworkLink(ctx, vnetLinksClient, integrationTestResourceGroup, integrationTestPrivateZoneName, integrationTestDNSVNetLinkName, vnetID, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network link: %v", err)
		}

		err = waitForVirtualNetworkLinkAvailable(ctx, vnetLinksClient, integrationTestResourceGroup, integrationTestPrivateZoneName, integrationTestDNSVNetLinkName)
		if err != nil {
			t.Fatalf("Failed waiting for virtual network link: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetVirtualNetworkLink", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkDNSVirtualNetworkLink(
				clients.NewVirtualNetworkLinksClient(vnetLinksClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestPrivateZoneName, integrationTestDNSVNetLinkName)
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

			expectedUnique := shared.CompositeLookupKey(integrationTestPrivateZoneName, integrationTestDNSVNetLinkName)
			if uniqueAttrValue != expectedUnique {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved virtual network link %s", integrationTestDNSVNetLinkName)
		})

		t.Run("SearchVirtualNetworkLinks", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkDNSVirtualNetworkLink(
				clients.NewVirtualNetworkLinksClient(vnetLinksClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestPrivateZoneName, true)
			if err != nil {
				t.Fatalf("Failed to search virtual network links: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one virtual network link, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == shared.CompositeLookupKey(integrationTestPrivateZoneName, integrationTestDNSVNetLinkName) {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find link %s in the search results", integrationTestDNSVNetLinkName)
			}

			log.Printf("Found %d virtual network links in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkDNSVirtualNetworkLink(
				clients.NewVirtualNetworkLinksClient(vnetLinksClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestPrivateZoneName, integrationTestDNSVNetLinkName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasPrivateDNSZoneLink, hasVNetLink bool
			for _, liq := range linkedQueries {
				q := liq.GetQuery()
				if q.GetType() == azureshared.NetworkPrivateDNSZone.String() && q.GetQuery() == integrationTestPrivateZoneName {
					hasPrivateDNSZoneLink = true
				}
				if q.GetType() == azureshared.NetworkVirtualNetwork.String() && q.GetQuery() == integrationTestVNetForDNSName {
					hasVNetLink = true
				}
			}

			if !hasPrivateDNSZoneLink {
				t.Error("Expected linked query to Private DNS Zone, but didn't find one")
			}
			if !hasVNetLink {
				t.Error("Expected linked query to Virtual Network, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for virtual network link %s", len(linkedQueries), integrationTestDNSVNetLinkName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkDNSVirtualNetworkLink(
				clients.NewVirtualNetworkLinksClient(vnetLinksClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestPrivateZoneName, integrationTestDNSVNetLinkName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkDNSVirtualNetworkLink.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkDNSVirtualNetworkLink.String(), sdpItem.GetType())
			}

			expectedScope := subscriptionID + "." + integrationTestResourceGroup
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Expected item to validate, got: %v", err)
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteVirtualNetworkLink(ctx, vnetLinksClient, integrationTestResourceGroup, integrationTestPrivateZoneName, integrationTestDNSVNetLinkName)
		if err != nil {
			t.Fatalf("Failed to delete virtual network link: %v", err)
		}

		log.Printf("Waiting 30 seconds for VNet link deletion to propagate before deleting DNS zone...")
		time.Sleep(30 * time.Second)

		err = deletePrivateDNSZoneForLink(ctx, privateDNSZonesClient, integrationTestResourceGroup, integrationTestPrivateZoneName)
		if err != nil {
			t.Fatalf("Failed to delete private DNS zone: %v", err)
		}

		err = deleteVNetForDNS(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetForDNSName)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}

func createVNetForDNS(ctx context.Context, client *armnetwork.VirtualNetworksClient, rg, name, location string) error {
	_, err := client.Get(ctx, rg, name, nil)
	if err == nil {
		log.Printf("Virtual network %s already exists, skipping creation", name)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, rg, name, armnetwork.VirtualNetwork{
		Location: new(location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{new("10.100.0.0/16")},
			},
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Virtual network %s already exists (conflict), skipping", name)
			return nil
		}
		return fmt.Errorf("failed to create virtual network: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create virtual network: %w", err)
	}
	log.Printf("Virtual network %s created successfully", name)
	return nil
}

func createPrivateDNSZoneForLink(ctx context.Context, client *armprivatedns.PrivateZonesClient, rg, zoneName string) error {
	_, err := client.Get(ctx, rg, zoneName, nil)
	if err == nil {
		log.Printf("Private DNS zone %s already exists, skipping creation", zoneName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, rg, zoneName, armprivatedns.PrivateZone{
		Location: new("global"),
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Private DNS zone %s already exists (conflict), skipping", zoneName)
			return nil
		}
		return fmt.Errorf("failed to create private DNS zone: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create private DNS zone: %w", err)
	}
	log.Printf("Private DNS zone %s created successfully", zoneName)
	return nil
}

func waitForPrivateDNSZoneAvailable(ctx context.Context, client *armprivatedns.PrivateZonesClient, rg, zoneName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, rg, zoneName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking private DNS zone: %w", err)
		}
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armprivatedns.ProvisioningStateSucceeded {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for private DNS zone %s", zoneName)
}

func createVirtualNetworkLink(ctx context.Context, client *armprivatedns.VirtualNetworkLinksClient, rg, zoneName, linkName, vnetID, location string) error {
	_, err := client.Get(ctx, rg, zoneName, linkName, nil)
	if err == nil {
		log.Printf("Virtual network link %s already exists, skipping creation", linkName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, rg, zoneName, linkName, armprivatedns.VirtualNetworkLink{
		Location: new("global"),
		Properties: &armprivatedns.VirtualNetworkLinkProperties{
			VirtualNetwork: &armprivatedns.SubResource{
				ID: &vnetID,
			},
			RegistrationEnabled: new(false),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Virtual network link %s already exists (conflict), skipping", linkName)
			return nil
		}
		return fmt.Errorf("failed to create virtual network link: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create virtual network link: %w", err)
	}
	log.Printf("Virtual network link %s created successfully", linkName)
	return nil
}

func waitForVirtualNetworkLinkAvailable(ctx context.Context, client *armprivatedns.VirtualNetworkLinksClient, rg, zoneName, linkName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, rg, zoneName, linkName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking virtual network link: %w", err)
		}
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armprivatedns.ProvisioningStateSucceeded {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for virtual network link %s", linkName)
}

func deleteVirtualNetworkLink(ctx context.Context, client *armprivatedns.VirtualNetworkLinksClient, rg, zoneName, linkName string) error {
	poller, err := client.BeginDelete(ctx, rg, zoneName, linkName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Virtual network link %s not found, skipping deletion", linkName)
			return nil
		}
		return fmt.Errorf("failed to delete virtual network link: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual network link: %w", err)
	}
	log.Printf("Virtual network link %s deleted successfully", linkName)
	return nil
}

func deletePrivateDNSZoneForLink(ctx context.Context, client *armprivatedns.PrivateZonesClient, rg, zoneName string) error {
	poller, err := client.BeginDelete(ctx, rg, zoneName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Private DNS zone %s not found, skipping deletion", zoneName)
			return nil
		}
		return fmt.Errorf("failed to delete private DNS zone: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete private DNS zone: %w", err)
	}
	log.Printf("Private DNS zone %s deleted successfully", zoneName)
	return nil
}

func deleteVNetForDNS(ctx context.Context, client *armnetwork.VirtualNetworksClient, rg, name string) error {
	poller, err := client.BeginDelete(ctx, rg, name, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Virtual network %s not found, skipping deletion", name)
			return nil
		}
		return fmt.Errorf("failed to delete virtual network: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete virtual network: %w", err)
	}
	log.Printf("Virtual network %s deleted successfully", name)
	return nil
}
