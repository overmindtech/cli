package proc

import (
	"context"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Metadata contains the metadata for the GCP source
var Metadata = sdp.AdapterMetadataList{}

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

	// Register adapters metadata
	for _, adapter := range adapters {
		Metadata.Register(adapter.Metadata())
	}

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
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("GCP_PROJECT_ID environment variable not set")
	}

	l := &config{
		ProjectID: projectID,
	}

	// TODO: In the future, we will try to get the zones via Search API
	// https://github.com/overmindtech/workspace/issues/1340

	zonesEnv := os.Getenv("GCP_ZONES")
	if zonesEnv == "" {
		return nil, fmt.Errorf("GCP_ZONES environment variable not set")
	}

	regions := make(map[string]bool)
	for _, zone := range strings.Split(zonesEnv, ",") {
		if zone == "" {
			return nil, fmt.Errorf("zone name is empty")
		}

		l.Zones = append(l.Zones, zone)

		region := gcpshared.ZoneToRegion(zone)
		if region == "" {
			return nil, fmt.Errorf("zone %s is not valid", zone)
		}

		regions[region] = true
	}

	for region := range regions {
		l.Regions = append(l.Regions, region)
	}

	return l, nil
}
