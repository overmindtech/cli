package cmd

import (
	"reflect"
	"testing"

	gcpproc "github.com/overmindtech/cli/sources/gcp/proc"
)

func TestUnifiedGCPConfigs(t *testing.T) {
	t.Run("Multiple configs with different project IDs - no unification", func(t *testing.T) {
		configs := []*gcpproc.GCPConfig{
			{
				ProjectID: "project-1",
				Regions:   []string{"us-central1", "us-east1"},
				Zones:     []string{"us-central1-a", "us-east1-a"},
			},
			{
				ProjectID: "project-2",
				Regions:   []string{"us-central1", "us-east1"},
				Zones:     []string{"us-central1-a", "us-east1-a"},
			},
			{
				ProjectID: "project-3",
				Regions:   []string{"europe-west1"},
				Zones:     []string{"europe-west1-b"},
			},
		}

		result := unifiedGCPConfigs(configs)

		// Should have 3 configs (no unification since all project IDs are different)
		if len(result) != 3 {
			t.Fatalf("Expected 3 configs, got %d", len(result))
		}

		// Verify each project ID appears exactly once
		projectIDs := make(map[string]int)
		for _, config := range result {
			projectIDs[config.ProjectID]++
		}

		expectedProjects := []string{"project-1", "project-2", "project-3"}
		for _, projectID := range expectedProjects {
			if count, exists := projectIDs[projectID]; !exists || count != 1 {
				t.Fatalf("Expected project %s to appear exactly once, got %d", projectID, count)
			}
		}

		// Find and verify each config maintains its original regions and zones
		for _, originalConfig := range configs {
			var foundConfig *gcpproc.GCPConfig
			for _, resultConfig := range result {
				if resultConfig.ProjectID == originalConfig.ProjectID {
					foundConfig = resultConfig
					break
				}
			}

			if foundConfig == nil {
				t.Fatalf("Could not find config for project %s in result", originalConfig.ProjectID)
			}

			if !reflect.DeepEqual(foundConfig.Regions, originalConfig.Regions) {
				t.Fatalf("Regions for project %s don't match. Expected %v, got %v",
					originalConfig.ProjectID, originalConfig.Regions, foundConfig.Regions)
			}

			if !reflect.DeepEqual(foundConfig.Zones, originalConfig.Zones) {
				t.Fatalf("Zones for project %s don't match. Expected %v, got %v",
					originalConfig.ProjectID, originalConfig.Zones, foundConfig.Zones)
			}
		}
	})

	t.Run("Same project ID with different regions - unification", func(t *testing.T) {
		configs := []*gcpproc.GCPConfig{
			{
				ProjectID: "unified-project",
				Regions:   []string{"us-central1", "us-east1"},
				Zones:     []string{"us-central1-a"},
			},
			{
				ProjectID: "unified-project",
				Regions:   []string{"europe-west1", "asia-east1"},
				Zones:     []string{"europe-west1-b"},
			},
			{
				ProjectID: "different-project",
				Regions:   []string{"us-west1"},
				Zones:     []string{"us-west1-a"},
			},
		}

		result := unifiedGCPConfigs(configs)

		// Should have 2 configs (unified-project configs merged)
		if len(result) != 2 {
			t.Fatalf("Expected 2 configs, got %d", len(result))
		}

		// Find the unified config
		var unifiedConfig *gcpproc.GCPConfig
		var differentConfig *gcpproc.GCPConfig

		for _, config := range result {
			switch config.ProjectID {
			case "unified-project":
				unifiedConfig = config
			case "different-project":
				differentConfig = config
			}
		}

		if unifiedConfig == nil {
			t.Fatal("Could not find unified-project config in result")
		}
		if differentConfig == nil {
			t.Fatal("Could not find different-project config in result")
		}

		// Verify unified config has all regions
		expectedRegions := []string{"us-central1", "us-east1", "europe-west1", "asia-east1"}

		if !reflect.DeepEqual(unifiedConfig.Regions, expectedRegions) {
			t.Fatalf("Unified regions don't match. Expected %v, got %v", expectedRegions, unifiedConfig.Regions)
		}

		// Verify unified config has all zones
		expectedZones := []string{"us-central1-a", "europe-west1-b"}

		if !reflect.DeepEqual(unifiedConfig.Zones, expectedZones) {
			t.Fatalf("Unified zones don't match. Expected %v, got %v", expectedZones, unifiedConfig.Zones)
		}

		// Verify different-project config is unchanged
		if !reflect.DeepEqual(differentConfig.Regions, []string{"us-west1"}) {
			t.Fatalf("Different project regions changed. Expected [us-west1], got %v", differentConfig.Regions)
		}
		if !reflect.DeepEqual(differentConfig.Zones, []string{"us-west1-a"}) {
			t.Fatalf("Different project zones changed. Expected [us-west1-a], got %v", differentConfig.Zones)
		}
	})

	t.Run("Same project ID with different zones and regions - unification", func(t *testing.T) {
		configs := []*gcpproc.GCPConfig{
			{
				ProjectID: "zone-project",
				Regions:   []string{"us-central1"},
				Zones:     []string{"us-central1-a", "us-central1-b"},
			},
			{
				ProjectID: "zone-project",
				Regions:   []string{"us-east1"},
				Zones:     []string{"us-east1-a", "us-east1-c"},
			},
		}

		result := unifiedGCPConfigs(configs)

		// Should have 1 config (both configs merged)
		if len(result) != 1 {
			t.Fatalf("Expected 1 config, got %d", len(result))
		}

		unifiedConfig := result[0]
		if unifiedConfig.ProjectID != "zone-project" {
			t.Fatalf("Expected project ID 'zone-project', got %s", unifiedConfig.ProjectID)
		}

		// Verify unified config has all regions
		expectedRegions := []string{"us-central1", "us-east1"}

		if !reflect.DeepEqual(unifiedConfig.Regions, expectedRegions) {
			t.Fatalf("Unified regions don't match. Expected %v, got %v", expectedRegions, unifiedConfig.Regions)
		}

		// Verify unified config has all zones
		expectedZones := []string{"us-central1-a", "us-central1-b", "us-east1-a", "us-east1-c"}

		if !reflect.DeepEqual(unifiedConfig.Zones, expectedZones) {
			t.Fatalf("Unified zones don't match. Expected %v, got %v", expectedZones, unifiedConfig.Zones)
		}
	})

	t.Run("Same project ID with overlapping regions and zones - proper unification", func(t *testing.T) {
		configs := []*gcpproc.GCPConfig{
			{
				ProjectID: "overlap-project",
				Regions:   []string{"us-central1", "us-east1", "europe-west1"},
				Zones:     []string{"us-central1-a", "us-central1-b", "europe-west1-a"},
			},
			{
				ProjectID: "overlap-project",
				Regions:   []string{"us-central1", "asia-east1"},     // us-central1 overlaps
				Zones:     []string{"us-central1-a", "asia-east1-a"}, // us-central1-a overlaps
			},
			{
				ProjectID: "overlap-project",
				Regions:   []string{"europe-west1", "us-west1"},     // europe-west1 overlaps
				Zones:     []string{"europe-west1-a", "us-west1-b"}, // europe-west1-a overlaps
			},
		}

		result := unifiedGCPConfigs(configs)

		// Should have 1 config (all configs merged)
		if len(result) != 1 {
			t.Fatalf("Expected 1 config, got %d", len(result))
		}

		unifiedConfig := result[0]
		if unifiedConfig.ProjectID != "overlap-project" {
			t.Fatalf("Expected project ID 'overlap-project', got %s", unifiedConfig.ProjectID)
		}

		expectedRegions := []string{"us-central1", "us-east1", "europe-west1", "asia-east1", "us-west1"}

		if !reflect.DeepEqual(unifiedConfig.Regions, expectedRegions) {
			t.Fatalf("Unified regions don't match. Expected %v, got %v", expectedRegions, unifiedConfig.Regions)
		}

		expectedZones := []string{"us-central1-a", "us-central1-b", "europe-west1-a", "asia-east1-a", "us-west1-b"}

		if !reflect.DeepEqual(unifiedConfig.Zones, expectedZones) {
			t.Fatalf("Unified zones don't match. Expected %v, got %v", expectedZones, unifiedConfig.Zones)
		}
	})
}
