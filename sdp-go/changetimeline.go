package sdp

// If you add/delete/move an entry here, make sure to update/check the following:
// - the PopulateChangeTimelineV2 function
// - GetChangeTimelineV2 in api-server/server/changesservice.go
// - resetChangeAnalysisTables in api-server/server/changeanalysis/shared.go
// - the cli tool if we are waiting for a change analysis to finish
// - frontend/src/features/changes-v2/change-timeline/ChangeTimeline.tsx - also update the entryNames object as this is used for comparing entry names
// All timeline entries are now defined using ChangeTimelineEntryV2ID variables below.
// Use the .Label field for database lookups and the .Name field for user-facing display.

type ChangeTimelineEntryV2ID struct {
	// The internal label for the entry, this is used to identify the entry in
	// the database and tell whether two entries are the same type of thing.
	// This means that if we want to change the way an entry behaves, we can
	// create a new label and keep the old one for backwards compatibility.
	Label string
	// The name of the entry, this is the user facing name of the entry and can
	// be changed safely. This is stored in both the code and the database, the
	// reason we store it in the code is so that we know what value to populate
	// in the database when we create the timeline entries in the first place,
	// when returning the timeline to the user we use the name from the database
	// which means that old changes will still show the old name.
	Name string
}

// if you add/delete/move an entry here, make sure to update/check the following:
// - changeTimelineEntryNameInProgress
// - changeTimelineEntryNameInProgressReverse
// - allChangeTimelineEntryV2IDs
var (
	// This is the entry that is created when we map the resources for a change,
	// this happens before we start blast radius simulation, it involves taking
	// the mapping queries that were sent up, and running them against the
	// gateway to see whether any of them resolve into real items.
	ChangeTimelineEntryV2IDMapResources = ChangeTimelineEntryV2ID{
		Label: "mapped_resources",
		Name:  "Map resources",
	}
	// This is the entry that is created when we calculate the blast radius for a
	// change, this happens after we map the resources for a change, it involves
	// taking the mapped resources and running them through the blast radius
	// simulation to see how many items are in the blast radius.
	ChangeTimelineEntryV2IDCalculatedBlastRadius = ChangeTimelineEntryV2ID{
		Label: "calculated_blast_radius",
		Name:  "Simulate blast radius",
	}
	// we do not show this entry in the timeline anymore
	// This is the entry tracks the calculation of routine signals for all of
	// the modifications within this change
	ChangeTimelineEntryV2IDAnalyzedSignals = ChangeTimelineEntryV2ID{
		Label: "calculated_routineness",
		Name:  "Analyze signals",
	}
	// This is the entry that tracks the calculation of risks and returns them
	// in the timeline. At the time of writing this has been replaced and we are
	// no longer showing risks directly in the timeline. The risk calculation
	// still happens, but the timeline focuses on Observations -> Hypotheses ->
	// Investigations instead. This means that this step will be no longer used
	// after Dec '25
	ChangeTimelineEntryV2IDCalculatedRisks = ChangeTimelineEntryV2ID{
		Label: "calculated_risks",
		Name:  "Calculated Risks",
	}
	// Tracks the application of auto-label rules for a change
	ChangeTimelineEntryV2IDCalculatedLabels = ChangeTimelineEntryV2ID{
		Label: "calculated_labels",
		Name:  "Apply auto labels",
	}
	// Tracks the application of auto tags for a change
	ChangeTimelineEntryV2IDAutoTagging = ChangeTimelineEntryV2ID{
		Label: "auto_tagging",
		Name:  "Auto Tagging",
	}
	// Tracks the validation of a change. This happens after the change is
	// complete and at time of writing is not generally available
	ChangeTimelineEntryV2IDChangeValidation = ChangeTimelineEntryV2ID{
		Label: "change_validation",
		Name:  "Change Validation",
	}
	// This is the entry that tracks observations being recorded during blast radius simulation
	ChangeTimelineEntryV2IDRecordObservations = ChangeTimelineEntryV2ID{
		Label: "record_observations",
		Name:  "Record observations",
	}
	// This is the entry that tracks hypotheses being formed from observations via batch processing
	ChangeTimelineEntryV2IDFormHypotheses = ChangeTimelineEntryV2ID{
		Label: "form_hypotheses",
		Name:  "Form hypotheses",
	}
	// This is the entry that tracks investigation of hypotheses via one-shot analysis
	ChangeTimelineEntryV2IDInvestigateHypotheses = ChangeTimelineEntryV2ID{
		Label: "investigate_hypotheses",
		Name:  "Investigate hypotheses",
	}
)

// changeTimelineEntryNameInProgress maps default/done names to their in-progress equivalents.
// This map is used to convert timeline entry names based on their status.
var changeTimelineEntryNameInProgress = map[string]string{
	"Map resources":          "Mapping resources...",
	"Simulate blast radius":  "Simulating blast radius...",
	"Record observations":    "Recording observations...",
	"Form hypotheses":        "Forming hypotheses...",
	"Investigate hypotheses": "Investigating hypotheses...",
	"Analyze signals":        "Analyzing signals...",
	"Apply auto labels":      "Applying auto labels...",
}

// changeTimelineEntryNameInProgressReverse maps in-progress names back to their default/done equivalents.
// This is used for archive imports where we need to normalize names to look up labels.
var changeTimelineEntryNameInProgressReverse = func() map[string]string {
	reverse := make(map[string]string, len(changeTimelineEntryNameInProgress))
	for defaultName, inProgressName := range changeTimelineEntryNameInProgress {
		reverse[inProgressName] = defaultName
	}
	return reverse
}()

// allChangeTimelineEntryV2IDs is a slice of all timeline entry ID constants for iteration.
var allChangeTimelineEntryV2IDs = []ChangeTimelineEntryV2ID{
	ChangeTimelineEntryV2IDMapResources,
	ChangeTimelineEntryV2IDCalculatedBlastRadius,
	ChangeTimelineEntryV2IDAnalyzedSignals,
	ChangeTimelineEntryV2IDCalculatedRisks,
	ChangeTimelineEntryV2IDCalculatedLabels,
	ChangeTimelineEntryV2IDAutoTagging,
	ChangeTimelineEntryV2IDChangeValidation,
	ChangeTimelineEntryV2IDRecordObservations,
	ChangeTimelineEntryV2IDFormHypotheses,
	ChangeTimelineEntryV2IDInvestigateHypotheses,
}

// GetChangeTimelineEntryNameForStatus returns the appropriate name for a timeline entry
// based on its status. If the status is IN_PROGRESS, it returns the in-progress name.
// Otherwise, it returns the name as-is (which is the default/done name).
func GetChangeTimelineEntryNameForStatus(name string, status ChangeTimelineEntryStatus) string {
	if status == ChangeTimelineEntryStatus_IN_PROGRESS {
		if inProgressName, ok := changeTimelineEntryNameInProgress[name]; ok {
			return inProgressName
		}
	}
	return name
}

// GetChangeTimelineEntryLabelFromName converts a timeline entry name (either in-progress or default/done)
// to its corresponding label. This is used for archive imports where we need to match names to labels.
// Returns an empty string if the name doesn't match any known timeline entry.
func GetChangeTimelineEntryLabelFromName(name string) string {
	// First, normalize the name: if it's an in-progress name, convert it to default/done name
	normalizedName := name
	if defaultName, ok := changeTimelineEntryNameInProgressReverse[name]; ok {
		normalizedName = defaultName
	}

	// Then look up the label from the constants
	for _, entryID := range allChangeTimelineEntryV2IDs {
		if entryID.Name == normalizedName {
			return entryID.Label
		}
	}

	return ""
}
