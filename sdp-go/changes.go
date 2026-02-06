package sdp

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

func (a *ChangeMetadata) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *ChangeMetadata) GetNullUUID() uuid.NullUUID {
	u := a.GetUUIDParsed()
	if u == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *u, Valid: true}
}

func (a *ChangeProperties) GetChangingItemsBookmarkUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetChangingItemsBookmarkUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *ChangeProperties) GetSystemBeforeSnapshotUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetSystemBeforeSnapshotUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *ChangeProperties) GetSystemAfterSnapshotUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetSystemAfterSnapshotUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *GetChangeRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *UpdateChangeRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *DeleteChangeRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *GetDiffRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *ListChangingItemsSummaryRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *StartChangeRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *EndChangeRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (c *Change) ToMap() map[string]any {
	return map[string]any{
		"metadata":   c.GetMetadata().ToMap(),
		"properties": c.GetProperties().ToMap(),
	}
}

func stringFromUuidBytes(b []byte) string {
	u, err := uuid.FromBytes(b)
	if err != nil {
		return ""
	}
	return u.String()
}

func (r *Reference) ToMap() map[string]any {
	return map[string]any{
		"type":                 r.GetType(),
		"uniqueAttributeValue": r.GetUniqueAttributeValue(),
		"scope":                r.GetScope(),
	}
}

func (r *Risk) ToMap() map[string]any {
	relatedItems := make([]map[string]any, len(r.GetRelatedItems()))
	for i, ri := range r.GetRelatedItems() {
		relatedItems[i] = ri.ToMap()
	}

	return map[string]any{
		"uuid":         stringFromUuidBytes(r.GetUUID()),
		"title":        r.GetTitle(),
		"severity":     r.GetSeverity().String(),
		"description":  r.GetDescription(),
		"relatedItems": relatedItems,
	}
}

func (r *GetChangeRisksResponse) ToMap() map[string]any {
	rmd := r.GetChangeRiskMetadata()
	risks := make([]map[string]any, len(rmd.GetRisks()))
	for i, ri := range rmd.GetRisks() {
		risks[i] = ri.ToMap()
	}

	return map[string]any{
		"risks":                risks,
		"numHighRisk":          rmd.GetNumHighRisk(),
		"numMediumRisk":        rmd.GetNumMediumRisk(),
		"numLowRisk":           rmd.GetNumLowRisk(),
		"changeAnalysisStatus": rmd.GetChangeAnalysisStatus().ToMap(),
	}
}

func (cm *ChangeMetadata) ToMap() map[string]any {
	return map[string]any{
		"UUID":                stringFromUuidBytes(cm.GetUUID()),
		"createdAt":           cm.GetCreatedAt().AsTime(),
		"updatedAt":           cm.GetUpdatedAt().AsTime(),
		"status":              cm.GetStatus().String(),
		"creatorName":         cm.GetCreatorName(),
		"numAffectedItems":    cm.GetNumAffectedItems(),
		"numAffectedEdges":    cm.GetNumAffectedEdges(),
		"numUnchangedItems":   cm.GetNumUnchangedItems(),
		"numCreatedItems":     cm.GetNumCreatedItems(),
		"numUpdatedItems":     cm.GetNumUpdatedItems(),
		"numDeletedItems":     cm.GetNumDeletedItems(),
		"UnknownHealthChange": cm.GetUnknownHealthChange(),
		"OkHealthChange":      cm.GetOkHealthChange(),
		"WarningHealthChange": cm.GetWarningHealthChange(),
		"ErrorHealthChange":   cm.GetErrorHealthChange(),
		"PendingHealthChange": cm.GetPendingHealthChange(),
	}
}

func (i *Item) ToMap() map[string]any {
	return map[string]any{
		"type":                 i.GetType(),
		"uniqueAttributeValue": i.UniqueAttributeValue(),
		"scope":                i.GetScope(),
		"attributes":           i.GetAttributes().GetAttrStruct().GetFields(),
	}
}

