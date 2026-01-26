package proc

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"

	// TODO: Uncomment when Azure dynamic adapters are implemented
	// "github.com/overmindtech/cli/sources/azure/dynamic"
	// _ "github.com/overmindtech/cli/sources/azure/dynamic/adapters" // Import all adapters to register them
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

// Metadata contains the metadata for the Azure source
var Metadata = sdp.AdapterMetadataList{}

// AzureConfig holds configuration for Azure source.
// The YAML tags match the keys used in the source.yaml config file.
type AzureConfig struct {
	SubscriptionID string   `yaml:"azure-subscription-id"`
	TenantID       string   `yaml:"azure-tenant-id"`
	ClientID       string   `yaml:"azure-client-id"`
	Regions        []string `yaml:"azure-regions"`
}

func init() {
	// Register the Azure source metadata for documentation purposes
	ctx := context.Background()

	// subscription, regions are just placeholders here
	// They are not used in the metadata content
	discoveryAdapters, err := adapters(
		ctx,
		"subscription",
		"tenant",
		"client",
		[]string{"region"},
		nil, // No credentials needed for metadata registration
		nil,
		false,
		sdpcache.NewNoOpCache(), // no-op cache for metadata registration
	)
	if err != nil {
		panic(fmt.Errorf("error creating adapters: %w", err))
	}

	for _, adapter := range discoveryAdapters {
		Metadata.Register(adapter.Metadata())
	}

	log.Debug("Registered Azure source metadata", " with ", len(Metadata.AllAdapterMetadata()), " adapters")
}

func Initialize(ctx context.Context, ec *discovery.EngineConfig, cfg *AzureConfig) (*discovery.Engine, error) {
	engine, err := discovery.NewEngine(ec)
	if err != nil {
		return nil, fmt.Errorf("error initializing Engine: %w", err)
	}

	// ReadinessCheck verifies adapters are healthy by using a StorageAccount adapter
	// Timeout is handled by SendHeartbeat, HTTP handlers rely on request context
	engine.SetReadinessCheck(func(ctx context.Context) error {
		// Find a StorageAccount adapter to verify adapter health
		adapters := engine.AdaptersByType("azure-storage-account")
		if len(adapters) == 0 {
			return fmt.Errorf("readiness check failed: no azure-storage-account adapters available")
		}
		// Use first adapter and try to list from first scope
		adapter := adapters[0]
		scopes := adapter.Scopes()
		if len(scopes) == 0 {
			return fmt.Errorf("readiness check failed: no scopes available for azure-storage-account adapter")
		}
		listableAdapter, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			return fmt.Errorf("readiness check failed: azure-storage-account adapter is not listable")
		}
		_, err := listableAdapter.List(ctx, scopes[0], true)
		if err != nil {
			return fmt.Errorf("readiness check (listing storage accounts) failed: %w", err)
		}
		return nil
	})

	// Create a shared cache for all adapters in this source
	sharedCache := sdpcache.NewCache(ctx)

	err = func() error {
		var logmsg string
		// Use provided config, otherwise fall back to viper
		if cfg != nil {
			logmsg = "Using directly provided config"
		} else {
			var err error
			cfg, err = readConfig()
			if err != nil {
				return fmt.Errorf("error creating config from command line: %w", err)
			}
			logmsg = "Using config from viper"

		}
		log.WithFields(log.Fields{
			"ovm.source.type":            "azure",
			"ovm.source.subscription_id": cfg.SubscriptionID,
			"ovm.source.tenant_id":       cfg.TenantID,
			"ovm.source.client_id":       cfg.ClientID,
			"ovm.source.regions":         cfg.Regions,
		}).Info(logmsg)

		// Regions are optional for Azure, but subscription ID is required
		if cfg.SubscriptionID == "" {
			return fmt.Errorf("Azure source must specify subscription ID")
		}

		// Set Azure SDK environment variables from viper config if not already set.
		// The Azure SDK's DefaultAzureCredential reads AZURE_CLIENT_ID and AZURE_TENANT_ID
		// directly from environment variables for federated authentication.
		//
		// When using Azure Workload Identity webhook, these env vars are already injected
		// by the webhook, so we only set them if they're not present. This supports both:
		// 1. Azure Workload Identity webhook (env vars already injected)
		// 2. Manual configuration (env vars set from viper config)
		//
		// Reference: https://azure.github.io/azure-workload-identity/docs/
		if os.Getenv("AZURE_CLIENT_ID") == "" && cfg.ClientID != "" {
			os.Setenv("AZURE_CLIENT_ID", cfg.ClientID)
		}
		if os.Getenv("AZURE_TENANT_ID") == "" && cfg.TenantID != "" {
			os.Setenv("AZURE_TENANT_ID", cfg.TenantID)
		}

		// Initialize Azure credentials
		cred, err := azureshared.NewAzureCredential(ctx)
		if err != nil {
			return fmt.Errorf("error creating Azure credentials: %w", err)
		}

		// TODO: Implement linker when Azure dynamic adapters are available
		var linker interface{} = nil

		discoveryAdapters, err := adapters(ctx, cfg.SubscriptionID, cfg.TenantID, cfg.ClientID, cfg.Regions, cred, linker, true, sharedCache)
		if err != nil {
			return fmt.Errorf("error creating discovery adapters: %w", err)
		}

		// Verify subscription access before adding adapters
		err = checkSubscriptionAccess(ctx, cfg.SubscriptionID, cred)
		if err != nil {
			log.WithContext(ctx).WithError(err).WithFields(log.Fields{
				"ovm.source.type":            "azure",
				"ovm.source.subscription_id": cfg.SubscriptionID,
			}).Error("Permission check failed for subscription")
		} else {
			log.WithContext(ctx).WithFields(log.Fields{
				"ovm.source.type":            "azure",
				"ovm.source.subscription_id": cfg.SubscriptionID,
			}).Info("Permission check passed for subscription")
		}

		// Add the adapters to the engine
		err = engine.AddAdapters(discoveryAdapters...)
		if err != nil {
			return fmt.Errorf("error adding adapters to engine: %w", err)
		}

		return nil
	}()

	if err != nil {
		log.WithError(err).Debug("Error initializing Azure source")
		return nil, fmt.Errorf("error initializing Azure source: %w", err)
	}

	// Start sending heartbeats after adapters are successfully added
	// This ensures the first heartbeat has adapters available for readiness checks
	engine.StartSendingHeartbeats(ctx)
	brokenHeart := engine.SendHeartbeat(ctx, nil) // Send the error immediately through the custom health check func
	if brokenHeart != nil {
		log.WithError(brokenHeart).Error("Error sending heartbeat")
	}

	log.Debug("Sources initialized")
	// If there is no error then return the engine
	return engine, nil
}

