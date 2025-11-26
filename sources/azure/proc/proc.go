package proc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"

	// TODO: Uncomment when Azure dynamic adapters are implemented
	// "github.com/overmindtech/cli/sources/azure/dynamic"
	// _ "github.com/overmindtech/cli/sources/azure/dynamic/adapters" // Import all adapters to register them
	"github.com/overmindtech/cli/sources/azure/manual"
	// TODO: Uncomment when azureshared.Linker is implemented
	// azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

// Metadata contains the metadata for the Azure source
var Metadata = sdp.AdapterMetadataList{}

// AzureConfig holds configuration for Azure source
type AzureConfig struct {
	SubscriptionID string
	TenantID       string
	ClientID       string
	Regions        []string
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
		nil,
		false,
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

	var permissionCheck func() error

	var startupErrorMutex sync.Mutex
	startupError := errors.New("source is starting")
	if ec.HeartbeatOptions != nil {
		ec.HeartbeatOptions.HealthCheck = func(_ context.Context) error {
			startupErrorMutex.Lock()
			defer startupErrorMutex.Unlock()
			if startupError != nil {
				// If there is a startup error, return it
				return startupError
			}

			if permissionCheck != nil {
				// If the permission check is set, run it
				return permissionCheck()
			}
			return nil
		}
	}

	engine.StartSendingHeartbeats(ctx)

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

		// TODO: Implement linker when Azure dynamic adapters are available in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
		var linker interface{} = nil

		discoveryAdapters, err := adapters(ctx, cfg.SubscriptionID, cfg.TenantID, cfg.ClientID, cfg.Regions, linker, true)
		if err != nil {
			return fmt.Errorf("error creating discovery adapters: %w", err)
		}

		// TODO: Add permission check for Azure subscription access in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
		// This would verify that the credentials can access the subscription
		permissionCheck = func() error {
			// For now, we'll skip the permission check until we have a subscription adapter
			// In the future, we can add a check similar to GCP's project check
			return nil
		}

		err = permissionCheck()
		if err != nil {
			return fmt.Errorf("error checking permissions: %w", err)
		}

		// Add the adapters to the engine
		err = engine.AddAdapters(discoveryAdapters...)
		if err != nil {
			return fmt.Errorf("error adding adapters to engine: %w", err)
		}

		return nil
	}()

	startupErrorMutex.Lock()
	startupError = err
	startupErrorMutex.Unlock()
	brokenHeart := engine.SendHeartbeat(ctx, nil) // Send the error immediately through the custom health check func
	if brokenHeart != nil {
		log.WithError(brokenHeart).Error("Error sending heartbeat")
	}

	if err != nil {
		log.WithError(err).Debug("Error initializing Azure source")

		return nil, fmt.Errorf("error initializing Azure source: %w", err)
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
	linker interface{}, // TODO: Use *azureshared.Linker when azureshared package is fully implemented in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
	initAzureClients bool,
) ([]discovery.Adapter, error) {
	discoveryAdapters := make([]discovery.Adapter, 0)

	// Add manual adapters
	// Note: manual.Adapters currently uses projectID parameter name but accepts subscriptionID
	manualAdapters, err := manual.Adapters(
		ctx,
		subscriptionID, // passed as projectID parameter (will be updated in manual.Adapters)
		regions,
		nil, // zones not used in Azure
		nil, // tokenSource not used with federated credentials
		initAzureClients,
	)
	if err != nil {
		return nil, err
	}

	initiatedManualAdapters := make(map[string]bool)
	for _, adapter := range manualAdapters {
		initiatedManualAdapters[adapter.Type()] = true
	}

	discoveryAdapters = append(discoveryAdapters, manualAdapters...)

	// Azure SDK handles authentication automatically via federated credentials
	// when running in Kubernetes/EKS with workload identity
	// For local development, Azure SDK will use Azure CLI or environment variables
	_ = initAzureClients // TODO: Use this when implementing Azure client initialization in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials

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

	return discoveryAdapters, nil
}
