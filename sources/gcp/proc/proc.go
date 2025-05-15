package proc

import (
	"context"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/adapters"
)

func Initialize(ctx context.Context, ec *discovery.EngineConfig) (*discovery.Engine, error) {
	engine, err := discovery.NewEngine(ec)
	if err != nil {
		return nil, fmt.Errorf("error initializing Engine: %w", err)
	}

	l, err := makeLocations()
	if err != nil {
		return nil, fmt.Errorf("error creating locations: %w", err)
	}

	log.WithFields(log.Fields{
		"project_id": l.ProjectID,
		"regions":    l.Regions,
		"zones":      l.Zones,
	}).Info("Got locations")

	adapters, err := adapters.Adapters(ctx, l.ProjectID, l.Regions, l.Zones)
	if err != nil {
		return nil, fmt.Errorf("error creating adapters: %w", err)
	}

	// Add the adapters to the engine
	err = engine.AddAdapters(adapters...)
	if err != nil {
		return nil, fmt.Errorf("error adding adapters to engine: %w", err)
	}

	return engine, nil
}

type locations struct {
	ProjectID string
	Regions   []string
	Zones     []string
}

func makeLocations() (*locations, error) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("GCP_PROJECT_ID environment variable not set")
	}

	l := &locations{
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

		region := zoneToRegion(zone)
		if region == "" {
			return nil, fmt.Errorf("zone %s is not valid", zone)
		}

		regions[region] = true
	}

	return l, nil
}

// zoneToRegion converts a GCP zone to a region.
// The fully-qualified name for a zone is made up of <region>-<zone>.
// For example, the fully qualified name for zone a in region us-central1 is us-central1-a.
// https://cloud.google.com/compute/docs/regions-zones#identifying_a_region_or_zone
func zoneToRegion(zone string) string {
	parts := strings.Split(zone, "-")
	if len(parts) < 2 {
		return ""
	}

	return strings.Join(parts[:len(parts)-1], "-")
}
