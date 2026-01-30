package cmd

import (
	"testing"
	"time"
)

func TestBlastRadiusConfigCreation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                             string
		blastRadiusMaxDepth              int32
		blastRadiusMaxItems              int32
		blastRadiusMaxTime               time.Duration
		changeAnalysisTargetDuration         time.Duration
		expectBlastRadiusConfig          bool
		expectedBlastRadiusMaxItems      int32
		expectedBlastRadiusLinkDepth     int32
		expectChangeAnalysisTargetDuration   bool
		expectedChangeAnalysisTargetDuration time.Duration
		expectError                      bool
		expectedErrorMsg                 string
	}{
		{
			name:                    "No flags specified",
			blastRadiusMaxDepth:     0,
			blastRadiusMaxItems:     0,
			blastRadiusMaxTime:      0,
			expectBlastRadiusConfig: false,
		},
		{
			name:                         "Only maxDepth specified",
			blastRadiusMaxDepth:          5,
			blastRadiusMaxItems:          0,
			blastRadiusMaxTime:           0,
			expectBlastRadiusConfig:      true,
			expectedBlastRadiusMaxItems:  0,
			expectedBlastRadiusLinkDepth: 5,
		},
		{
			name:                         "Only maxItems specified",
			blastRadiusMaxDepth:          0,
			blastRadiusMaxItems:          1000,
			blastRadiusMaxTime:           0,
			expectBlastRadiusConfig:      true,
			expectedBlastRadiusMaxItems:  1000,
			expectedBlastRadiusLinkDepth: 0,
		},
		{
			name:                    "Only maxTime specified - BUG: creates config with zero values",
			blastRadiusMaxDepth:     0,
			blastRadiusMaxItems:     0,
			blastRadiusMaxTime:      10 * time.Minute,
			expectBlastRadiusConfig: true,
			// BUG DEMONSTRATED: When only maxTime is specified, a BlastRadiusConfig is created
			// with MaxItems=0 and LinkDepth=0. These explicit zeros will override the server's
			// defaults (100,000 and 1,000), effectively breaking the blast radius calculation.
			// The server should treat 0 values as "use defaults" rather than literal zeros.
			expectedBlastRadiusMaxItems:      0,
			expectedBlastRadiusLinkDepth:     0,
			expectChangeAnalysisTargetDuration:   true,
			expectedChangeAnalysisTargetDuration: 15 * time.Minute, // maxTime * 1.5
		},
		{
			name:                             "All flags specified",
			blastRadiusMaxDepth:              5,
			blastRadiusMaxItems:              1000,
			blastRadiusMaxTime:               15 * time.Minute,
			changeAnalysisTargetDuration:         20 * time.Minute,
			expectBlastRadiusConfig:          true,
			expectedBlastRadiusMaxItems:      1000,
			expectedBlastRadiusLinkDepth:     5,
			expectChangeAnalysisTargetDuration:   true,
			expectedChangeAnalysisTargetDuration: 20 * time.Minute, // changeAnalysisTargetDuration overrides maxTime
		},
		{
			name:                             "maxTime and maxDepth specified",
			blastRadiusMaxDepth:              3,
			blastRadiusMaxItems:              0,
			blastRadiusMaxTime:               5 * time.Minute,
			expectBlastRadiusConfig:          true,
			expectedBlastRadiusMaxItems:      0,
			expectedBlastRadiusLinkDepth:     3,
			expectChangeAnalysisTargetDuration:   true,
			expectedChangeAnalysisTargetDuration: 7*time.Minute + 30*time.Second, // maxTime * 1.5
		},
		{
			name:                             "maxTime and maxItems specified",
			blastRadiusMaxDepth:              0,
			blastRadiusMaxItems:              500,
			blastRadiusMaxTime:               20 * time.Minute,
			expectBlastRadiusConfig:          true,
			expectedBlastRadiusMaxItems:      500,
			expectedBlastRadiusLinkDepth:     0,
			expectChangeAnalysisTargetDuration:   true,
			expectedChangeAnalysisTargetDuration: 30 * time.Minute, // maxTime * 1.5
		},
		{
			name:                             "Only changeAnalysisTargetDuration specified",
			blastRadiusMaxDepth:              0,
			blastRadiusMaxItems:              0,
			blastRadiusMaxTime:               0,
			changeAnalysisTargetDuration:         10 * time.Minute,
			expectBlastRadiusConfig:          true,
			expectedBlastRadiusMaxItems:      0,
			expectedBlastRadiusLinkDepth:     0,
			expectChangeAnalysisTargetDuration:   true,
			expectedChangeAnalysisTargetDuration: 10 * time.Minute,
		},
		{
			name:                     "changeAnalysisTargetDuration too low",
			blastRadiusMaxDepth:      0,
			blastRadiusMaxItems:      0,
			blastRadiusMaxTime:       0,
			changeAnalysisTargetDuration: 30 * time.Second,
			expectBlastRadiusConfig:  true,
			expectError:              true,
			expectedErrorMsg:         "--change-analysis-target-duration must be between 1 minute and 30 minutes",
		},
		{
			name:                     "changeAnalysisTargetDuration too high",
			blastRadiusMaxDepth:      0,
			blastRadiusMaxItems:      0,
			blastRadiusMaxTime:       0,
			changeAnalysisTargetDuration: 31 * time.Minute,
			expectBlastRadiusConfig:  true,
			expectError:              true,
			expectedErrorMsg:         "--change-analysis-target-duration must be between 1 minute and 30 minutes",
		},
		{
			name:                    "maxTime results in timeout too low",
			blastRadiusMaxDepth:     0,
			blastRadiusMaxItems:     0,
			blastRadiusMaxTime:      30 * time.Second, // * 1.5 = 45 seconds, which is < 1 minute
			expectBlastRadiusConfig: true,
			expectError:             true,
			expectedErrorMsg:        "--change-analysis-target-duration must be between 1 minute and 30 minutes",
		},
		{
			name:                    "maxTime results in timeout too high",
			blastRadiusMaxDepth:     0,
			blastRadiusMaxItems:     0,
			blastRadiusMaxTime:      21 * time.Minute, // * 1.5 = 31.5 minutes, which is > 30 minutes
			expectBlastRadiusConfig: true,
			expectError:             true,
			expectedErrorMsg:        "--change-analysis-target-duration must be between 1 minute and 30 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			blastRadiusConfigOverride, err := createBlastRadiusConfig(tt.blastRadiusMaxDepth, tt.blastRadiusMaxItems, tt.blastRadiusMaxTime, tt.changeAnalysisTargetDuration)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
					return
				}
				if err.Error() != tt.expectedErrorMsg {
					t.Errorf("Expected error message %q, but got %q", tt.expectedErrorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify expectations
			if tt.expectBlastRadiusConfig && blastRadiusConfigOverride == nil {
				t.Errorf("Expected BlastRadiusConfig to be created, but got nil")
				return
			}
			if !tt.expectBlastRadiusConfig && blastRadiusConfigOverride != nil {
				t.Errorf("Expected BlastRadiusConfig to be nil, but got %+v", blastRadiusConfigOverride)
				return
			}

			if tt.expectBlastRadiusConfig {
				if blastRadiusConfigOverride.GetMaxItems() != tt.expectedBlastRadiusMaxItems {
					t.Errorf("Expected MaxItems to be %d, but got %d", tt.expectedBlastRadiusMaxItems, blastRadiusConfigOverride.GetMaxItems())
				}
				if blastRadiusConfigOverride.GetLinkDepth() != tt.expectedBlastRadiusLinkDepth {
					t.Errorf("Expected LinkDepth to be %d, but got %d", tt.expectedBlastRadiusLinkDepth, blastRadiusConfigOverride.GetLinkDepth())
				}
				if tt.expectChangeAnalysisTargetDuration {
					if blastRadiusConfigOverride.GetChangeAnalysisTargetDuration() == nil {
						t.Errorf("Expected ChangeAnalysisTargetDuration to be set, but got nil")
					} else {
						actualTimeout := blastRadiusConfigOverride.GetChangeAnalysisTargetDuration().AsDuration()
						if actualTimeout != tt.expectedChangeAnalysisTargetDuration {
							t.Errorf("Expected ChangeAnalysisTargetDuration to be %v, but got %v", tt.expectedChangeAnalysisTargetDuration, actualTimeout)
						}
					}
				} else {
					if blastRadiusConfigOverride.GetChangeAnalysisTargetDuration() != nil {
						t.Errorf("Expected ChangeAnalysisTargetDuration to be nil, but got %v", blastRadiusConfigOverride.GetChangeAnalysisTargetDuration())
					}
				}
			}
		})
	}
}
