package proc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Metadata contains the metadata for the GCP source
var Metadata = sdp.AdapterMetadataList{}

func init() {
	// Register the GCP source metadata for documentation purposes
	ctx := context.Background()

	// project, regions, and zones are just placeholders here
	// They are not used in the metadata content
	discoveryAdapters, err := adapters(
		ctx,
		"project",
		[]string{"region"},
		[]string{"zone"},
		nil,
		false,
	)
	if err != nil {
		panic(fmt.Errorf("error creating adapters: %w", err))
	}

	for _, adapter := range discoveryAdapters {
		Metadata.Register(adapter.Metadata())
	}

	log.Info("Registered GCP source metadata")
}

func Initialize(ctx context.Context, ec *discovery.EngineConfig) (*discovery.Engine, error) {
	engine, err := discovery.NewEngine(ec)
	if err != nil {
		return nil, fmt.Errorf("error initializing Engine: %w", err)
	}

	var startupErrorMutex sync.Mutex
	startupError := errors.New("source is starting")
	if ec.HeartbeatOptions != nil {
		ec.HeartbeatOptions.HealthCheck = func(_ context.Context) error {
			startupErrorMutex.Lock()
			defer startupErrorMutex.Unlock()
			return startupError
		}
	}

	engine.StartSendingHeartbeats(ctx)

	err = func() error {
		cfg, err := readConfig()
		if err != nil {
			return fmt.Errorf("error creating config: %w", err)
		}

		log.WithFields(log.Fields{
			"project_id": cfg.ProjectID,
			"regions":    cfg.Regions,
			"zones":      cfg.Zones,
		}).Info("Got config")

		linker := gcpshared.NewLinker()

		discoveryAdapters, err := adapters(ctx, cfg.ProjectID, cfg.Regions, cfg.Zones, linker, true)
		if err != nil {
			return fmt.Errorf("error creating discovery adapters: %w", err)
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
		log.WithError(err).Debug("Error initializing GCP source")

		return nil, fmt.Errorf("error initializing GCP source: %w", err)
	}

	log.Debug("Sources initialized")
	// If there is no error then return the engine
	return engine, nil
}

type config struct {
	ProjectID string
	Regions   []string
	Zones     []string
}

func readConfig() (*config, error) {
	projectID := viper.GetString("gcp-project-id")
	if projectID == "" {
		return nil, fmt.Errorf("gcp-project-id not set")
	}

	l := &config{
		ProjectID: projectID,
	}

	// TODO: In the future, we will try to get the zones via Search API
	// https://github.com/overmindtech/workspace/issues/1340

	zones := viper.GetStringSlice("gcp-zones")
	regions := viper.GetStringSlice("gcp-regions")
	if len(zones) == 0 && len(regions) == 0 {
		return nil, fmt.Errorf("need at least one gcp-zones or gcp-regions value")
	}

	uniqueRegions := make(map[string]bool)
	for _, region := range regions {
		uniqueRegions[region] = true
	}

	for _, zone := range zones {
		if zone == "" {
			return nil, fmt.Errorf("zone name is empty")
		}

		l.Zones = append(l.Zones, zone)

		region := gcpshared.ZoneToRegion(zone)
		if region == "" {
			return nil, fmt.Errorf("zone %s is not valid", zone)
		}

		uniqueRegions[region] = true
	}

	for region := range uniqueRegions {
		l.Regions = append(l.Regions, region)
	}

	return l, nil
}

// adapters returns a list of discovery adapters for GCP
// It includes both manual adapters and dynamic adapters.
func adapters(
	ctx context.Context,
	projectID string,
	regions []string,
	zones []string,
	linker *gcpshared.Linker,
	initGCPClients bool,
) ([]discovery.Adapter, error) {
	discoveryAdapters := make([]discovery.Adapter, 0)

	// Add manual adapters
	manualAdapters, err := manual.Adapters(
		ctx,
		projectID,
		regions,
		zones,
		initGCPClients,
	)
	if err != nil {
		return nil, err
	}

	initiatedManualAdapters := make(map[string]bool)
	for _, adapter := range manualAdapters {
		initiatedManualAdapters[adapter.Type()] = true
	}

	discoveryAdapters = append(discoveryAdapters, manualAdapters...)

	httpClient := http.DefaultClient
	if initGCPClients {
		var errCli error
		httpClient, errCli = gcpshared.GCPHTTPClientWithOtel()
		if errCli != nil {
			return nil, fmt.Errorf("error creating GCP HTTP client: %w", errCli)
		}
	}

	// Add dynamic adapters
	dynamicAdapters, err := dynamic.Adapters(
		projectID,
		regions,
		zones,
		linker,
		httpClient,
		initiatedManualAdapters,
	)
	if err != nil {
		return nil, err
	}

	discoveryAdapters = append(discoveryAdapters, dynamicAdapters...)

	return discoveryAdapters, nil
}
