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

var (
	// This is the entry that is created when we map the resources for a change,
	// this happens before we start blast radius simulation, it involves taking
	// the mapping queries that were sent up, and running them against the
	// gateway to see whether any of them resolve into real items.
	ChangeTimelineEntryV2IDMapResources = ChangeTimelineEntryV2ID{
		Label: "mapped_resources",
		Name:  "Map Resources",
	}
	// This is the entry that is created when we calculate the blast radius for a
	// change, this happens after we map the resources for a change, it involves
	// taking the mapped resources and running them through the blast radius
	// simulation to see how many items are in the blast radius.
	ChangeTimelineEntryV2IDCalculatedBlastRadius = ChangeTimelineEntryV2ID{
		Label: "calculated_blast_radius",
		Name:  "Calculated Blast Radius",
	}
	// This is the entry tracks the calculation of routine signals for all of
	// the modifications within this change
	ChangeTimelineEntryV2IDAnalyzedSignals = ChangeTimelineEntryV2ID{
		Label: "calculated_routineness",
		Name:  "Analyze Signals",
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
		Name:  "Calculated Labels",
	}
	// Tracks the calculation of auto tags for a change. This has been replaced
	// by auto labels and will not be run on new changes anymore after Jan '26
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
		Name:  "Record Observations",
	}
	// This is the entry that tracks hypotheses being formed from observations via batch processing
	ChangeTimelineEntryV2IDFormHypotheses = ChangeTimelineEntryV2ID{
		Label: "form_hypotheses",
		Name:  "Form Hypotheses",
	}
	// This is the entry that tracks investigation of hypotheses via one-shot analysis
	ChangeTimelineEntryV2IDInvestigateHypotheses = ChangeTimelineEntryV2ID{
		Label: "investigate_hypotheses",
		Name:  "Investigate Hypotheses",
	}
)
