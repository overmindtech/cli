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
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestLBForProbeName     = "ovm-integ-test-lb-probe"
	integrationTestVNetForProbeName   = "ovm-integ-test-vnet-for-probe"
	integrationTestSubnetForProbeName = "default"
	integrationTestPublicIPForProbeLB = "ovm-integ-test-pip-for-probe-lb"
	integrationTestProbeName          = "ovm-integ-test-health-probe"
	integrationTestProbeHTTPName      = "ovm-integ-test-http-probe"
)

func TestNetworkLoadBalancerProbeIntegration(t *testing.T) {
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

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Public IP Addresses client: %v", err)
	}

	lbClient, err := armnetwork.NewLoadBalancersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Load Balancers client: %v", err)
	}

	probesClient, err := armnetwork.NewLoadBalancerProbesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Load Balancer Probes client: %v", err)
	}

	setupCompleted := false

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createVNetForProbeTest(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetForProbeName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create virtual network: %v", err)
		}

		err = createPublicIPForProbeTest(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPForProbeLB, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create public IP address: %v", err)
		}

		publicIPResp, err := publicIPClient.Get(ctx, integrationTestResourceGroup, integrationTestPublicIPForProbeLB, nil)
		if err != nil {
			t.Fatalf("Failed to get public IP address: %v", err)
		}

		err = createLBWithProbes(ctx, lbClient, subscriptionID, integrationTestResourceGroup, integrationTestLBForProbeName, integrationTestLocation, *publicIPResp.ID)
		if err != nil {
			t.Fatalf("Failed to create load balancer with probes: %v", err)
		}

		setupCompleted = true
		log.Printf("Setup completed: Load balancer %s with probes created", integrationTestLBForProbeName)
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetProbe", func(t *testing.T) {
			ctx := t.Context()

			probeWrapper := manual.NewNetworkLoadBalancerProbe(
				clients.NewLoadBalancerProbesClient(probesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := probeWrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(probeWrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestLBForProbeName, integrationTestProbeName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkLoadBalancerProbe.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerProbe, sdpItem.GetType())
			}

			expectedUniqueValue := shared.CompositeLookupKey(integrationTestLBForProbeName, integrationTestProbeName)
			if sdpItem.UniqueAttributeValue() != expectedUniqueValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueValue, sdpItem.UniqueAttributeValue())
			}

			log.Printf("Successfully retrieved probe %s from load balancer %s", integrationTestProbeName, integrationTestLBForProbeName)
		})

		t.Run("SearchProbes", func(t *testing.T) {
			ctx := t.Context()

			probeWrapper := manual.NewNetworkLoadBalancerProbe(
				clients.NewLoadBalancerProbesClient(probesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := probeWrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(probeWrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestLBForProbeName, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if len(sdpItems) < 2 {
				t.Fatalf("Expected at least 2 probes, got: %d", len(sdpItems))
			}

			foundTCP := false
			foundHTTP := false
			for _, item := range sdpItems {
				val := item.UniqueAttributeValue()
				if val == shared.CompositeLookupKey(integrationTestLBForProbeName, integrationTestProbeName) {
					foundTCP = true
				}
				if val == shared.CompositeLookupKey(integrationTestLBForProbeName, integrationTestProbeHTTPName) {
					foundHTTP = true
				}
			}

			if !foundTCP {
				t.Errorf("Expected to find TCP probe %s in search results", integrationTestProbeName)
			}
			if !foundHTTP {
				t.Errorf("Expected to find HTTP probe %s in search results", integrationTestProbeHTTPName)
			}

			log.Printf("Successfully searched %d probes for load balancer %s", len(sdpItems), integrationTestLBForProbeName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			probeWrapper := manual.NewNetworkLoadBalancerProbe(
				clients.NewLoadBalancerProbesClient(probesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := probeWrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(probeWrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestLBForProbeName, integrationTestProbeName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			for _, liq := range linkedQueries {
				q := liq.GetQuery()
				if q.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if q.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if q.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}
				if q.GetMethod() != sdp.QueryMethod_GET && q.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Linked item query has invalid Method: %v", q.GetMethod())
				}
			}

			foundParentLB := false
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.NetworkLoadBalancer.String() {
					foundParentLB = true
					if liq.GetQuery().GetQuery() != integrationTestLBForProbeName {
						t.Errorf("Expected parent LB query %s, got %s", integrationTestLBForProbeName, liq.GetQuery().GetQuery())
					}
					break
				}
			}
			if !foundParentLB {
				t.Error("Expected to find parent Load Balancer linked query")
			}

			log.Printf("Verified %d linked item queries for probe %s", len(linkedQueries), integrationTestProbeName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			probeWrapper := manual.NewNetworkLoadBalancerProbe(
				clients.NewLoadBalancerProbesClient(probesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := probeWrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(probeWrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestLBForProbeName, integrationTestProbeName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkLoadBalancerProbe.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerProbe, sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Validation failed: %v", err)
			}

			log.Printf("Verified item attributes for probe %s", integrationTestProbeName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteLBForProbeTest(ctx, lbClient, integrationTestResourceGroup, integrationTestLBForProbeName)
		if err != nil {
			t.Fatalf("Failed to delete load balancer: %v", err)
		}

		err = deletePublicIPForProbeTest(ctx, publicIPClient, integrationTestResourceGroup, integrationTestPublicIPForProbeLB)
		if err != nil {
			t.Fatalf("Failed to delete public IP address: %v", err)
		}

		err = deleteVNetForProbeTest(ctx, vnetClient, integrationTestResourceGroup, integrationTestVNetForProbeName)
		if err != nil {
			t.Fatalf("Failed to delete virtual network: %v", err)
		}
	})
}

func createVNetForProbeTest(ctx context.Context, client *armnetwork.VirtualNetworksClient, rg, name, location string) error {
	_, err := client.Get(ctx, rg, name, nil)
	if err == nil {
		log.Printf("Virtual network %s already exists, skipping creation", name)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, rg, name, armnetwork.VirtualNetwork{
		Location: new(location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{new("10.3.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: new(integrationTestSubnetForProbeName),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: new("10.3.0.0/24"),
					},
				},
			},
		},
		Tags: map[string]*string{"purpose": new("overmind-integration-tests")},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, rg, name, nil); getErr == nil {
				log.Printf("Virtual network %s already exists (conflict), skipping", name)
				return nil
			}
			return fmt.Errorf("virtual network %s conflict but not retrievable: %w", name, err)
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

func deleteVNetForProbeTest(ctx context.Context, client *armnetwork.VirtualNetworksClient, rg, name string) error {
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

func createPublicIPForProbeTest(ctx context.Context, client *armnetwork.PublicIPAddressesClient, rg, name, location string) error {
	_, err := client.Get(ctx, rg, name, nil)
	if err == nil {
		log.Printf("Public IP address %s already exists, skipping creation", name)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, rg, name, armnetwork.PublicIPAddress{
		Location: new(location),
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: new(armnetwork.IPAllocationMethodStatic),
			PublicIPAddressVersion:   new(armnetwork.IPVersionIPv4),
		},
		SKU: &armnetwork.PublicIPAddressSKU{
			Name: new(armnetwork.PublicIPAddressSKUNameStandard),
		},
		Tags: map[string]*string{"purpose": new("overmind-integration-tests")},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, rg, name, nil); getErr == nil {
				log.Printf("Public IP address %s already exists (conflict), skipping", name)
				return nil
			}
			return fmt.Errorf("public IP %s conflict but not retrievable: %w", name, err)
		}
		return fmt.Errorf("failed to create public IP address: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create public IP address: %w", err)
	}
	log.Printf("Public IP address %s created successfully", name)
	return nil
}

func deletePublicIPForProbeTest(ctx context.Context, client *armnetwork.PublicIPAddressesClient, rg, name string) error {
	poller, err := client.BeginDelete(ctx, rg, name, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Public IP address %s not found, skipping deletion", name)
			return nil
		}
		return fmt.Errorf("failed to delete public IP address: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete public IP address: %w", err)
	}
	log.Printf("Public IP address %s deleted successfully", name)
	return nil
}

func createLBWithProbes(ctx context.Context, client *armnetwork.LoadBalancersClient, subscriptionID, rg, name, location, publicIPID string) error {
	_, err := client.Get(ctx, rg, name, nil)
	if err == nil {
		log.Printf("Load balancer %s already exists, skipping creation", name)
		return nil
	}

	frontendIPConfigID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/frontend-config", subscriptionID, rg, name)
	backendPoolID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/backend-pool", subscriptionID, rg, name)
	tcpProbeID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/probes/%s", subscriptionID, rg, name, integrationTestProbeName)

	port80 := int32(80)
	port443 := int32(443)
	intervalInSeconds := int32(15)
	numberOfProbes := int32(2)

	poller, err := client.BeginCreateOrUpdate(ctx, rg, name, armnetwork.LoadBalancer{
		Location: new(location),
		SKU: &armnetwork.LoadBalancerSKU{
			Name: new(armnetwork.LoadBalancerSKUNameStandard),
		},
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					Name: new("frontend-config"),
					Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: new(publicIPID),
						},
					},
				},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{
				{Name: new("backend-pool")},
			},
			Probes: []*armnetwork.Probe{
				{
					Name: new(integrationTestProbeName),
					Properties: &armnetwork.ProbePropertiesFormat{
						Protocol:          new(armnetwork.ProbeProtocolTCP),
						Port:              &port80,
						IntervalInSeconds: &intervalInSeconds,
						NumberOfProbes:    &numberOfProbes,
					},
				},
				{
					Name: new(integrationTestProbeHTTPName),
					Properties: &armnetwork.ProbePropertiesFormat{
						Protocol:          new(armnetwork.ProbeProtocolHTTP),
						Port:              &port443,
						IntervalInSeconds: &intervalInSeconds,
						NumberOfProbes:    &numberOfProbes,
						RequestPath:       new("/health"),
					},
				},
			},
			LoadBalancingRules: []*armnetwork.LoadBalancingRule{
				{
					Name: new("lb-rule-with-probe"),
					Properties: &armnetwork.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &armnetwork.SubResource{ID: new(frontendIPConfigID)},
						BackendAddressPool:      &armnetwork.SubResource{ID: new(backendPoolID)},
						Probe:                   &armnetwork.SubResource{ID: new(tcpProbeID)},
						Protocol:                new(armnetwork.TransportProtocolTCP),
						FrontendPort:            &port80,
						BackendPort:             &port80,
						EnableFloatingIP:        new(false),
						IdleTimeoutInMinutes:    new(int32(4)),
					},
				},
			},
		},
		Tags: map[string]*string{"purpose": new("overmind-integration-tests")},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, rg, name, nil); getErr == nil {
				log.Printf("Load balancer %s already exists (conflict), skipping", name)
				return nil
			}
			return fmt.Errorf("load balancer %s conflict but not retrievable: %w", name, err)
		}
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create load balancer: %w", err)
	}
	log.Printf("Load balancer %s with probes created successfully", name)
	return nil
}

func deleteLBForProbeTest(ctx context.Context, client *armnetwork.LoadBalancersClient, rg, name string) error {
	poller, err := client.BeginDelete(ctx, rg, name, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Load balancer %s not found, skipping deletion", name)
			return nil
		}
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}
	log.Printf("Load balancer %s deleted successfully", name)
	return nil
}
