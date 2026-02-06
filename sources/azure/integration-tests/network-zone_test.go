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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestZoneName = "ovm-integ-test-zone.com"
)

func TestNetworkZoneIntegration(t *testing.T) {
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
	zonesClient, err := armdns.NewZonesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create DNS Zones client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Generate unique zone name (DNS zone names must be globally unique)
	zoneName := generateZoneName(integrationTestZoneName)

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create DNS zone
		err = createDNSZone(ctx, zonesClient, integrationTestResourceGroup, zoneName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create DNS zone: %v", err)
		}

		// Wait for DNS zone to be available
		err = waitForDNSZoneAvailable(ctx, zonesClient, integrationTestResourceGroup, zoneName)
		if err != nil {
			t.Fatalf("Failed waiting for DNS zone to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetDNSZone", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving DNS zone %s in subscription %s, resource group %s",
				zoneName, subscriptionID, integrationTestResourceGroup)

			zoneWrapper := manual.NewNetworkZone(
				clients.NewZonesClient(zonesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := zoneWrapper.Scopes()[0]

			zoneAdapter := sources.WrapperToAdapter(zoneWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := zoneAdapter.Get(ctx, scope, zoneName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.NetworkZone.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkZone.String(), sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			if uniqueAttrValue != zoneName {
				t.Errorf("Expected unique attribute value %s, got %s", zoneName, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved DNS zone %s", zoneName)
		})

		t.Run("ListDNSZones", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing DNS zones in resource group %s", integrationTestResourceGroup)

			zoneWrapper := manual.NewNetworkZone(
				clients.NewZonesClient(zonesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := zoneWrapper.Scopes()[0]

			zoneAdapter := sources.WrapperToAdapter(zoneWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports list
			listable, ok := zoneAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list DNS zones: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one DNS zone, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					if v == zoneName {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find DNS zone %s in the list results", zoneName)
			}

			log.Printf("Found %d DNS zones in list results", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for DNS zone %s", zoneName)

			zoneWrapper := manual.NewNetworkZone(
				clients.NewZonesClient(zonesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := zoneWrapper.Scopes()[0]

			zoneAdapter := sources.WrapperToAdapter(zoneWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := zoneAdapter.Get(ctx, scope, zoneName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.NetworkZone.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.NetworkZone.String(), sdpItem.GetType())
			}

			// Verify scope
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}

			// Verify item validation
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for DNS zone %s", zoneName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for DNS zone %s", zoneName)

			zoneWrapper := manual.NewNetworkZone(
				clients.NewZonesClient(zonesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := zoneWrapper.Scopes()[0]

			zoneAdapter := sources.WrapperToAdapter(zoneWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := zoneAdapter.Get(ctx, scope, zoneName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			// Verify expected child resource links exist
			expectedChildResources := map[string]bool{
				azureshared.NetworkDNSRecordSet.String(): false,
			}

			// Track found resources
			var hasDNSRecordSetLink bool
			var hasNameServerLinks bool

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				query := liq.GetQuery().GetQuery()
				method := liq.GetQuery().GetMethod()
				linkedScope := liq.GetQuery().GetScope()

				// Verify DNS Record Set link (child resource)
				if linkedType == azureshared.NetworkDNSRecordSet.String() {
					hasDNSRecordSetLink = true
					if expectedChildResources[linkedType] {
						t.Errorf("Found duplicate linked query for type %s", linkedType)
					}
					expectedChildResources[linkedType] = true

					if method != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected linked query method SEARCH for %s, got %s", linkedType, method)
					}

					if query != zoneName {
						t.Errorf("Expected linked query to use zone name %s, got %s", zoneName, query)
					}

					if linkedScope != scope {
						t.Errorf("Expected linked query scope %s, got %s", scope, linkedScope)
					}

					// Verify blast propagation
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Errorf("Expected BlastPropagation to be set for %s", linkedType)
					} else {
						if bp.GetIn() != true {
							t.Errorf("Expected BlastPropagation.In=true for %s, got false", linkedType)
						}
						if bp.GetOut() != true {
							t.Errorf("Expected BlastPropagation.Out=true for %s, got false", linkedType)
						}
					}
				}

				// Verify DNS name server links (standard library)
				if linkedType == "dns" {
					hasNameServerLinks = true
					if method != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected linked query method SEARCH for DNS name server, got %s", method)
					}

					if linkedScope != "global" {
						t.Errorf("Expected linked query scope 'global' for DNS name server, got %s", linkedScope)
					}

					// Verify blast propagation
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Errorf("Expected BlastPropagation to be set for DNS name server")
					} else {
						if bp.GetIn() != true {
							t.Errorf("Expected BlastPropagation.In=true for DNS name server, got false")
						}
						if bp.GetOut() != true {
							t.Errorf("Expected BlastPropagation.Out=true for DNS name server, got false")
						}
					}
				}

				// Verify Virtual Network links (if present)
				if linkedType == azureshared.NetworkVirtualNetwork.String() {
					if method != sdp.QueryMethod_GET {
						t.Errorf("Expected linked query method GET for Virtual Network, got %s", method)
					}

					// Verify blast propagation
					bp := liq.GetBlastPropagation()
					if bp == nil {
						t.Errorf("Expected BlastPropagation to be set for Virtual Network")
					} else {
						if bp.GetIn() != true {
							t.Errorf("Expected BlastPropagation.In=true for Virtual Network, got false")
						}
						if bp.GetOut() != false {
							t.Errorf("Expected BlastPropagation.Out=false for Virtual Network, got true")
						}
					}
				}
			}

			// Check that all expected child resources are linked
			if !hasDNSRecordSetLink {
				t.Error("Expected linked query to DNS Record Set, but didn't find one")
			}

			// Name servers should be present (Azure automatically assigns them)
			if !hasNameServerLinks {
				t.Error("Expected linked queries to DNS name servers, but didn't find any")
			}

			log.Printf("Verified %d linked item queries for DNS zone %s", len(linkedQueries), zoneName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete DNS zone
		err := deleteDNSZone(ctx, zonesClient, integrationTestResourceGroup, zoneName)
		if err != nil {
			t.Fatalf("Failed to delete DNS zone: %v", err)
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

// generateZoneName generates a unique DNS zone name by appending a timestamp
// DNS zone names must be globally unique
func generateZoneName(baseName string) string {
	return fmt.Sprintf("%s-%d", baseName, time.Now().Unix())
}

// createDNSZone creates an Azure DNS zone (idempotent)
func createDNSZone(ctx context.Context, client *armdns.ZonesClient, resourceGroupName, zoneName, location string) error {
	// Check if zone already exists
	_, err := client.Get(ctx, resourceGroupName, zoneName, nil)
	if err == nil {
		log.Printf("DNS zone %s already exists, skipping creation", zoneName)
		return nil
	}

	// Create the DNS zone
	resp, err := client.CreateOrUpdate(ctx, resourceGroupName, zoneName, armdns.Zone{
		Location: ptr.To(location),
		Properties: &armdns.ZoneProperties{
			ZoneType: ptr.To(armdns.ZoneTypePublic),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"managed": ptr.To("true"),
		},
	}, nil)
	if err != nil {
		// Check if zone already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("DNS zone %s already exists (conflict), skipping creation", zoneName)
			return nil
		}
		return fmt.Errorf("failed to create DNS zone: %w", err)
	}

	// Verify the zone was created successfully
	if resp.ID == nil {
		return fmt.Errorf("DNS zone created but ID is unknown")
	}

	log.Printf("DNS zone %s created successfully", zoneName)
	return nil
}

// waitForDNSZoneAvailable waits for a DNS zone to be available
func waitForDNSZoneAvailable(ctx context.Context, client *armdns.ZonesClient, resourceGroupName, zoneName string) error {
	maxAttempts := 10
	pollInterval := 5 * time.Second

	for i := range maxAttempts {
		resp, err := client.Get(ctx, resourceGroupName, zoneName, nil)
		if err == nil {
			// DNS zones don't have a provisioning state, so if we can get it, it's available
			if resp.ID != nil {
				log.Printf("DNS zone %s is available", zoneName)
				return nil
			}
		}

		if i < maxAttempts-1 {
			time.Sleep(pollInterval)
		}
	}

	return fmt.Errorf("DNS zone %s did not become available within the timeout period", zoneName)
}

// deleteDNSZone deletes an Azure DNS zone
func deleteDNSZone(ctx context.Context, client *armdns.ZonesClient, resourceGroupName, zoneName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, zoneName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("DNS zone %s not found, skipping deletion", zoneName)
			return nil
		}
		return fmt.Errorf("failed to begin deletion of DNS zone: %w", err)
	}

	// Wait for deletion to complete
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("DNS zone %s not found during deletion, assuming already deleted", zoneName)
			return nil
		}
		return fmt.Errorf("failed to delete DNS zone: %w", err)
	}

	log.Printf("DNS zone %s deleted successfully", zoneName)
	return nil
}
