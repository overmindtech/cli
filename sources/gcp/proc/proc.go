package proc

import (
	"context"
	"fmt"

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
	manualAdapters, err := manual.Adapters(ctx, "project", []string{"region"}, []string{"zone"})
	if err != nil {
		panic(fmt.Errorf("error creating manual adapters: %w", err))
	}

	for _, adapter := range manualAdapters {
		Metadata.Register(adapter.Metadata())
	}

	initiatedManualAdapters := make(map[string]bool)
	for _, adapter := range manualAdapters {
		initiatedManualAdapters[adapter.Type()] = true
	}

	dynamicAdapters, err := dynamic.Adapters(
		"project",
		[]string{"region"},
		[]string{"zone"},
		nil,
		nil,
		initiatedManualAdapters,
	)
	if err != nil {
		panic(fmt.Errorf("error creating dynamic adapters: %w", err))
	}

	for _, adapter := range dynamicAdapters {
		Metadata.Register(adapter.Metadata())
	}

	log.Info("Registered GCP source metadata")
}

func Initialize(ctx context.Context, ec *discovery.EngineConfig) (*discovery.Engine, error) {
	engine, err := discovery.NewEngine(ec)
	if err != nil {
		return nil, fmt.Errorf("error initializing Engine: %w", err)
	}

	cfg, err := readConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating config: %w", err)
	}

	log.WithFields(log.Fields{
		"project_id": cfg.ProjectID,
		"regions":    cfg.Regions,
		"zones":      cfg.Zones,
	}).Info("Got config")

	linker := gcpshared.NewLinker()

	manualAdapters, err := manual.Adapters(ctx, cfg.ProjectID, cfg.Regions, cfg.Zones)
	if err != nil {
		return nil, fmt.Errorf("error creating manual adapters: %w", err)
	}

	initiatedManualAdapters := make(map[string]bool)
	for _, adapter := range manualAdapters {
		initiatedManualAdapters[adapter.Type()] = true
	}

	gcpHTTPCliWithOtel, err := gcpshared.GCPHTTPClientWithOtel()
	if err != nil {
		return nil, err
	}

	dynamicAdapters, err := dynamic.Adapters(cfg.ProjectID, cfg.Regions, cfg.Zones, linker, gcpHTTPCliWithOtel, initiatedManualAdapters)
	if err != nil {
		return nil, fmt.Errorf("error creating dynamic adapters: %w", err)
	}

	var adapters []discovery.Adapter
	adapters = append(adapters, manualAdapters...)
	adapters = append(adapters, dynamicAdapters...)

	// Add the adapters to the engine
	err = engine.AddAdapters(adapters...)
	if err != nil {
		return nil, fmt.Errorf("error adding adapters to engine: %w", err)
	}

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
