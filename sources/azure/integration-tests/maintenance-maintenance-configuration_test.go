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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/maintenance/armmaintenance"
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
	integrationTestMaintenanceConfigName = "ovm-integ-test-maint-config"
)

func TestMaintenanceMaintenanceConfigurationIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	configurationsClient, err := armmaintenance.NewConfigurationsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Maintenance Configurations client: %v", err)
	}

	configurationsForResourceGroupClient, err := armmaintenance.NewConfigurationsForResourceGroupClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Maintenance Configurations For Resource Group client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createMaintenanceConfig(ctx, configurationsClient, integrationTestResourceGroup, integrationTestMaintenanceConfigName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create maintenance configuration: %v", err)
		}

		err = waitForMaintenanceConfigAvailable(ctx, configurationsClient, integrationTestResourceGroup, integrationTestMaintenanceConfigName)
		if err != nil {
			t.Fatalf("Failed waiting for maintenance configuration to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetMaintenanceConfiguration", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving maintenance configuration %s in subscription %s, resource group %s",
				integrationTestMaintenanceConfigName, subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewMaintenanceMaintenanceConfiguration(
				clients.NewMaintenanceConfigurationClient(configurationsClient, configurationsForResourceGroupClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestMaintenanceConfigName, true)
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

			if uniqueAttrValue != integrationTestMaintenanceConfigName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestMaintenanceConfigName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved maintenance configuration %s", integrationTestMaintenanceConfigName)
		})

		t.Run("ListMaintenanceConfigurations", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing maintenance configurations in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewMaintenanceMaintenanceConfiguration(
				clients.NewMaintenanceConfigurationClient(configurationsClient, configurationsForResourceGroupClient),
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
				t.Fatalf("Failed to list maintenance configurations: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one maintenance configuration, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestMaintenanceConfigName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find maintenance configuration %s in the list", integrationTestMaintenanceConfigName)
			}

			log.Printf("Found %d maintenance configurations in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for maintenance configuration %s", integrationTestMaintenanceConfigName)

			wrapper := manual.NewMaintenanceMaintenanceConfiguration(
				clients.NewMaintenanceConfigurationClient(configurationsClient, configurationsForResourceGroupClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestMaintenanceConfigName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.MaintenanceMaintenanceConfiguration.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.MaintenanceMaintenanceConfiguration, sdpItem.GetType())
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

			log.Printf("Verified item attributes for maintenance configuration %s", integrationTestMaintenanceConfigName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for maintenance configuration %s", integrationTestMaintenanceConfigName)

			wrapper := manual.NewMaintenanceMaintenanceConfiguration(
				clients.NewMaintenanceConfigurationClient(configurationsClient, configurationsForResourceGroupClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, integrationTestMaintenanceConfigName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for maintenance configuration %s", len(linkedQueries), integrationTestMaintenanceConfigName)

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
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

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope())
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteMaintenanceConfig(ctx, configurationsClient, integrationTestResourceGroup, integrationTestMaintenanceConfigName)
		if err != nil {
			t.Fatalf("Failed to delete maintenance configuration: %v", err)
		}
	})
}

func createMaintenanceConfig(ctx context.Context, client *armmaintenance.ConfigurationsClient, resourceGroupName, configName, location string) error {
	_, err := client.Get(ctx, resourceGroupName, configName, nil)
	if err == nil {
		log.Printf("Maintenance configuration %s already exists, skipping creation", configName)
		return nil
	}

	maintenanceScope := armmaintenance.MaintenanceScopeHost
	visibility := armmaintenance.VisibilityCustom

	_, err = client.CreateOrUpdate(ctx, resourceGroupName, configName, armmaintenance.Configuration{
		Location: &location,
		Properties: &armmaintenance.ConfigurationProperties{
			MaintenanceScope: &maintenanceScope,
			Visibility:       &visibility,
			MaintenanceWindow: &armmaintenance.Window{
				StartDateTime: new("2025-01-01 00:00"),
				Duration:      new("02:00"),
				TimeZone:      new("Pacific Standard Time"),
				RecurEvery:    new("Day"),
			},
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("maintenance-configuration"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Maintenance configuration %s already exists (conflict), skipping creation", configName)
			return nil
		}
		return fmt.Errorf("failed to create maintenance configuration: %w", err)
	}

	log.Printf("Maintenance configuration %s created successfully", configName)
	return nil
}

func waitForMaintenanceConfigAvailable(ctx context.Context, client *armmaintenance.ConfigurationsClient, resourceGroupName, configName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for maintenance configuration %s to be available via API...", configName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := client.Get(ctx, resourceGroupName, configName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Maintenance configuration %s not yet available (attempt %d/%d), waiting %v...", configName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking maintenance configuration availability: %w", err)
		}

		log.Printf("Maintenance configuration %s is available", configName)
		return nil
	}

	return fmt.Errorf("timeout waiting for maintenance configuration %s to be available after %d attempts", configName, maxAttempts)
}

func deleteMaintenanceConfig(ctx context.Context, client *armmaintenance.ConfigurationsClient, resourceGroupName, configName string) error {
	_, err := client.Delete(ctx, resourceGroupName, configName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Maintenance configuration %s not found, skipping deletion", configName)
			return nil
		}
		return fmt.Errorf("failed to delete maintenance configuration: %w", err)
	}

	log.Printf("Maintenance configuration %s deleted successfully", configName)
	return nil
}
