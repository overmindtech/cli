package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
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
	integrationTestFlowLogName        = "ovm-integ-test-flow-log"
	integrationTestFlowLogNSGName     = "ovm-integ-test-flow-log-nsg"
	integrationTestFlowLogStorageName = "ovmintegflowlogstor"
	integrationTestNetworkWatcherName = "NetworkWatcher_westus2"
	integrationTestNetworkWatcherRG   = "NetworkWatcherRG"
)

func TestNetworkFlowLogIntegration(t *testing.T) {
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

	nsgClient, err := armnetwork.NewSecurityGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create NSG client: %v", err)
	}

	storageClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Storage Accounts client: %v", err)
	}

	flowLogsSDKClient, err := armnetwork.NewFlowLogsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Flow Logs client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createResourceGroup(ctx, rgClient, integrationTestNetworkWatcherRG, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create NetworkWatcherRG: %v", err)
		}

		err = createFlowLogNSG(ctx, nsgClient, integrationTestResourceGroup, integrationTestFlowLogNSGName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create NSG: %v", err)
		}

		err = createFlowLogStorageAccount(ctx, storageClient, integrationTestResourceGroup, integrationTestFlowLogStorageName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create storage account: %v", err)
		}

		err = waitForFlowLogStorageAccountAvailable(ctx, storageClient, integrationTestResourceGroup, integrationTestFlowLogStorageName)
		if err != nil {
			t.Fatalf("Failed waiting for storage account: %v", err)
		}

		nsgID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s",
			subscriptionID, integrationTestResourceGroup, integrationTestFlowLogNSGName)
		storageID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s",
			subscriptionID, integrationTestResourceGroup, integrationTestFlowLogStorageName)

		err = createFlowLog(ctx, flowLogsSDKClient, integrationTestNetworkWatcherRG, integrationTestNetworkWatcherName, integrationTestFlowLogName, nsgID, storageID, integrationTestLocation)
		if err != nil {
			if strings.Contains(err.Error(), "NsgFlowLogCreationBlocked") {
				t.Skipf("Skipping: Azure has retired new NSG flow log creation: %v", err)
			}
			t.Fatalf("Failed to create flow log: %v", err)
		}

		err = waitForFlowLogAvailable(ctx, flowLogsSDKClient, integrationTestNetworkWatcherRG, integrationTestNetworkWatcherName, integrationTestFlowLogName)
		if err != nil {
			t.Fatalf("Failed waiting for flow log: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		ctx := t.Context()
		_, checkErr := flowLogsSDKClient.Get(ctx, integrationTestNetworkWatcherRG, integrationTestNetworkWatcherName, integrationTestFlowLogName, nil)
		if checkErr != nil {
			var respErr *azcore.ResponseError
			if errors.As(checkErr, &respErr) && respErr.StatusCode == http.StatusNotFound {
				t.Skipf("Flow log %s does not exist (Setup may have been skipped). Skipping Run tests.", integrationTestFlowLogName)
			}
			t.Fatalf("Failed preflight check for flow log %s: %v", integrationTestFlowLogName, checkErr)
		}

		t.Run("GetFlowLog", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkFlowLog(
				clients.NewFlowLogsClient(flowLogsSDKClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestNetworkWatcherRG)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestNetworkWatcherName, integrationTestFlowLogName)
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

			expectedUnique := shared.CompositeLookupKey(integrationTestNetworkWatcherName, integrationTestFlowLogName)
			if uniqueAttrValue != expectedUnique {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved flow log %s", integrationTestFlowLogName)
		})

		t.Run("SearchFlowLogs", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkFlowLog(
				clients.NewFlowLogsClient(flowLogsSDKClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestNetworkWatcherRG)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestNetworkWatcherName, true)
			if err != nil {
				t.Fatalf("Failed to search flow logs: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one flow log, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == shared.CompositeLookupKey(integrationTestNetworkWatcherName, integrationTestFlowLogName) {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find flow log %s in the search results", integrationTestFlowLogName)
			}

			log.Printf("Found %d flow logs in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkFlowLog(
				clients.NewFlowLogsClient(flowLogsSDKClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestNetworkWatcherRG)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestNetworkWatcherName, integrationTestFlowLogName)
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
					t.Errorf("Linked item query of type %s has empty Query", q.GetType())
				}
				if q.GetScope() == "" {
					t.Errorf("Linked item query of type %s has empty Scope", q.GetType())
				}
				method := q.GetMethod()
				if method != 1 && method != 2 { // GET=1, SEARCH=2
					t.Errorf("Linked item query of type %s has unexpected Method %d", q.GetType(), method)
				}
			}

			log.Printf("Verified %d linked item queries for flow log %s", len(linkedQueries), integrationTestFlowLogName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewNetworkFlowLog(
				clients.NewFlowLogsClient(flowLogsSDKClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestNetworkWatcherRG)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(integrationTestNetworkWatcherName, integrationTestFlowLogName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.NetworkFlowLog.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkFlowLog.String(), sdpItem.GetType())
			}

			expectedScope := subscriptionID + "." + integrationTestNetworkWatcherRG
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

		err := deleteFlowLog(ctx, flowLogsSDKClient, integrationTestNetworkWatcherRG, integrationTestNetworkWatcherName, integrationTestFlowLogName)
		if err != nil {
			t.Fatalf("Failed to delete flow log: %v", err)
		}

		err = deleteFlowLogStorageAccount(ctx, storageClient, integrationTestResourceGroup, integrationTestFlowLogStorageName)
		if err != nil {
			t.Fatalf("Failed to delete storage account: %v", err)
		}

		err = deleteFlowLogNSG(ctx, nsgClient, integrationTestResourceGroup, integrationTestFlowLogNSGName)
		if err != nil {
			t.Fatalf("Failed to delete NSG: %v", err)
		}
	})
}

