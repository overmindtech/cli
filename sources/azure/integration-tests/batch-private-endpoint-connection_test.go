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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
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
	integrationTestBatchPECAccountName = "ovm-integ-test-bpec"
	integrationTestBatchPECSAName      = "ovm-integ-test-sa-bpec"
	integrationTestBatchPECVNetName    = "ovm-integ-test-vnet-bpec"
	integrationTestBatchPECSubnetName  = "ovm-integ-test-subnet-bpec"
	integrationTestBatchPECPEName      = "ovm-integ-test-pe-bpec"
)

func TestBatchPrivateEndpointConnectionIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	batchClient, err := armbatch.NewAccountClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Batch Account client: %v", err)
	}

	pecClient, err := armbatch.NewPrivateEndpointConnectionClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Batch Private Endpoint Connection client: %v", err)
	}

	saClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Storage Accounts client: %v", err)
	}

	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Virtual Networks client: %v", err)
	}

	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Subnets client: %v", err)
	}

	peClient, err := armnetwork.NewPrivateEndpointsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Private Endpoints client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	batchAccountName := generateBatchAccountName(integrationTestBatchPECAccountName)
	storageAccountName := generateStorageAccountName(integrationTestBatchPECSAName)
	vnetName := integrationTestBatchPECVNetName
	subnetName := integrationTestBatchPECSubnetName
	peName := integrationTestBatchPECPEName

	setupCompleted := false
	var privateEndpointConnectionName string

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createStorageAccount(ctx, saClient, integrationTestResourceGroup, storageAccountName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create storage account: %v", err)
		}

		err = waitForStorageAccountAvailable(ctx, saClient, integrationTestResourceGroup, storageAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for storage account to be available: %v", err)
		}

		saResp, err := saClient.GetProperties(ctx, integrationTestResourceGroup, storageAccountName, nil)
		if err != nil {
			t.Fatalf("Failed to get storage account properties: %v", err)
		}
		storageAccountID := *saResp.ID

		err = createBatchAccountWithPrivateEndpointPolicy(ctx, batchClient, integrationTestResourceGroup, batchAccountName, integrationTestLocation, storageAccountID)
		if err != nil {
			if errors.Is(err, errBatchQuotaExceeded) {
				t.Skipf("Skipping Batch private endpoint connection integration test due to Azure subscription quota: %v", err)
			}
			t.Fatalf("Failed to create batch account: %v", err)
		}

		err = waitForBatchAccountAvailable(ctx, batchClient, integrationTestResourceGroup, batchAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for batch account to be available: %v", err)
		}

		err = createVNetForBatchPEC(ctx, vnetClient, integrationTestResourceGroup, vnetName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create VNet: %v", err)
		}

		err = createSubnetForBatchPEC(ctx, subnetClient, integrationTestResourceGroup, vnetName, subnetName)
		if err != nil {
			t.Fatalf("Failed to create subnet: %v", err)
		}

		batchResp, err := batchClient.Get(ctx, integrationTestResourceGroup, batchAccountName, nil)
		if err != nil {
			t.Fatalf("Failed to get batch account: %v", err)
		}
		batchAccountID := *batchResp.ID

		err = createPrivateEndpointForBatch(ctx, peClient, integrationTestResourceGroup, peName, integrationTestLocation, batchAccountID, vnetName, subnetName)
		if err != nil {
			t.Fatalf("Failed to create private endpoint: %v", err)
		}

		privateEndpointConnectionName, err = waitForBatchPrivateEndpointConnection(ctx, pecClient, integrationTestResourceGroup, batchAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for private endpoint connection: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetPrivateEndpointConnection", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving batch private endpoint connection %s in account %s", privateEndpointConnectionName, batchAccountName)

			pecWrapper := manual.NewBatchPrivateEndpointConnection(
				clients.NewBatchPrivateEndpointConnectionClient(pecClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pecWrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(pecWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(batchAccountName, privateEndpointConnectionName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.BatchBatchPrivateEndpointConnection.String() {
				t.Errorf("Expected type %s, got %s", azureshared.BatchBatchPrivateEndpointConnection, sdpItem.GetType())
			}

			expectedUniqueAttr := shared.CompositeLookupKey(batchAccountName, privateEndpointConnectionName)
			if sdpItem.UniqueAttributeValue() != expectedUniqueAttr {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttr, sdpItem.UniqueAttributeValue())
			}

			log.Printf("Successfully retrieved private endpoint connection %s", privateEndpointConnectionName)
		})

		t.Run("SearchPrivateEndpointConnections", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching private endpoint connections in batch account %s", batchAccountName)

			pecWrapper := manual.NewBatchPrivateEndpointConnection(
				clients.NewBatchPrivateEndpointConnectionClient(pecClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pecWrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(pecWrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, batchAccountName, true)
			if err != nil {
				t.Fatalf("Failed to search private endpoint connections: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one private endpoint connection, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == shared.CompositeLookupKey(batchAccountName, privateEndpointConnectionName) {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find private endpoint connection %s in the search results", privateEndpointConnectionName)
			}

			log.Printf("Found %d private endpoint connections in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for private endpoint connection %s", privateEndpointConnectionName)

			pecWrapper := manual.NewBatchPrivateEndpointConnection(
				clients.NewBatchPrivateEndpointConnectionClient(pecClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pecWrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(pecWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(batchAccountName, privateEndpointConnectionName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query.GetType() == "" {
					t.Error("LinkedItemQuery has empty Type")
				}
				if query.GetMethod() != sdp.QueryMethod_GET && query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("LinkedItemQuery has invalid Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("LinkedItemQuery has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("LinkedItemQuery has empty Scope")
				}
			}

			var hasBatchAccountLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.BatchBatchAccount.String() {
					hasBatchAccountLink = true
					if liq.GetQuery().GetQuery() != batchAccountName {
						t.Errorf("Expected linked query to batch account %s, got %s", batchAccountName, liq.GetQuery().GetQuery())
					}
					break
				}
			}

			if !hasBatchAccountLink {
				t.Error("Expected linked query to batch account, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for private endpoint connection %s", len(linkedQueries), privateEndpointConnectionName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			pecWrapper := manual.NewBatchPrivateEndpointConnection(
				clients.NewBatchPrivateEndpointConnectionClient(pecClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := pecWrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(pecWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(batchAccountName, privateEndpointConnectionName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.BatchBatchPrivateEndpointConnection.String() {
				t.Errorf("Expected type %s, got %s", azureshared.BatchBatchPrivateEndpointConnection, sdpItem.GetType())
			}

			expectedScope := subscriptionID + "." + integrationTestResourceGroup
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Item validation failed: %v", err)
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deletePrivateEndpointForBatch(ctx, peClient, integrationTestResourceGroup, peName)
		if err != nil {
			t.Errorf("Failed to delete private endpoint: %v", err)
		}

		err = deleteBatchAccount(ctx, batchClient, integrationTestResourceGroup, batchAccountName)
		if err != nil {
			t.Errorf("Failed to delete batch account: %v", err)
		}

		err = deleteSubnetForBatchPEC(ctx, subnetClient, integrationTestResourceGroup, vnetName, subnetName)
		if err != nil {
			t.Errorf("Failed to delete subnet: %v", err)
		}

		err = deleteVNetForBatchPEC(ctx, vnetClient, integrationTestResourceGroup, vnetName)
		if err != nil {
			t.Errorf("Failed to delete VNet: %v", err)
		}

		err = deleteStorageAccount(ctx, saClient, integrationTestResourceGroup, storageAccountName)
		if err != nil {
			t.Errorf("Failed to delete storage account: %v", err)
		}
	})
}

func createBatchAccountWithPrivateEndpointPolicy(ctx context.Context, client *armbatch.AccountClient, resourceGroupName, accountName, location, storageAccountID string) error {
	_, err := client.Get(ctx, resourceGroupName, accountName, nil)
	if err == nil {
		log.Printf("Batch account %s already exists, skipping creation", accountName)
		return nil
	}

	publicNetworkDisabled := armbatch.PublicNetworkAccessTypeDisabled

	poller, err := client.BeginCreate(ctx, resourceGroupName, accountName, armbatch.AccountCreateParameters{
		Location: new(location),
		Properties: &armbatch.AccountCreateProperties{
			AutoStorage: &armbatch.AutoStorageBaseProperties{
				StorageAccountID: new(storageAccountID),
			},
			PoolAllocationMode:  new(armbatch.PoolAllocationModeBatchService),
			PublicNetworkAccess: &publicNetworkDisabled,
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("batch-private-endpoint-connection"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			if respErr.StatusCode == http.StatusConflict {
				log.Printf("Batch account %s already exists (conflict), skipping creation", accountName)
				return nil
			}
			if respErr.ErrorCode == "SubscriptionQuotaExceeded" {
				return fmt.Errorf("%w: %s", errBatchQuotaExceeded, respErr.Error())
			}
		}
		return fmt.Errorf("failed to begin creating batch account: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.ErrorCode == "SubscriptionQuotaExceeded" {
			return fmt.Errorf("%w: %s", errBatchQuotaExceeded, respErr.Error())
		}
		return fmt.Errorf("failed to create batch account: %w", err)
	}

	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("batch account created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != armbatch.ProvisioningStateSucceeded {
		return fmt.Errorf("batch account provisioning state is %s, expected %s", provisioningState, armbatch.ProvisioningStateSucceeded)
	}

	log.Printf("Batch account %s created successfully with private endpoint support", accountName)
	return nil
}

func createVNetForBatchPEC(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, vnetName, nil)
	if err == nil {
		log.Printf("VNet %s already exists, skipping creation", vnetName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, armnetwork.VirtualNetwork{
		Location: new(location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{new("10.0.0.0/16")},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
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

func createSubnetForBatchPEC(ctx context.Context, client *armnetwork.SubnetsClient, resourceGroupName, vnetName, subnetName string) error {
	_, err := client.Get(ctx, resourceGroupName, vnetName, subnetName, nil)
	if err == nil {
		log.Printf("Subnet %s already exists, skipping creation", subnetName)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, subnetName, armnetwork.Subnet{
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: new("10.0.1.0/24"),
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

func createPrivateEndpointForBatch(ctx context.Context, client *armnetwork.PrivateEndpointsClient, resourceGroupName, peName, location, batchAccountID, vnetName, subnetName string) error {
	_, err := client.Get(ctx, resourceGroupName, peName, nil)
	if err == nil {
		log.Printf("Private endpoint %s already exists, skipping creation", peName)
		return nil
	}

	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		os.Getenv("AZURE_SUBSCRIPTION_ID"), resourceGroupName, vnetName, subnetName)

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, peName, armnetwork.PrivateEndpoint{
		Location: new(location),
		Properties: &armnetwork.PrivateEndpointProperties{
			Subnet: &armnetwork.Subnet{
				ID: new(subnetID),
			},
			PrivateLinkServiceConnections: []*armnetwork.PrivateLinkServiceConnection{
				{
					Name: new(peName + "-connection"),
					Properties: &armnetwork.PrivateLinkServiceConnectionProperties{
						PrivateLinkServiceID: new(batchAccountID),
						GroupIDs:             []*string{new("batchAccount")},
					},
				},
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Private endpoint %s already exists (conflict), skipping creation", peName)
			return nil
		}
		return fmt.Errorf("failed to begin creating private endpoint: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create private endpoint: %w", err)
	}

	log.Printf("Private endpoint %s created successfully", peName)
	return nil
}

func waitForBatchPrivateEndpointConnection(ctx context.Context, client *armbatch.PrivateEndpointConnectionClient, resourceGroupName, accountName string) (string, error) {
	maxAttempts := 30
	pollInterval := 10 * time.Second

	log.Printf("Waiting for private endpoint connection on batch account %s...", accountName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pager := client.NewListByBatchAccountPager(resourceGroupName, accountName, nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				log.Printf("Error listing private endpoint connections (attempt %d/%d): %v", attempt, maxAttempts, err)
				break
			}
			for _, conn := range page.Value {
				if conn != nil && conn.Name != nil {
					log.Printf("Found private endpoint connection: %s", *conn.Name)
					return *conn.Name, nil
				}
			}
		}
		log.Printf("No private endpoint connections found yet (attempt %d/%d), waiting...", attempt, maxAttempts)
		time.Sleep(pollInterval)
	}

	return "", fmt.Errorf("timeout waiting for private endpoint connection on batch account %s", accountName)
}

func deletePrivateEndpointForBatch(ctx context.Context, client *armnetwork.PrivateEndpointsClient, resourceGroupName, peName string) error {
	log.Printf("Deleting private endpoint %s...", peName)

	poller, err := client.BeginDelete(ctx, resourceGroupName, peName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Private endpoint %s not found, skipping deletion", peName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting private endpoint: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete private endpoint: %w", err)
	}

	log.Printf("Private endpoint %s deleted successfully", peName)
	return nil
}

func deleteSubnetForBatchPEC(ctx context.Context, client *armnetwork.SubnetsClient, resourceGroupName, vnetName, subnetName string) error {
	log.Printf("Deleting subnet %s...", subnetName)

	poller, err := client.BeginDelete(ctx, resourceGroupName, vnetName, subnetName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Subnet %s not found, skipping deletion", subnetName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting subnet: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete subnet: %w", err)
	}

	log.Printf("Subnet %s deleted successfully", subnetName)
	return nil
}

func deleteVNetForBatchPEC(ctx context.Context, client *armnetwork.VirtualNetworksClient, resourceGroupName, vnetName string) error {
	log.Printf("Deleting VNet %s...", vnetName)

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