func (id *ItemDiff) ToMap() map[string]any {
	result := map[string]any{
		"status": id.GetStatus().String(),
	}
	if id.GetItem() != nil {
		result["item"] = id.GetItem().ToMap()
	}
	if id.GetBefore() != nil {
		result["before"] = id.GetBefore().ToMap()
	}
	if id.GetAfter() != nil {
		result["after"] = id.GetAfter().ToMap()
	}
	return result
}

func (id *ItemDiff) GloballyUniqueName() string {
	if id.GetItem() != nil {
		return id.GetItem().GloballyUniqueName()
	} else if id.GetBefore() != nil {
		return id.GetBefore().GloballyUniqueName()
	} else if id.GetAfter() != nil {
		return id.GetAfter().GloballyUniqueName()
	} else {
		return "empty item diff"
	}
}

func (cp *ChangeProperties) ToMap() map[string]any {
	plannedChanges := make([]map[string]any, len(cp.GetPlannedChanges()))
	for i, id := range cp.GetPlannedChanges() {
		plannedChanges[i] = id.ToMap()
	}

	return map[string]any{
		"title":                     cp.GetTitle(),
		"description":               cp.GetDescription(),
		"ticketLink":                cp.GetTicketLink(),
		"owner":                     cp.GetOwner(),
		"ccEmails":                  cp.GetCcEmails(),
		"changingItemsBookmarkUUID": stringFromUuidBytes(cp.GetChangingItemsBookmarkUUID()),
		"blastRadiusSnapshotUUID":   stringFromUuidBytes(cp.GetBlastRadiusSnapshotUUID()),
		"systemBeforeSnapshotUUID":  stringFromUuidBytes(cp.GetSystemBeforeSnapshotUUID()),
		"systemAfterSnapshotUUID":   stringFromUuidBytes(cp.GetSystemAfterSnapshotUUID()),
		"plannedChanges":            cp.GetPlannedChanges(),
		"rawPlan":                   cp.GetRawPlan(),
		"codeChanges":               cp.GetCodeChanges(),
		"repo":                      cp.GetRepo(),
		"tags":                      cp.GetEnrichedTags(),
	}
}

func (rcs *ChangeAnalysisStatus) ToMap() map[string]any {
	if rcs == nil {
		return map[string]any{}
	}

	return map[string]any{
		"status": rcs.GetStatus().String(),
	}
}

func (s StartChangeResponse_State) ToMessage() string {
	switch s {
	case StartChangeResponse_STATE_UNSPECIFIED:
		return "unknown"
	case StartChangeResponse_STATE_TAKING_SNAPSHOT:
		return "Snapshot is being taken"
	case StartChangeResponse_STATE_SAVING_SNAPSHOT:
		return "Snapshot is being saved"
	case StartChangeResponse_STATE_DONE:
		return "Everything is complete"
	default:
		return "unknown"
	}
}

func (s EndChangeResponse_State) ToMessage() string {
	switch s {
	case EndChangeResponse_STATE_UNSPECIFIED:
		return "unknown"
	case EndChangeResponse_STATE_TAKING_SNAPSHOT:
		return "Snapshot is being taken"
	case EndChangeResponse_STATE_SAVING_SNAPSHOT:
		return "Snapshot is being saved"
	case EndChangeResponse_STATE_DONE:
		return "Everything is complete"
	default:
		return "unknown"
	}
}

// RoutineChangesYAML represents the YAML structure for routine changes configuration.
// It defines parameters for detecting routine changes in infrastructure:
// - Sensitivity: Threshold for determining what constitutes a routine change (0 or higher)
// - DurationInDays: Time window in days to analyze for routine patterns (must be >= 1)
// - EventsPerDay: Expected number of change events per day for routine detection (must be >= 1)
type RoutineChangesYAML struct {
	Sensitivity    float32 `yaml:"sensitivity"`
	DurationInDays float32 `yaml:"duration_in_days"`
	EventsPerDay   float32 `yaml:"events_per_day"`
}

