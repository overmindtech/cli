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
		"autoTaggingRuleSource":     cp.GetAutoTaggingRuleSource().ToMessage(),
		"skippedAutoTags":           cp.GetSkippedAutoTags(),
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

func (s ChangeProperties_AutoTaggingRuleSource) ToMessage() string {
	switch s {
	case ChangeProperties_AUTO_TAGGING_RULE_SOURCE_UNSPECIFIED:
		return "unknown"
	case ChangeProperties_AUTO_TAGGING_RULE_SOURCE_FILE:
		return "file"
	case ChangeProperties_AUTO_TAGGING_RULE_SOURCE_UI:
		return "ui"
	default:
		return "unknown"
	}
}

// allow custom auto tag rules to be passed on the cli, via a yaml file
//
// rules:
//   - name: Rule1
//     tag_key: "tag1"
//     enabled: true
//     instructions: "This is the instruction for Rule1"
//     valid_values: ["value1 with a space ", "value2"]
//   - name: Rule2
//     tag_key: "tag2"
//     enabled: false
//     instructions: "This is the instruction for Rule2"
//     valid_values: []
//   - name: "Rule3 with spaces"
//     tag_key: "tag3"
//     enabled: true
//     instructions: "This is the instruction for Rule3"
//     valid_values: ["value3", "value4", "value5"]
//   - name: "Rule4"
//     tag_key: "tag4"
//     enabled: true
//     instructions: "This is the instruction for Rule4"
//     valid_values:
//     ____- "value6"
//     ____- "value7"
//     ____- "value8"
//
// ____ are spaces above ^ formatter doesn't like spaces in the comment
// NB - the valid_values yaml is valid for a []strings or entries with a single string
// this is used by the cli for unmarshalling a file to rule properties
// and by the api server to marshal the rules from the database to yaml on export
type AutoTaggingRulesYaml struct {
	Rules []AutoTaggingRuleYAML `yaml:"rules"`
}

type AutoTaggingRuleYAML struct {
	Name         string   `yaml:"name"`
	TagKey       string   `yaml:"tag_key"`
	Enabled      bool     `yaml:"enabled"`
	Instructions string   `yaml:"instructions"`
	ValidValues  []string `yaml:"valid_values"`
}

type RoutineChangesConfigYAML struct {
	Sensitivity    float32 `yaml:"sensitivity"`
	DurationInDays float32 `yaml:"duration_in_days"`
	EventsPerDay   float32 `yaml:"events_per_day"`
}

// YamlStringToRuleProperties converts a yaml string to a slice of RuleProperties
func YamlStringToRuleProperties(yamlString string) ([]*RuleProperties, error) {
	var rulesYaml AutoTaggingRulesYaml
	err := yaml.Unmarshal([]byte(yamlString), &rulesYaml)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling yaml to rules: %w", err)
	}
	if len(rulesYaml.Rules) == 0 {
		return nil, errors.New("no rules found in yaml")
	}

	var rules []*RuleProperties
	for _, ruleYaml := range rulesYaml.Rules {
		rule := &RuleProperties{
			Name:         ruleYaml.Name,
			TagKey:       ruleYaml.TagKey,
			Enabled:      ruleYaml.Enabled,
			Instructions: ruleYaml.Instructions,
			ValidValues:  ruleYaml.ValidValues,
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// YamlStringToRoutineChangesConfig converts a yaml string to RoutineChangesConfig
func YamlStringToRoutineChangesConfig(yamlString string) (*RoutineChangesConfig, error) {
	var routineChangesConfigYAML RoutineChangesConfigYAML
	err := yaml.Unmarshal([]byte(yamlString), &routineChangesConfigYAML)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling yaml to routine changes config: %w", err)
	}
	if routineChangesConfigYAML.EventsPerDay < 1 {
		return nil, fmt.Errorf("events_per_day must be greater than 1, got %v", routineChangesConfigYAML.EventsPerDay)
	}
	if routineChangesConfigYAML.DurationInDays < 1 {
		return nil, fmt.Errorf("duration_in_days must be greater than 1, got %v", routineChangesConfigYAML.DurationInDays)
	}
	if routineChangesConfigYAML.Sensitivity < 0 {
		return nil, fmt.Errorf("sensitivity must be 0 or higher, got %v", routineChangesConfigYAML.Sensitivity)
	}
	routineChangesConfig := &RoutineChangesConfig{
		Sensitivity:   routineChangesConfigYAML.Sensitivity,
		EventsPer:     routineChangesConfigYAML.EventsPerDay,
		EventsPerUnit: RoutineChangesConfig_DAYS,
		Duration:      routineChangesConfigYAML.DurationInDays,
		DurationUnit:  RoutineChangesConfig_DAYS,
	}
	return routineChangesConfig, nil
}

// TimelineFindInProgressEntry returns the current running entry in the list of entries
// The function handles the following cases:
//   - If the input slice is nil or empty, it returns an error.
//   - The first entry that has a status of IN_PROGRESS, PENDING, or ERROR, it returns the entry's name, status, and a nil error.
//   - If an entry has an unknown status, it returns an error.
//   - If the timeline is complete it returns an empty string, DONE status, and a nil error.
func TimelineFindInProgressEntry(entries []*ChangeTimelineEntryV2) (string, ChangeTimelineEntryStatus, error) {
	if entries == nil {
		return "", ChangeTimelineEntryStatus_UNSPECIFIED, errors.New("entries is nil")
	}
	if len(entries) == 0 {
		return "", ChangeTimelineEntryStatus_UNSPECIFIED, errors.New("entries is empty")
	}

	for _, entry := range entries {
		switch entry.GetStatus() {
		case ChangeTimelineEntryStatus_IN_PROGRESS, ChangeTimelineEntryStatus_PENDING, ChangeTimelineEntryStatus_ERROR:
			// if the entry is in progress or about to start, or has an error(to be retried)
			return entry.GetName(), entry.GetStatus(), nil
		case ChangeTimelineEntryStatus_UNSPECIFIED, ChangeTimelineEntryStatus_DONE:
			// do nothing
		default:
			return "", ChangeTimelineEntryStatus_UNSPECIFIED, fmt.Errorf("unknown status: %s", entry.GetStatus().String())
		}
	}

	return "", ChangeTimelineEntryStatus_DONE, nil
}
