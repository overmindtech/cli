package sdp

import "testing"

// TestChangeTimelineEntryNameConversion tests both GetChangeTimelineEntryNameForStatus
// and GetChangeTimelineEntryLabelFromName together, including round-trip conversions.
func TestChangeTimelineEntryNameConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		entryID              ChangeTimelineEntryV2ID
		hasInProgressVariant bool
	}{
		{
			name:                 "Map resources",
			entryID:              ChangeTimelineEntryV2IDMapResources,
			hasInProgressVariant: true,
		},
		{
			name:                 "Simulate blast radius",
			entryID:              ChangeTimelineEntryV2IDCalculatedBlastRadius,
			hasInProgressVariant: true,
		},
		{
			name:                 "Record observations",
			entryID:              ChangeTimelineEntryV2IDRecordObservations,
			hasInProgressVariant: true,
		},
		{
			name:                 "Form hypotheses",
			entryID:              ChangeTimelineEntryV2IDFormHypotheses,
			hasInProgressVariant: true,
		},
		{
			name:                 "Investigate hypotheses",
			entryID:              ChangeTimelineEntryV2IDInvestigateHypotheses,
			hasInProgressVariant: true,
		},
		{
			name:                 "Analyze signals",
			entryID:              ChangeTimelineEntryV2IDAnalyzedSignals,
			hasInProgressVariant: true,
		},
		{
			name:                 "Apply auto labels",
			entryID:              ChangeTimelineEntryV2IDCalculatedLabels,
			hasInProgressVariant: true,
		},
		{
			name:                 "Calculated risks (no in-progress variant)",
			entryID:              ChangeTimelineEntryV2IDCalculatedRisks,
			hasInProgressVariant: false,
		},
		{
			name:                 "Auto Tagging (no in-progress variant)",
			entryID:              ChangeTimelineEntryV2IDAutoTagging,
			hasInProgressVariant: false,
		},
		{
			name:                 "Change Validation (no in-progress variant)",
			entryID:              ChangeTimelineEntryV2IDChangeValidation,
			hasInProgressVariant: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultName := tt.entryID.Name
			expectedLabel := tt.entryID.Label

			// Test 1: Default name -> IN_PROGRESS -> in-progress name
			if tt.hasInProgressVariant {
				gotInProgressName := GetChangeTimelineEntryNameForStatus(defaultName, ChangeTimelineEntryStatus_IN_PROGRESS)
				// Verify that the in-progress name is different from the default name
				if gotInProgressName == defaultName {
					t.Errorf("GetChangeTimelineEntryNameForStatus(%q, IN_PROGRESS) should return in-progress name, got %q", defaultName, gotInProgressName)
				}
				// Verify it ends with "..." to indicate in-progress
				if len(gotInProgressName) < 3 || gotInProgressName[len(gotInProgressName)-3:] != "..." {
					t.Errorf("GetChangeTimelineEntryNameForStatus(%q, IN_PROGRESS) = %q, expected in-progress name ending with '...'", defaultName, gotInProgressName)
				}
				expectedInProgressName := gotInProgressName // Use the function result as the expected value

				// Test 2: In-progress name -> label (for archive imports)
				gotLabelFromInProgress := GetChangeTimelineEntryLabelFromName(expectedInProgressName)
				if gotLabelFromInProgress != expectedLabel {
					t.Errorf("GetChangeTimelineEntryLabelFromName(%q) = %q, want %q", expectedInProgressName, gotLabelFromInProgress, expectedLabel)
				}

				// Test 3: Round-trip: default -> in-progress -> label -> should match expected label
				inProgressName := GetChangeTimelineEntryNameForStatus(defaultName, ChangeTimelineEntryStatus_IN_PROGRESS)
				labelFromRoundTrip := GetChangeTimelineEntryLabelFromName(inProgressName)
				if labelFromRoundTrip != expectedLabel {
					t.Errorf("Round-trip: default(%q) -> in-progress(%q) -> label(%q), want label %q", defaultName, inProgressName, labelFromRoundTrip, expectedLabel)
				}
			}

			// Test 4: Default name -> DONE status -> should return default name
			gotDoneName := GetChangeTimelineEntryNameForStatus(defaultName, ChangeTimelineEntryStatus_DONE)
			if gotDoneName != defaultName {
				t.Errorf("GetChangeTimelineEntryNameForStatus(%q, DONE) = %q, want %q", defaultName, gotDoneName, defaultName)
			}

			// Test 5: Default name -> PENDING status -> should return default name
			gotPendingName := GetChangeTimelineEntryNameForStatus(defaultName, ChangeTimelineEntryStatus_PENDING)
			if gotPendingName != defaultName {
				t.Errorf("GetChangeTimelineEntryNameForStatus(%q, PENDING) = %q, want %q", defaultName, gotPendingName, defaultName)
			}

			// Test 6: Default name -> ERROR status -> should return default name
			gotErrorName := GetChangeTimelineEntryNameForStatus(defaultName, ChangeTimelineEntryStatus_ERROR)
			if gotErrorName != defaultName {
				t.Errorf("GetChangeTimelineEntryNameForStatus(%q, ERROR) = %q, want %q", defaultName, gotErrorName, defaultName)
			}

			// Test 7: Default name -> UNSPECIFIED status -> should return default name
			gotUnspecifiedName := GetChangeTimelineEntryNameForStatus(defaultName, ChangeTimelineEntryStatus_UNSPECIFIED)
			if gotUnspecifiedName != defaultName {
				t.Errorf("GetChangeTimelineEntryNameForStatus(%q, UNSPECIFIED) = %q, want %q", defaultName, gotUnspecifiedName, defaultName)
			}

			// Test 8: Default name -> label (for archive imports)
			gotLabelFromDefault := GetChangeTimelineEntryLabelFromName(defaultName)
			if gotLabelFromDefault != expectedLabel {
				t.Errorf("GetChangeTimelineEntryLabelFromName(%q) = %q, want %q", defaultName, gotLabelFromDefault, expectedLabel)
			}
		})
	}

	// Test edge cases
	t.Run("Unknown name with IN_PROGRESS returns name as-is", func(t *testing.T) {
		unknownName := "Unknown Entry"
		result := GetChangeTimelineEntryNameForStatus(unknownName, ChangeTimelineEntryStatus_IN_PROGRESS)
		if result != unknownName {
			t.Errorf("GetChangeTimelineEntryNameForStatus(%q, IN_PROGRESS) = %q, want %q", unknownName, result, unknownName)
		}
	})

	t.Run("Unknown name returns empty label", func(t *testing.T) {
		unknownName := "Unknown Entry"
		result := GetChangeTimelineEntryLabelFromName(unknownName)
		if result != "" {
			t.Errorf("GetChangeTimelineEntryLabelFromName(%q) = %q, want empty string", unknownName, result)
		}
	})

	t.Run("Empty string returns empty label", func(t *testing.T) {
		result := GetChangeTimelineEntryLabelFromName("")
		if result != "" {
			t.Errorf("GetChangeTimelineEntryLabelFromName(\"\") = %q, want empty string", result)
		}
	})
}
