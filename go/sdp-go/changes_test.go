package sdp

import (
	"strings"
	"testing"
)

func TestFindInProgressEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		entries        []*ChangeTimelineEntryV2
		expectedName   string
		expectedStatus ChangeTimelineEntryStatus
		expectError    bool
	}{
		{
			name:           "nil entries",
			entries:        nil,
			expectedName:   "",
			expectedStatus: ChangeTimelineEntryStatus_UNSPECIFIED,
			expectError:    true,
		},
		{
			name:           "empty entries",
			entries:        []*ChangeTimelineEntryV2{},
			expectedName:   "",
			expectedStatus: ChangeTimelineEntryStatus_UNSPECIFIED,
			expectError:    true,
		},
		{
			name: "in progress entry",
			entries: []*ChangeTimelineEntryV2{
				{
					Name:   "entry1",
					Status: ChangeTimelineEntryStatus_IN_PROGRESS,
				},
				{
					Name:   "entry2",
					Status: ChangeTimelineEntryStatus_PENDING,
				},
			},
			expectedName:   "entry1",
			expectedStatus: ChangeTimelineEntryStatus_IN_PROGRESS,
			expectError:    false,
		},
		{
			name: "pending entry",
			entries: []*ChangeTimelineEntryV2{
				{
					Name:   "entry1",
					Status: ChangeTimelineEntryStatus_DONE,
				},
				{
					Name:   "entry2",
					Status: ChangeTimelineEntryStatus_PENDING,
				},
			},
			expectedName:   "entry2",
			expectedStatus: ChangeTimelineEntryStatus_PENDING,
			expectError:    false,
		},
		{
			name: "error entry",
			entries: []*ChangeTimelineEntryV2{
				{
					Name:   "entry1",
					Status: ChangeTimelineEntryStatus_DONE,
				},
				{
					Name:   "entry2",
					Status: ChangeTimelineEntryStatus_ERROR,
				},
			},
			expectedName:   "entry2",
			expectedStatus: ChangeTimelineEntryStatus_ERROR,
			expectError:    false,
		},
		{
			name: "no in progress entry",
			entries: []*ChangeTimelineEntryV2{
				{
					Name:   "entry1",
					Status: ChangeTimelineEntryStatus_DONE,
				},
				{
					Name:   "entry2",
					Status: ChangeTimelineEntryStatus_UNSPECIFIED,
				},
			},
			expectedName:   "",
			expectedStatus: ChangeTimelineEntryStatus_DONE,
			expectError:    false,
		},
		{
			name: "unknown status",
			entries: []*ChangeTimelineEntryV2{
				{
					Name:   "entry1",
					Status: 100, // some unknown status
				},
			},
			expectedName:   "",
			expectedStatus: ChangeTimelineEntryStatus_UNSPECIFIED,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, _, status, err := TimelineFindInProgressEntry(tt.entries)

			if tt.expectError && err == nil {
				t.Errorf("Expected an error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if name != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, name)
			}

			if status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, status)
			}
		})
	}
}

func TestTimelineEntryContentDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entry    *ChangeTimelineEntryV2
		expected string
	}{
		{
			name: "mapped items",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_MappedItems{
					MappedItems: &MappedItemsTimelineEntry{
						MappedItems: []*MappedItemDiff{{}, {}, {}},
					},
				},
			},
			expected: "3 mapped items",
		},
		{
			name: "calculated blast radius",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_CalculatedBlastRadius{
					CalculatedBlastRadius: &CalculatedBlastRadiusTimelineEntry{
						NumItems: 10,
						NumEdges: 25,
					},
				},
			},
			expected: "10 items, 25 edges",
		},
		{
			name: "calculated risks",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_CalculatedRisks{
					CalculatedRisks: &CalculatedRisksTimelineEntry{
						Risks: []*Risk{{}, {}},
					},
				},
			},
			expected: "2 risks",
		},
		{
			name: "calculated labels",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_CalculatedLabels{
					CalculatedLabels: &CalculatedLabelsTimelineEntry{
						Labels: []*Label{{}, {}, {}, {}},
					},
				},
			},
			expected: "4 labels",
		},
		{
			name: "change validation",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_ChangeValidation{
					ChangeValidation: &ChangeValidationTimelineEntry{
						ValidationChecklist: []*ChangeValidationCategory{{}},
					},
				},
			},
			expected: "1 validation categories",
		},
		{
			name: "form hypotheses",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_FormHypotheses{
					FormHypotheses: &FormHypothesesTimelineEntry{
						NumHypotheses: 5,
					},
				},
			},
			expected: "5 hypotheses",
		},
		{
			name: "investigate hypotheses",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_InvestigateHypotheses{
					InvestigateHypotheses: &InvestigateHypothesesTimelineEntry{
						NumProven:        2,
						NumDisproven:     3,
						NumInvestigating: 1,
					},
				},
			},
			expected: "2 proven, 3 disproven, 1 investigating",
		},
		{
			name: "record observations",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_RecordObservations{
					RecordObservations: &RecordObservationsTimelineEntry{
						NumObservations: 42,
					},
				},
			},
			expected: "42 observations",
		},
		{
			name: "error content",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_Error{
					Error: "something went wrong",
				},
			},
			expected: "something went wrong",
		},
		{
			name: "status message",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_StatusMessage{
					StatusMessage: "processing data",
				},
			},
			expected: "processing data",
		},
		{
			name: "empty content",
			entry: &ChangeTimelineEntryV2{
				Content: &ChangeTimelineEntryV2_Empty{
					Empty: &EmptyContent{},
				},
			},
			expected: "",
		},
		{
			name: "nil content",
			entry: &ChangeTimelineEntryV2{
				Content: nil,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TimelineEntryContentDescription(tt.entry)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateRoutineChangesConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *RoutineChangesYAML
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &RoutineChangesYAML{
				EventsPerDay:   10.0,
				DurationInDays: 7.0,
				Sensitivity:    0.5,
			},
			wantErr: false,
		},
		{
			name: "valid config with minimum values",
			config: &RoutineChangesYAML{
				EventsPerDay:   1.0,
				DurationInDays: 1.0,
				Sensitivity:    0.0,
			},
			wantErr: false,
		},
		{
			name: "events_per_day less than 1",
			config: &RoutineChangesYAML{
				EventsPerDay:   0.5,
				DurationInDays: 7.0,
				Sensitivity:    0.5,
			},
			wantErr:     true,
			errContains: "events_per_day must be greater than 1",
		},
		{
			name: "events_per_day equals 0",
			config: &RoutineChangesYAML{
				EventsPerDay:   0.0,
				DurationInDays: 7.0,
				Sensitivity:    0.5,
			},
			wantErr:     true,
			errContains: "events_per_day must be greater than 1",
		},
		{
			name: "events_per_day negative",
			config: &RoutineChangesYAML{
				EventsPerDay:   -1.0,
				DurationInDays: 7.0,
				Sensitivity:    0.5,
			},
			wantErr:     true,
			errContains: "events_per_day must be greater than 1",
		},
		{
			name: "duration_in_days less than 1",
			config: &RoutineChangesYAML{
				EventsPerDay:   10.0,
				DurationInDays: 0.5,
				Sensitivity:    0.5,
			},
			wantErr:     true,
			errContains: "duration_in_days must be greater than 1",
		},
		{
			name: "duration_in_days equals 0",
			config: &RoutineChangesYAML{
				EventsPerDay:   10.0,
				DurationInDays: 0.0,
				Sensitivity:    0.5,
			},
			wantErr:     true,
			errContains: "duration_in_days must be greater than 1",
		},
		{
			name: "duration_in_days negative",
			config: &RoutineChangesYAML{
				EventsPerDay:   10.0,
				DurationInDays: -1.0,
				Sensitivity:    0.5,
			},
			wantErr:     true,
			errContains: "duration_in_days must be greater than 1",
		},
		{
			name: "sensitivity negative",
			config: &RoutineChangesYAML{
				EventsPerDay:   10.0,
				DurationInDays: 7.0,
				Sensitivity:    -0.1,
			},
			wantErr:     true,
			errContains: "sensitivity must be 0 or higher",
		},
		{
			name: "multiple invalid fields - events_per_day checked first",
			config: &RoutineChangesYAML{
				EventsPerDay:   0.0,
				DurationInDays: 0.0,
				Sensitivity:    -1.0,
			},
			wantErr:     true,
			errContains: "events_per_day must be greater than 1",
		},
		{
			name: "multiple invalid fields - duration_in_days checked second",
			config: &RoutineChangesYAML{
				EventsPerDay:   10.0,
				DurationInDays: 0.0,
				Sensitivity:    -1.0,
			},
			wantErr:     true,
			errContains: "duration_in_days must be greater than 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRoutineChangesConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRoutineChangesConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil {
					t.Errorf("validateRoutineChangesConfig() expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateRoutineChangesConfig() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestYamlStringToSignalConfig_NilCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yamlString  string
		wantErr     bool
		wantRoutine bool
		wantGithub  bool
	}{
		{
			name:        "both nil -> error",
			yamlString:  "{}\n",
			wantErr:     true,
			wantRoutine: false,
			wantGithub:  false,
		},
		{
			name:        "only routine present",
			yamlString:  "routine_changes_config:\n  sensitivity: 0\n  duration_in_days: 1\n  events_per_day: 1\n",
			wantErr:     false,
			wantRoutine: true,
			wantGithub:  false,
		},
		{
			name:        "only github present",
			yamlString:  "github_organisation_profile:\n  primary_branch_name: main\n",
			wantErr:     false,
			wantRoutine: false,
			wantGithub:  true,
		},
		{
			name:        "both present",
			yamlString:  "routine_changes_config:\n  sensitivity: 0\n  duration_in_days: 1\n  events_per_day: 1\ngithub_organisation_profile:\n  primary_branch_name: main\n",
			wantErr:     false,
			wantRoutine: true,
			wantGithub:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := YamlStringToSignalConfig(tt.yamlString)
			if (err != nil) != tt.wantErr {
				t.Errorf("YamlStringToSignalConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if (got.RoutineChangesConfig != nil) != tt.wantRoutine {
				t.Errorf("RoutineChangesConfig presence = %v, want %v", got.RoutineChangesConfig != nil, tt.wantRoutine)
			}
			if (got.GithubOrganisationProfile != nil) != tt.wantGithub {
				t.Errorf("GithubOrganisationProfile presence = %v, want %v", got.GithubOrganisationProfile != nil, tt.wantGithub)
			}
		})
	}
}