// GithubOrganisationYAML represents the YAML structure for GitHub organization profile configuration.
// It contains organization-specific settings such as the primary branch name used for
// change detection and analysis.
type GithubOrganisationYAML struct {
	PrimaryBranchName string `yaml:"primary_branch_name"`
}

// SignalConfigYAML represents the root YAML structure for signal configuration files.
// It can contain either or both of:
// - RoutineChangesConfig: Configuration for routine change detection
// - GithubOrganisationProfile: GitHub organization-specific settings
// At least one section must be provided in the YAML file.
type SignalConfigYAML struct {
	RoutineChangesConfig      *RoutineChangesYAML     `yaml:"routine_changes_config,omitempty"`
	GithubOrganisationProfile *GithubOrganisationYAML `yaml:"github_organisation_profile,omitempty"`
}

// SignalConfigFile represents the internal, parsed signal configuration structure.
// This is the converted form of SignalConfigYAML, where YAML-specific types are
// transformed into their corresponding protocol buffer types for use in the application.
type SignalConfigFile struct {
	RoutineChangesConfig      *RoutineChangesConfig
	GithubOrganisationProfile *GithubOrganisationProfile
}

// YamlStringToSignalConfig parses a YAML string containing signal configuration and converts it
// into a SignalConfigFile. It validates that at least one configuration section is provided
// and performs validation on the routine changes configuration if present.
//
// The function handles conversion from YAML-friendly types (e.g., float32 for durations)
// to the internal protocol buffer types (e.g., RoutineChangesConfig with unit specifications).
//
// Returns an error if:
// - The YAML is invalid or cannot be unmarshaled
// - No configuration sections are provided
// - Routine changes configuration validation fails
func YamlStringToSignalConfig(yamlString string) (*SignalConfigFile, error) {
	var signalConfigYAML SignalConfigYAML
	err := yaml.Unmarshal([]byte(yamlString), &signalConfigYAML)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling yaml to signal config: %w", err)
	}

	// check that at least one section is provided
	if signalConfigYAML.RoutineChangesConfig == nil && signalConfigYAML.GithubOrganisationProfile == nil {
		return nil, fmt.Errorf("signal config file must contain at least one of: routine_changes_config or github_organisation_profile")
	}

	// validate the routine changes config
	if signalConfigYAML.RoutineChangesConfig != nil {
		if err := validateRoutineChangesConfig(signalConfigYAML.RoutineChangesConfig); err != nil {
			return nil, err
		}
	}

	var routineCfg *RoutineChangesConfig
	if signalConfigYAML.RoutineChangesConfig != nil {
		routineCfg = &RoutineChangesConfig{
			Sensitivity:   signalConfigYAML.RoutineChangesConfig.Sensitivity,
			EventsPer:     signalConfigYAML.RoutineChangesConfig.EventsPerDay,
			EventsPerUnit: RoutineChangesConfig_DAYS,
			Duration:      signalConfigYAML.RoutineChangesConfig.DurationInDays,
			DurationUnit:  RoutineChangesConfig_DAYS,
		}
	}

	var githubProfile *GithubOrganisationProfile
	if signalConfigYAML.GithubOrganisationProfile != nil {
		githubProfile = &GithubOrganisationProfile{
			PrimaryBranchName: signalConfigYAML.GithubOrganisationProfile.PrimaryBranchName,
		}
	}

	signalConfigFile := &SignalConfigFile{
		RoutineChangesConfig:      routineCfg,
		GithubOrganisationProfile: githubProfile,
	}
	return signalConfigFile, nil
}

// validateRoutineChangesConfig validates the routine changes configuration values.
// It ensures that:
// - EventsPerDay is at least 1
// - DurationInDays is at least 1
// - Sensitivity is 0 or higher
//
// Returns an error with a descriptive message if any validation fails.
func validateRoutineChangesConfig(routineChangesConfigYAML *RoutineChangesYAML) error {
	if routineChangesConfigYAML.EventsPerDay < 1 {
		return fmt.Errorf("events_per_day must be greater than 1, got %v", routineChangesConfigYAML.EventsPerDay)
	}
	if routineChangesConfigYAML.DurationInDays < 1 {
		return fmt.Errorf("duration_in_days must be greater than 1, got %v", routineChangesConfigYAML.DurationInDays)
	}
	if routineChangesConfigYAML.Sensitivity < 0 {
		return fmt.Errorf("sensitivity must be 0 or higher, got %v", routineChangesConfigYAML.Sensitivity)
	}
	return nil
}

