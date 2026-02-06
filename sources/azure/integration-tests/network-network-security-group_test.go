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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
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
	integrationTestNSGName = "ovm-integ-test-nsg"
)

func TestNetworkNetworkSecurityGroupIntegration(t *testing.T) {
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
	nsgClient, err := armnetwork.NewSecurityGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Network Security Groups client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create network security group
		err = createNetworkSecurityGroup(ctx, nsgClient, integrationTestResourceGroup, integrationTestNSGName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create network security group: %v", err)
		}

		// Wait for NSG to be fully available
		err = waitForNSGAvailable(ctx, nsgClient, integrationTestResourceGroup, integrationTestNSGName)
		if err != nil {
			t.Fatalf("Failed waiting for network security group to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetNetworkSecurityGroup", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving network security group %s in subscription %s, resource group %s",
				integrationTestNSGName, subscriptionID, integrationTestResourceGroup)

			nsgWrapper := manual.NewNetworkNetworkSecurityGroup(
				clients.NewNetworkSecurityGroupsClient(nsgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := nsgWrapper.Scopes()[0]

			nsgAdapter := sources.WrapperToAdapter(nsgWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := nsgAdapter.Get(ctx, scope, integrationTestNSGName, true)
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

			if uniqueAttrValue != integrationTestNSGName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestNSGName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved network security group %s", integrationTestNSGName)
		})

		t.Run("ListNetworkSecurityGroups", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing network security groups in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			nsgWrapper := manual.NewNetworkNetworkSecurityGroup(
				clients.NewNetworkSecurityGroupsClient(nsgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := nsgWrapper.Scopes()[0]

			nsgAdapter := sources.WrapperToAdapter(nsgWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := nsgAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list network security groups: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one network security group, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestNSGName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find network security group %s in the list of network security groups", integrationTestNSGName)
			}

			log.Printf("Found %d network security groups in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for network security group %s", integrationTestNSGName)

			nsgWrapper := manual.NewNetworkNetworkSecurityGroup(
				clients.NewNetworkSecurityGroupsClient(nsgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := nsgWrapper.Scopes()[0]

			nsgAdapter := sources.WrapperToAdapter(nsgWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := nsgAdapter.Get(ctx, scope, integrationTestNSGName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.NetworkNetworkSecurityGroup.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.NetworkNetworkSecurityGroup, sdpItem.GetType())
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

			log.Printf("Verified item attributes for network security group %s", integrationTestNSGName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for network security group %s", integrationTestNSGName)

			nsgWrapper := manual.NewNetworkNetworkSecurityGroup(
				clients.NewNetworkSecurityGroupsClient(nsgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := nsgWrapper.Scopes()[0]

			nsgAdapter := sources.WrapperToAdapter(nsgWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := nsgAdapter.Get(ctx, scope, integrationTestNSGName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (if any)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for network security group %s", len(linkedQueries), integrationTestNSGName)

			// For a newly created NSG, there should be default security rules
			// Verify the structure is correct if links exist
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				// Verify query has required fields
				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				// Method should be GET or SEARCH (not empty)
				if query.GetMethod() == sdp.QueryMethod_GET || query.GetMethod() == sdp.QueryMethod_SEARCH {
					// Valid method
				} else {
					t.Errorf("Linked item query has unexpected Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}

				// Verify blast propagation is set
				bp := liq.GetBlastPropagation()
				if bp == nil {
					t.Error("Linked item query has nil BlastPropagation")
				} else {
					// Blast propagation should have In and Out set (even if false)
					_ = bp.GetIn()
					_ = bp.GetOut()
				}

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s, In=%v, Out=%v",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope(),
					bp.GetIn(), bp.GetOut())
			}

			// Verify that default security rules are linked (they should always exist)
			var hasDefaultSecurityRuleLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.NetworkDefaultSecurityRule.String() {
					hasDefaultSecurityRuleLink = true
					// Verify blast propagation for default security rules
					bp := liq.GetBlastPropagation()
					if bp.GetIn() != true {
						t.Error("Expected default security rule blast propagation In=true, got false")
					}
					if bp.GetOut() != false {
						t.Error("Expected default security rule blast propagation Out=false, got true")
					}
					break
				}
			}
			if !hasDefaultSecurityRuleLink {
				t.Error("Expected linked query to default security rules, but didn't find one")
			}

			// Verify that custom security rules are linked (we created one named "AllowSSH")
			var hasSecurityRuleLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.NetworkSecurityRule.String() {
					hasSecurityRuleLink = true
					// Verify blast propagation for security rules
					bp := liq.GetBlastPropagation()
					if bp.GetIn() != true {
						t.Error("Expected security rule blast propagation In=true, got false")
					}
					if bp.GetOut() != false {
						t.Error("Expected security rule blast propagation Out=false, got true")
					}
					// Verify the query contains the NSG name and rule name
					query := liq.GetQuery().GetQuery()
					if query == "" {
						t.Error("Expected security rule query to be non-empty")
					}
					break
				}
			}
			if !hasSecurityRuleLink {
				t.Error("Expected linked query to security rules, but didn't find one")
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete network security group
		err := deleteNetworkSecurityGroup(ctx, nsgClient, integrationTestResourceGroup, integrationTestNSGName)
		if err != nil {
			t.Fatalf("Failed to delete network security group: %v", err)
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

// createNetworkSecurityGroup creates an Azure network security group (idempotent)
func createNetworkSecurityGroup(ctx context.Context, client *armnetwork.SecurityGroupsClient, resourceGroupName, nsgName, location string) error {
	// Check if NSG already exists
	existingNSG, err := client.Get(ctx, resourceGroupName, nsgName, nil)
	if err == nil {
		// NSG exists, check its provisioning state
		if existingNSG.Properties != nil && existingNSG.Properties.ProvisioningState != nil {
			state := *existingNSG.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Network security group %s already exists with state %s, skipping creation", nsgName, state)
				return nil
			}
			log.Printf("Network security group %s exists but in state %s, will wait for it", nsgName, state)
		} else {
			log.Printf("Network security group %s already exists, skipping creation", nsgName)
			return nil
		}
	}

	// Create a basic network security group with a sample security rule
	// This creates an NSG with a default allow rule for testing
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, nsgName, armnetwork.SecurityGroup{
		Location: ptr.To(location),
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			SecurityRules: []*armnetwork.SecurityRule{
				{
					Name: ptr.To("AllowSSH"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Protocol:                 ptr.To(armnetwork.SecurityRuleProtocolTCP),
						SourcePortRange:          ptr.To("*"),
						DestinationPortRange:     ptr.To("22"),
						SourceAddressPrefix:      ptr.To("*"),
						DestinationAddressPrefix: ptr.To("*"),
						Access:                   ptr.To(armnetwork.SecurityRuleAccessAllow),
						Priority:                 ptr.To[int32](1000),
						Direction:                ptr.To(armnetwork.SecurityRuleDirectionInbound),
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("network-network-security-group"),
		},
	}, nil)
	if err != nil {
		// Check if NSG already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Network security group %s already exists (conflict), skipping creation", nsgName)
			return nil
		}
		return fmt.Errorf("failed to begin creating network security group: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create network security group: %w", err)
	}

	// Verify the NSG was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("network security group created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("network security group provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Network security group %s created successfully with provisioning state: %s", nsgName, provisioningState)
	return nil
}

// waitForNSGAvailable polls until the NSG is available via the Get API
// This is needed because even after creation succeeds, there can be a delay before the NSG is queryable
func waitForNSGAvailable(ctx context.Context, client *armnetwork.SecurityGroupsClient, resourceGroupName, nsgName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for network security group %s to be available via API...", nsgName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, nsgName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Network security group %s not yet available (attempt %d/%d), waiting %v...", nsgName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking network security group availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Network security group %s is available with provisioning state: %s", nsgName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("network security group provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Network security group %s provisioning state: %s (attempt %d/%d), waiting...", nsgName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// NSG exists but no provisioning state - consider it available
		log.Printf("Network security group %s is available", nsgName)
		return nil
	}

	return fmt.Errorf("timeout waiting for network security group %s to be available after %d attempts", nsgName, maxAttempts)
}

// deleteNetworkSecurityGroup deletes an Azure network security group
func deleteNetworkSecurityGroup(ctx context.Context, client *armnetwork.SecurityGroupsClient, resourceGroupName, nsgName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, nsgName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Network security group %s not found, skipping deletion", nsgName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting network security group: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete network security group: %w", err)
	}

	log.Printf("Network security group %s deleted successfully", nsgName)
	return nil
}
