package sdp

type ChangeTimelineEntryV2Name string

// If you add/delete/move an entry here, make sure to update/check the following:
// - the PopulateChangeTimelineV2 function
// - GetChangeTimelineV2 in api-server/server/changesservice.go
// - resetChangeAnalysisTables in api-server/server/changeanalysis/shared.go
// - the cli tool if we are waiting for a change analysis to finish
const (
	ChangeTimelineEntryV2NameChangeCreated         ChangeTimelineEntryV2Name = "Change Created"
	ChangeTimelineEntryV2NameMappedResources       ChangeTimelineEntryV2Name = "Mapped Resources"
	ChangeTimelineEntryV2NameCalculatedBlastRadius ChangeTimelineEntryV2Name = "Calculated Blast Radius"
	ChangeTimelineEntryV2NameCalculatedRisks       ChangeTimelineEntryV2Name = "Calculated Risks"
	ChangeTimelineEntryV2NameAutoTagging           ChangeTimelineEntryV2Name = "Auto Tagging"
	ChangeTimelineEntryV2NameChangeValidation      ChangeTimelineEntryV2Name = "Change Validation"
	ChangeTimelineEntryV2NameChangeStarted         ChangeTimelineEntryV2Name = "Change Started"
	ChangeTimelineEntryV2NameChangeFinished        ChangeTimelineEntryV2Name = "Change Finished"
)