// TimelineEntryContentDescription returns a human-readable description of the
// entry's content based on its type.
func TimelineEntryContentDescription(entry *ChangeTimelineEntryV2) string {
	switch c := entry.GetContent().(type) {
	case *ChangeTimelineEntryV2_MappedItems:
		return fmt.Sprintf("%d mapped items", len(c.MappedItems.GetMappedItems()))
	case *ChangeTimelineEntryV2_CalculatedBlastRadius:
		return fmt.Sprintf("%d items, %d edges", c.CalculatedBlastRadius.GetNumItems(), c.CalculatedBlastRadius.GetNumEdges())
	case *ChangeTimelineEntryV2_CalculatedRisks:
		return fmt.Sprintf("%d risks", len(c.CalculatedRisks.GetRisks()))
	case *ChangeTimelineEntryV2_CalculatedLabels:
		return fmt.Sprintf("%d labels", len(c.CalculatedLabels.GetLabels()))
	case *ChangeTimelineEntryV2_ChangeValidation:
		return fmt.Sprintf("%d validation categories", len(c.ChangeValidation.GetValidationChecklist()))
	case *ChangeTimelineEntryV2_FormHypotheses:
		return fmt.Sprintf("%d hypotheses", c.FormHypotheses.GetNumHypotheses())
	case *ChangeTimelineEntryV2_InvestigateHypotheses:
		return fmt.Sprintf("%d proven, %d disproven, %d investigating",
			c.InvestigateHypotheses.GetNumProven(),
			c.InvestigateHypotheses.GetNumDisproven(),
			c.InvestigateHypotheses.GetNumInvestigating())
	case *ChangeTimelineEntryV2_RecordObservations:
		return fmt.Sprintf("%d observations", c.RecordObservations.GetNumObservations())
	case *ChangeTimelineEntryV2_Error:
		return c.Error
	case *ChangeTimelineEntryV2_StatusMessage:
		return c.StatusMessage
	case *ChangeTimelineEntryV2_Empty, nil:
		return ""
	default:
		return ""
	}
}

// TimelineFindInProgressEntry returns the current running entry in the list of entries
// The function handles the following cases:
//   - If the input slice is nil or empty, it returns an error.
//   - The first entry that has a status of IN_PROGRESS, PENDING, or ERROR, it returns the entry's name, content description, status, and a nil error.
//   - If an entry has an unknown status, it returns an error.
//   - If the timeline is complete it returns an empty string, empty content description, DONE status, and a nil error.
func TimelineFindInProgressEntry(entries []*ChangeTimelineEntryV2) (string, string, ChangeTimelineEntryStatus, error) {
	if entries == nil {
		return "", "", ChangeTimelineEntryStatus_UNSPECIFIED, errors.New("entries is nil")
	}
	if len(entries) == 0 {
		return "", "", ChangeTimelineEntryStatus_UNSPECIFIED, errors.New("entries is empty")
	}

	for _, entry := range entries {
		switch entry.GetStatus() {
		case ChangeTimelineEntryStatus_IN_PROGRESS, ChangeTimelineEntryStatus_PENDING, ChangeTimelineEntryStatus_ERROR:
			// if the entry is in progress or about to start, or has an error(to be retried)
			return entry.GetName(), TimelineEntryContentDescription(entry), entry.GetStatus(), nil
		case ChangeTimelineEntryStatus_UNSPECIFIED, ChangeTimelineEntryStatus_DONE:
			// do nothing
		default:
			return "", "", ChangeTimelineEntryStatus_UNSPECIFIED, fmt.Errorf("unknown status: %s", entry.GetStatus().String())
		}
	}

	return "", "", ChangeTimelineEntryStatus_DONE, nil
}
