package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestLoadAutoTagRulesFile(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		errorContains string
		numberOfRules int
	}{
		{
			name:          "NonExistent.yaml",
			fileContent:   "",
			errorContains: "does not exist",
		},
		{
			name:          "FailedToParse.yaml",
			fileContent:   "invalid yaml content",
			errorContains: "failed to parse",
		},
		{
			name:          "MoreThan10Rules.yaml",
			fileContent:   generateRules(11),
			errorContains: "10 rules",
		},
		{
			name:          "ValidRules.yaml",
			fileContent:   generateRules(5),
			errorContains: "there should be no error",
			numberOfRules: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.name
			if tt.fileContent != "" {
				err := os.WriteFile(filePath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Errorf("Failed to write file %q: %v", filePath, err)
				}
				defer os.Remove(filePath)
			}

			result, err := loadAutoTagRulesFile(filePath)
			if err != nil && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Expected error\n%q\nto contain\n%q", err.Error(), tt.errorContains)
			}
			if tt.numberOfRules > 0 {

				if len(result) != tt.numberOfRules {
					t.Errorf("Expected %d rules, got %d", tt.numberOfRules, len(result))
				}
			}
		})
	}
}

func generateRules(count int) string {
	rules := "rules:"
	rules += `
  - name: rule0
    tag_key: key0
    enabled: true
    instructions: Instructions for rule 0
    valid_values:
      - value 0`
	for i := 1; i < count; i++ {
		rules += fmt.Sprintf(`
  - name: rule%d
    tag_key: key%d
    enabled: true
    instructions: Instructions for rule %d
    valid_values: ["value %d"]`, i, i, i, i)
	}
	return rules
}

func TestBlastRadiusConfigCreation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                       string
		maxDepth                   int32
		maxItems                   int32
		maxTime                    time.Duration
		expectConfig               bool
		expectedMaxItems           int32
		expectedLinkDepth          int32
		expectMaxBlastRadiusTime   bool
		expectedMaxBlastRadiusTime time.Duration
	}{
		{
			name:         "No flags specified",
			maxDepth:     0,
			maxItems:     0,
			maxTime:      0,
			expectConfig: false,
		},
		{
			name:              "Only maxDepth specified",
			maxDepth:          5,
			maxItems:          0,
			maxTime:           0,
			expectConfig:      true,
			expectedMaxItems:  0,
			expectedLinkDepth: 5,
		},
		{
			name:              "Only maxItems specified",
			maxDepth:          0,
			maxItems:          1000,
			maxTime:           0,
			expectConfig:      true,
			expectedMaxItems:  1000,
			expectedLinkDepth: 0,
		},
		{
			name:         "Only maxTime specified - BUG: creates config with zero values",
			maxDepth:     0,
			maxItems:     0,
			maxTime:      10 * time.Minute,
			expectConfig: true,
			// BUG DEMONSTRATED: When only maxTime is specified, a BlastRadiusConfig is created
			// with MaxItems=0 and LinkDepth=0. These explicit zeros will override the server's
			// defaults (100,000 and 1,000), effectively breaking the blast radius calculation.
			// The server should treat 0 values as "use defaults" rather than literal zeros.
			expectedMaxItems:           0,
			expectedLinkDepth:          0,
			expectMaxBlastRadiusTime:   true,
			expectedMaxBlastRadiusTime: 10 * time.Minute,
		},
		{
			name:                       "All flags specified",
			maxDepth:                   5,
			maxItems:                   1000,
			maxTime:                    15 * time.Minute,
			expectConfig:               true,
			expectedMaxItems:           1000,
			expectedLinkDepth:          5,
			expectMaxBlastRadiusTime:   true,
			expectedMaxBlastRadiusTime: 15 * time.Minute,
		},
		{
			name:                       "maxTime and maxDepth specified",
			maxDepth:                   3,
			maxItems:                   0,
			maxTime:                    5 * time.Minute,
			expectConfig:               true,
			expectedMaxItems:           0,
			expectedLinkDepth:          3,
			expectMaxBlastRadiusTime:   true,
			expectedMaxBlastRadiusTime: 5 * time.Minute,
		},
		{
			name:                       "maxTime and maxItems specified",
			maxDepth:                   0,
			maxItems:                   500,
			maxTime:                    20 * time.Minute,
			expectConfig:               true,
			expectedMaxItems:           500,
			expectedLinkDepth:          0,
			expectMaxBlastRadiusTime:   true,
			expectedMaxBlastRadiusTime: 20 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// This is the logic from changes_submit_plan.go lines 222-235
			var blastRadiusConfigOverride *sdp.BlastRadiusConfig
			if tt.maxDepth > 0 || tt.maxItems > 0 || tt.maxTime > 0 {
				blastRadiusConfigOverride = &sdp.BlastRadiusConfig{
					MaxItems:  tt.maxItems,
					LinkDepth: tt.maxDepth,
				}
				if tt.maxTime > 0 {
					blastRadiusConfigOverride.MaxBlastRadiusTime = durationpb.New(tt.maxTime)
				}
			}

			// Verify expectations
			if tt.expectConfig && blastRadiusConfigOverride == nil {
				t.Errorf("Expected BlastRadiusConfig to be created, but got nil")
				return
			}
			if !tt.expectConfig && blastRadiusConfigOverride != nil {
				t.Errorf("Expected BlastRadiusConfig to be nil, but got %+v", blastRadiusConfigOverride)
				return
			}

			if tt.expectConfig {
				if blastRadiusConfigOverride.GetMaxItems() != tt.expectedMaxItems {
					t.Errorf("Expected MaxItems to be %d, but got %d", tt.expectedMaxItems, blastRadiusConfigOverride.GetMaxItems())
				}
				if blastRadiusConfigOverride.GetLinkDepth() != tt.expectedLinkDepth {
					t.Errorf("Expected LinkDepth to be %d, but got %d", tt.expectedLinkDepth, blastRadiusConfigOverride.GetLinkDepth())
				}
				if tt.expectMaxBlastRadiusTime {
					if blastRadiusConfigOverride.GetMaxBlastRadiusTime() == nil {
						t.Errorf("Expected MaxBlastRadiusTime to be set, but got nil")
					} else if blastRadiusConfigOverride.GetMaxBlastRadiusTime().AsDuration() != tt.expectedMaxBlastRadiusTime {
						t.Errorf("Expected MaxBlastRadiusTime to be %v, but got %v", tt.expectedMaxBlastRadiusTime, blastRadiusConfigOverride.GetMaxBlastRadiusTime().AsDuration())
					}
				} else {
					if blastRadiusConfigOverride.GetMaxBlastRadiusTime() != nil {
						t.Errorf("Expected MaxBlastRadiusTime to be nil, but got %v", blastRadiusConfigOverride.GetMaxBlastRadiusTime())
					}
				}
			}
		})
	}
}