func createFlowLogNSG(ctx context.Context, client *armnetwork.SecurityGroupsClient, rg, name, location string) error {
	_, err := client.Get(ctx, rg, name, nil)
	if err == nil {
		log.Printf("NSG %s already exists, skipping creation", name)
		return nil
	}

	poller, err := client.BeginCreateOrUpdate(ctx, rg, name, armnetwork.SecurityGroup{
		Location: &location,
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("NSG %s already exists (conflict), skipping", name)
			return nil
		}
		return fmt.Errorf("failed to create NSG: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create NSG: %w", err)
	}
	log.Printf("NSG %s created successfully", name)
	return nil
}

func createFlowLogStorageAccount(ctx context.Context, client *armstorage.AccountsClient, rg, name, location string) error {
	_, err := client.GetProperties(ctx, rg, name, nil)
	if err == nil {
		log.Printf("Storage account %s already exists, skipping creation", name)
		return nil
	}

	poller, err := client.BeginCreate(ctx, rg, name, armstorage.AccountCreateParameters{
		Location: &location,
		Kind:     new(armstorage.KindStorageV2),
		SKU: &armstorage.SKU{
			Name: new(armstorage.SKUNameStandardLRS),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Storage account %s already exists (conflict), skipping", name)
			return nil
		}
		return fmt.Errorf("failed to create storage account: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create storage account: %w", err)
	}
	log.Printf("Storage account %s created successfully", name)
	return nil
}

func waitForFlowLogStorageAccountAvailable(ctx context.Context, client *armstorage.AccountsClient, rg, name string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.GetProperties(ctx, rg, name, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking storage account: %w", err)
		}
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && *resp.Properties.ProvisioningState == armstorage.ProvisioningStateSucceeded {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for storage account %s", name)
}

func createFlowLog(ctx context.Context, client *armnetwork.FlowLogsClient, rg, networkWatcherName, flowLogName, nsgID, storageID, location string) error {
	_, err := client.Get(ctx, rg, networkWatcherName, flowLogName, nil)
	if err == nil {
		log.Printf("Flow log %s already exists, skipping creation", flowLogName)
		return nil
	}

	enabled := true
	poller, err := client.BeginCreateOrUpdate(ctx, rg, networkWatcherName, flowLogName, armnetwork.FlowLog{
		Location: &location,
		Properties: &armnetwork.FlowLogPropertiesFormat{
			TargetResourceID: &nsgID,
			StorageID:        &storageID,
			Enabled:          &enabled,
			RetentionPolicy: &armnetwork.RetentionPolicyParameters{
				Enabled: &enabled,
				Days:    new(int32(7)),
			},
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Flow log %s already exists (conflict), skipping", flowLogName)
			return nil
		}
		return fmt.Errorf("failed to create flow log: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create flow log: %w", err)
	}
	log.Printf("Flow log %s created successfully", flowLogName)
	return nil
}

func waitForFlowLogAvailable(ctx context.Context, client *armnetwork.FlowLogsClient, rg, networkWatcherName, flowLogName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, rg, networkWatcherName, flowLogName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking flow log: %w", err)
		}
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil && string(*resp.Properties.ProvisioningState) == "Succeeded" {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for flow log %s", flowLogName)
}

func deleteFlowLog(ctx context.Context, client *armnetwork.FlowLogsClient, rg, networkWatcherName, flowLogName string) error {
	poller, err := client.BeginDelete(ctx, rg, networkWatcherName, flowLogName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Flow log %s not found, skipping deletion", flowLogName)
			return nil
		}
		return fmt.Errorf("failed to delete flow log: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete flow log: %w", err)
	}
	log.Printf("Flow log %s deleted successfully", flowLogName)
	return nil
}

func deleteFlowLogStorageAccount(ctx context.Context, client *armstorage.AccountsClient, rg, name string) error {
	_, err := client.Delete(ctx, rg, name, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Storage account %s not found, skipping deletion", name)
			return nil
		}
		return fmt.Errorf("failed to delete storage account: %w", err)
	}
	log.Printf("Storage account %s deleted successfully", name)
	return nil
}

func deleteFlowLogNSG(ctx context.Context, client *armnetwork.SecurityGroupsClient, rg, name string) error {
	poller, err := client.BeginDelete(ctx, rg, name, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("NSG %s not found, skipping deletion", name)
			return nil
		}
		return fmt.Errorf("failed to delete NSG: %w", err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete NSG: %w", err)
	}
	log.Printf("NSG %s deleted successfully", name)
	return nil
}