func readConfig() (*AzureConfig, error) {
	subscriptionID := viper.GetString("azure-subscription-id")
	if subscriptionID == "" {
		return nil, fmt.Errorf("azure-subscription-id not set")
	}

	tenantID := viper.GetString("azure-tenant-id")
	if tenantID == "" {
		return nil, fmt.Errorf("azure-tenant-id not set")
	}

	clientID := viper.GetString("azure-client-id")
	if clientID == "" {
		return nil, fmt.Errorf("azure-client-id not set")
	}

	l := &AzureConfig{
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		ClientID:       clientID,
	}

	// Regions are optional for Azure
	regions := viper.GetStringSlice("azure-regions")
	if len(regions) > 0 {
		l.Regions = regions
	}

	return l, nil
}

// adapters returns a list of discovery adapters for Azure
// It includes both manual adapters and dynamic adapters.
func adapters(
	ctx context.Context,
	subscriptionID string,
	tenantID string,
	clientID string,
	regions []string,
	cred *azidentity.DefaultAzureCredential,
	linker interface{}, // TODO: Use *azureshared.Linker when azureshared package is fully implemented
	initAzureClients bool,
	cache sdpcache.Cache,
) ([]discovery.Adapter, error) {
	discoveryAdapters := make([]discovery.Adapter, 0)

	// Add manual adapters
	manualAdapters, err := manual.Adapters(
		ctx,
		subscriptionID,
		regions,
		cred,
		initAzureClients,
		cache,
	)
	if err != nil {
		return nil, err
	}

	initiatedManualAdapters := make(map[string]bool)
	for _, adapter := range manualAdapters {
		initiatedManualAdapters[adapter.Type()] = true
	}

	discoveryAdapters = append(discoveryAdapters, manualAdapters...)

	// TODO: Add dynamic adapters when Azure dynamic adapter framework is implemented
	// dynamicAdapters, err := dynamic.Adapters(
	// 	subscriptionID,
	// 	tenantID,
	// 	clientID,
	// 	regions,
	// 	linker,
	// 	httpClient,
	// 	initiatedManualAdapters,
	// )
	// if err != nil {
	// 	return nil, err
	// }
	// discoveryAdapters = append(discoveryAdapters, dynamicAdapters...)

	_ = tenantID // Used for metadata/logging
	_ = clientID // Used for metadata/logging

	return discoveryAdapters, nil
}

// checkSubscriptionAccess verifies that the credentials have access to the specified subscription
func checkSubscriptionAccess(ctx context.Context, subscriptionID string, cred *azidentity.DefaultAzureCredential) error {
	// Create a resource groups client to test subscription access
	client, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create resource groups client: %w", err)
	}

	// Try to list resource groups to verify access
	pager := client.NewListPager(nil)
	if !pager.More() {
		// No resource groups, but that's okay - we just want to verify we can access the subscription
		log.WithField("ovm.source.subscription_id", subscriptionID).Info("Successfully verified subscription access (no resource groups found)")
		return nil
	}

	// Try to get the first page to verify we have access
	_, err = pager.NextPage(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify subscription access: %w", err)
	}

	log.WithField("ovm.source.subscription_id", subscriptionID).Info("Successfully verified subscription access")
	return nil
}
