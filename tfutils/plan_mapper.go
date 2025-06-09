package tfutils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	awsAdapters "github.com/overmindtech/cli/aws-source/adapters"
	k8sAdapters "github.com/overmindtech/cli/k8s-source/adapters"
	"github.com/overmindtech/cli/sdp-go"
	gcpAdapters "github.com/overmindtech/cli/sources/gcp/proc"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MapStatus int

func (m MapStatus) String() string {
	switch m {
	case MapStatusSuccess:
		return "success"
	case MapStatusNotEnoughInfo:
		return "not enough info"
	case MapStatusUnsupported:
		return "unsupported"
	default:
		return "unknown"
	}
}

const (
	MapStatusSuccess MapStatus = iota
	MapStatusNotEnoughInfo
	MapStatusUnsupported
)

const KnownAfterApply = `(known after apply)`

type PlannedChangeMapResult struct {
	// The full name of the resource in the Terraform plan
	TerraformName string

	// The terraform resource type
	TerraformType string

	// The status of the mapping
	Status MapStatus

	// The message that should be printed next to the status e.g. "mapped" or
	// "missing arn"
	Message string

	*sdp.MappedItemDiff
}

type PlanMappingResult struct {
	Results        []PlannedChangeMapResult
	RemovedSecrets int
}

func (r *PlanMappingResult) NumSuccess() int {
	return r.numStatus(MapStatusSuccess)
}

func (r *PlanMappingResult) NumNotEnoughInfo() int {
	return r.numStatus(MapStatusNotEnoughInfo)
}

func (r *PlanMappingResult) NumUnsupported() int {
	return r.numStatus(MapStatusUnsupported)
}

func (r *PlanMappingResult) NumTotal() int {
	return len(r.Results)
}

func (r *PlanMappingResult) GetItemDiffs() []*sdp.MappedItemDiff {
	diffs := make([]*sdp.MappedItemDiff, 0)

	for _, result := range r.Results {
		if result.MappedItemDiff != nil {
			diffs = append(diffs, result.MappedItemDiff)
		}
	}

	return diffs
}

func (r *PlanMappingResult) numStatus(status MapStatus) int {
	count := 0
	for _, result := range r.Results {
		if result.Status == status {
			count++
		}
	}
	return count
}

func MappedItemDiffsFromPlanFile(ctx context.Context, fileName string, lf log.Fields) (*PlanMappingResult, error) {
	// read results from `terraform show -json ${tfplan file}`
	planJSON, err := os.ReadFile(fileName)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to read terraform plan")
		return nil, err
	}

	return MappedItemDiffsFromPlan(ctx, planJSON, fileName, lf)
}

type TfMapData struct {
	// The overmind type name
	OvermindType string

	// The method that the query should use
	Method sdp.QueryMethod

	// The field within the resource that should be queried for
	QueryField string
}

// MappedItemDiffsFromPlan takes a plan JSON, file name, and log fields as input
// and returns the mapping results and an error. It parses the plan JSON,
// extracts resource changes, and creates mapped item differences for each
// resource change. It also generates mapping queries based on the resource type
// and current resource values. The function categorizes the mapped item
// differences into supported and unsupported changes. Finally, it logs the
// number of supported and unsupported changes and returns the mapped item
// differences.
func MappedItemDiffsFromPlan(ctx context.Context, planJson []byte, fileName string, lf log.Fields) (*PlanMappingResult, error) {
	// Create a span for this since we're going to be attaching events to it when things fail to map
	span := trace.SpanFromContext(ctx)
	defer span.End()

	// Check that we haven't been passed a state file
	if isStateFile(planJson) {
		return nil, fmt.Errorf("'%v' appears to be a state file, not a plan file", fileName)
	}

	// Load mapping data from the sources and convert into a map so that we can
	// index by Terraform type
	adapterMetadata := awsAdapters.Metadata.AllAdapterMetadata()
	adapterMetadata = append(adapterMetadata, k8sAdapters.Metadata.AllAdapterMetadata()...)
	adapterMetadata = append(adapterMetadata, gcpAdapters.Metadata.AllAdapterMetadata()...)
	// These mappings are from the terraform type, to required mapping data
	mappings := make(map[string][]TfMapData)
	for _, metadata := range adapterMetadata {
		if metadata.GetType() == "" {
			continue
		}

		for _, mapping := range metadata.GetTerraformMappings() {
			// Extract the query field and type from the mapping
			subs := strings.SplitN(mapping.GetTerraformQueryMap(), ".", 2)
			if len(subs) != 2 {
				log.WithContext(ctx).WithFields(lf).WithField("terraform-query-map", mapping.GetTerraformQueryMap()).Warn("Skipping mapping with invalid query map")
				continue
			}
			terraformType := subs[0]
			queryField := subs[1]

			// Add the mapping details
			mappings[terraformType] = append(mappings[terraformType], TfMapData{
				OvermindType: metadata.GetType(),
				Method:       mapping.GetTerraformMethod(),
				QueryField:   queryField,
			})
		}
	}

	var plan Plan
	err := json.Unmarshal(planJson, &plan)
	if err != nil {
		return nil, fmt.Errorf("failed to parse '%v': %w", fileName, err)
	}

	results := PlanMappingResult{
		Results:        make([]PlannedChangeMapResult, 0),
		RemovedSecrets: countSensitiveValuesInConfig(plan.Config.RootModule) + countSensitiveValuesInState(plan.PlannedValues.RootModule),
	}

	// for all managed resources:
	for _, resourceChange := range plan.ResourceChanges {
		if len(resourceChange.Change.Actions) == 0 || resourceChange.Change.Actions[0] == "no-op" || resourceChange.Mode == "data" {
			// skip resources with no changes and data updates
			continue
		}

		itemDiff, err := itemDiffFromResourceChange(resourceChange)
		if err != nil {
			return nil, fmt.Errorf("failed to create item diff for resource change: %w", err)
		}

		// Get the Terraform mappings for this specific type
		relevantMappings, ok := mappings[resourceChange.Type]
		if !ok {
			log.WithContext(ctx).WithFields(lf).WithField("terraform-address", resourceChange.Address).Debug("Skipping unmapped resource")
			results.Results = append(results.Results, PlannedChangeMapResult{
				TerraformName: resourceChange.Address,
				TerraformType: resourceChange.Type,
				Status:        MapStatusUnsupported,
				Message:       "unsupported",
				MappedItemDiff: &sdp.MappedItemDiff{
					Item:         itemDiff,
					MappingQuery: nil, // unmapped item has no mapping query
				},
			})
			continue
		}

		var currentResource *Resource

		// Look for the resource in the prior values first, since this is
		// the *previous* state we're like to be able to find it in the
		// actual infra
		if plan.PriorState.Values != nil {
			currentResource = plan.PriorState.Values.RootModule.DigResource(resourceChange.Address)
		}

		// If we didn't find it, look in the planned values
		if currentResource == nil {
			currentResource = plan.PlannedValues.RootModule.DigResource(resourceChange.Address)
		}

		if currentResource == nil {
			log.WithContext(ctx).
				WithFields(lf).
				WithField("terraform-address", resourceChange.Address).
				Warn("Skipping resource without values")
			continue
		}

		results.Results = append(results.Results, mapResourceToQuery(itemDiff, currentResource, relevantMappings))
	}

	// Attach failed mappings to the span
	for _, result := range results.Results {
		switch result.Status {
		case MapStatusUnsupported, MapStatusNotEnoughInfo:
			span.AddEvent("UnmappedResource", trace.WithAttributes(
				attribute.String("ovm.climap.status", result.Status.String()),
				attribute.String("ovm.climap.message", result.Message),
				attribute.String("ovm.climap.terraform-name", result.TerraformName),
				attribute.String("ovm.climap.terraform-type", result.TerraformType),
			))
		case MapStatusSuccess:
			// Don't include these
		}
	}

	return &results, nil
}

// Maps a resource to an Overmind query, or at least tries to given the provided
// mappings. If there are multiple valid queries, the first one will be used.
//
// In the future we might allow for multiple queries to be returned, this work
// will be tracked here: https://github.com/overmindtech/workspace/sdp/issues/272
func mapResourceToQuery(itemDiff *sdp.ItemDiff, terraformResource *Resource, mappings []TfMapData) PlannedChangeMapResult {
	attemptedMappings := make([]string, 0)

	if len(mappings) == 0 {
		return PlannedChangeMapResult{
			TerraformName: terraformResource.Address,
			TerraformType: terraformResource.Type,
			Status:        MapStatusUnsupported,
			Message:       "unsupported",
			MappedItemDiff: &sdp.MappedItemDiff{
				Item:         itemDiff,
				MappingQuery: nil, // unmapped item has no mapping query
			},
		}
	}

	for _, mapping := range mappings {
		// See if the query field exists in the resource. If it doesn't then we
		// will continue to the next mapping
		query, ok := terraformResource.AttributeValues.Dig(mapping.QueryField)
		if ok {
			// If the query field exists, we will create a query
			u := uuid.New()
			newQuery := &sdp.Query{
				Type:               mapping.OvermindType,
				Method:             mapping.Method,
				Query:              fmt.Sprintf("%v", query),
				Scope:              "*",
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
				UUID:               u[:],
				Deadline:           timestamppb.New(time.Now().Add(60 * time.Second)),
			}

			// Set the type of item to the Overmind-supported type rather than
			// the Terraform one
			if itemDiff.GetBefore() != nil {
				itemDiff.Before.Type = mapping.OvermindType
			}
			if itemDiff.GetAfter() != nil {
				itemDiff.After.Type = mapping.OvermindType
			}

			return PlannedChangeMapResult{
				TerraformName: terraformResource.Address,
				TerraformType: terraformResource.Type,
				Status:        MapStatusSuccess,
				Message:       "mapped",
				MappedItemDiff: &sdp.MappedItemDiff{
					Item:         itemDiff,
					MappingQuery: newQuery,
				},
			}
		}

		// It it wasn't successful, add the mapping to the list of attempted
		// mappings
		attemptedMappings = append(attemptedMappings, mapping.QueryField)
	}

	// If we get to this point, we haven't found a mapping
	return PlannedChangeMapResult{
		TerraformName: terraformResource.Address,
		TerraformType: terraformResource.Type,
		Status:        MapStatusNotEnoughInfo,
		Message:       fmt.Sprintf("missing mapping attribute: %v", strings.Join(attemptedMappings, ", ")),
		MappedItemDiff: &sdp.MappedItemDiff{
			Item:         itemDiff,
			MappingQuery: nil, // unmapped item has no mapping query
		},
	}
}

// Checks if the supplied JSON bytes are a state file. It's a common  mistake to
// pass a state file to Overmind rather than a plan file since the commands to
// create them are similar
func isStateFile(bytes []byte) bool {
	fields := make(map[string]interface{})

	err := json.Unmarshal(bytes, &fields)
	if err != nil {
		return false
	}

	if _, exists := fields["values"]; exists {
		return true
	}

	return false
}

func countSensitiveValuesInConfig(m ConfigModule) int {
	removedSecrets := 0
	for _, v := range m.Variables {
		if v.Sensitive {
			removedSecrets++
		}
	}
	for _, o := range m.Outputs {
		if o.Sensitive {
			removedSecrets++
		}
	}
	for _, c := range m.ModuleCalls {
		removedSecrets += countSensitiveValuesInConfig(c.Module)
	}
	return removedSecrets
}

func countSensitiveValuesInState(m Module) int {
	removedSecrets := 0
	for _, r := range m.Resources {
		removedSecrets += countSensitiveValuesInResource(r)
	}
	for _, c := range m.ChildModules {
		removedSecrets += countSensitiveValuesInState(c)
	}
	return removedSecrets
}

// follow itemAttributesFromResourceChangeData and maskSensitiveData
// implementation to count sensitive values
func countSensitiveValuesInResource(r Resource) int {
	// sensitiveMsg can be a bool or a map[string]any
	var isSensitive bool
	err := json.Unmarshal(r.SensitiveValues, &isSensitive)
	if err == nil && isSensitive {
		return 1 // one very large secret
	} else if err != nil {
		// only try parsing as map if parsing as bool failed
		var sensitive map[string]any
		err = json.Unmarshal(r.SensitiveValues, &sensitive)
		if err != nil {
			return 0
		}
		return countSensitiveAttributes(r.AttributeValues, sensitive)
	}
	return 0
}

func countSensitiveAttributes(attributes, sensitive any) int {
	if sensitive == true {
		return 1
	} else if sensitiveMap, ok := sensitive.(map[string]any); ok {
		if attributesMap, ok := attributes.(map[string]any); ok {
			result := 0
			for k, v := range attributesMap {
				result += countSensitiveAttributes(v, sensitiveMap[k])
			}
			return result
		} else {
			return 1
		}
	} else if sensitiveArr, ok := sensitive.([]any); ok {
		if attributesArr, ok := attributes.([]any); ok {
			if len(sensitiveArr) != len(attributesArr) {
				return 1
			}
			result := 0
			for i, v := range attributesArr {
				result += countSensitiveAttributes(v, sensitiveArr[i])
			}
			return result
		} else {
			return 1
		}
	}
	return 0
}

// Converts a ResourceChange form a terraform plan to an ItemDiff in SDP format.
// These items will use the scope `terraform_plan` since we haven't mapped them
// to an actual item in the infrastructure yet
func itemDiffFromResourceChange(resourceChange ResourceChange) (*sdp.ItemDiff, error) {
	status := sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNSPECIFIED

	if slices.Equal(resourceChange.Change.Actions, []string{"no-op"}) || slices.Equal(resourceChange.Change.Actions, []string{"read"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNCHANGED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"create"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_CREATED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"update"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"delete", "create"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"create", "delete"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED
	} else if slices.Equal(resourceChange.Change.Actions, []string{"delete"}) {
		status = sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED
	}

	beforeAttributes, err := itemAttributesFromResourceChangeData(resourceChange.Change.Before, resourceChange.Change.BeforeSensitive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse before attributes: %w", err)
	}
	afterAttributes, err := itemAttributesFromResourceChangeData(resourceChange.Change.After, resourceChange.Change.AfterSensitive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse after attributes: %w", err)
	}

	err = handleKnownAfterApply(beforeAttributes, afterAttributes, resourceChange.Change.AfterUnknown)
	if err != nil {
		return nil, fmt.Errorf("failed to remove known after apply fields: %w", err)
	}

	result := &sdp.ItemDiff{
		// Item: filled in by item mapping in UpdatePlannedChanges
		Status: status,
	}

	// shorten the address by removing the type prefix if and only if it is the
	// first part. Longer terraform addresses created in modules will not be
	// shortened to avoid confusion.
	trimmedAddress, _ := strings.CutPrefix(resourceChange.Address, fmt.Sprintf("%v.", resourceChange.Type))

	if beforeAttributes != nil {
		result.Before = &sdp.Item{
			Type:            resourceChange.Type,
			UniqueAttribute: "terraform_name",
			Attributes:      beforeAttributes,
			Scope:           "terraform_plan",
		}

		err = result.GetBefore().GetAttributes().Set("terraform_name", trimmedAddress)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_name '%v' on before attributes: %w", trimmedAddress, err))
		}

		err = result.GetBefore().GetAttributes().Set("terraform_address", resourceChange.Address)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_address of type %T (%v) on before attributes: %w", resourceChange.Address, resourceChange.Address, err))
		}
	}

	if afterAttributes != nil {
		result.After = &sdp.Item{
			Type:            resourceChange.Type,
			UniqueAttribute: "terraform_name",
			Attributes:      afterAttributes,
			Scope:           "terraform_plan",
		}

		err = result.GetAfter().GetAttributes().Set("terraform_name", trimmedAddress)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_name '%v' on after attributes: %w", trimmedAddress, err))
		}

		err = result.GetAfter().GetAttributes().Set("terraform_address", resourceChange.Address)
		if err != nil {
			// since Address is a string, this should never happen
			sentry.CaptureException(fmt.Errorf("failed to set terraform_address of type %T (%v) on after attributes: %w", resourceChange.Address, resourceChange.Address, err))
		}
	}

	return result, nil
}

func itemAttributesFromResourceChangeData(attributesMsg, sensitiveMsg json.RawMessage) (*sdp.ItemAttributes, error) {
	var attributes map[string]any
	err := json.Unmarshal(attributesMsg, &attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse attributes: %w", err)
	}

	// sensitiveMsg can be a bool or a map[string]any
	var isSensitive bool
	err = json.Unmarshal(sensitiveMsg, &isSensitive)
	if err == nil && isSensitive {
		attributes = maskAllData(attributes)
	} else if err != nil {
		// only try parsing as map if parsing as bool failed
		var sensitive map[string]any
		err = json.Unmarshal(sensitiveMsg, &sensitive)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sensitive: %w", err)
		}
		attributes = maskSensitiveData(attributes, sensitive).(map[string]any)
	}

	return sdp.ToAttributesSorted(attributes)
}

// maskAllData masks every entry in attributes as redacted
func maskAllData(attributes map[string]any) map[string]any {
	for k, v := range attributes {
		if mv, ok := v.(map[string]any); ok {
			attributes[k] = maskAllData(mv)
		} else {
			attributes[k] = "(sensitive value)"
		}
	}
	return attributes
}

// maskSensitiveData masks every entry in attributes that is set to true in sensitive. returns the redacted attributes
func maskSensitiveData(attributes, sensitive any) any {
	if sensitive == true {
		return "(sensitive value)"
	} else if sensitiveMap, ok := sensitive.(map[string]any); ok {
		if attributesMap, ok := attributes.(map[string]any); ok {
			result := map[string]any{}
			for k, v := range attributesMap {
				result[k] = maskSensitiveData(v, sensitiveMap[k])
			}
			return result
		} else {
			return "(sensitive value) (type mismatch)"
		}
	} else if sensitiveArr, ok := sensitive.([]any); ok {
		if attributesArr, ok := attributes.([]any); ok {
			if len(sensitiveArr) != len(attributesArr) {
				return "(sensitive value) (len mismatch)"
			}
			result := make([]any, len(attributesArr))
			for i, v := range attributesArr {
				result[i] = maskSensitiveData(v, sensitiveArr[i])
			}
			return result
		} else {
			return "(sensitive value) (type mismatch)"
		}
	}
	return attributes
}

// Finds fields from the `before` and `after` attributes that are known after
// apply and replaces the "after" value with the string "(known after apply)"
func handleKnownAfterApply(before, after *sdp.ItemAttributes, afterUnknown json.RawMessage) error {
	var afterUnknownInterface interface{}
	err := json.Unmarshal(afterUnknown, &afterUnknownInterface)
	if err != nil {
		return fmt.Errorf("could not unmarshal `after_unknown` from plan: %w", err)
	}

	// Convert the parent struct to a value so that we can treat them all the
	// same when we recurse
	beforeValue := structpb.Value{
		Kind: &structpb.Value_StructValue{
			StructValue: before.GetAttrStruct(),
		},
	}

	afterValue := structpb.Value{
		Kind: &structpb.Value_StructValue{
			StructValue: after.GetAttrStruct(),
		},
	}

	err = insertKnownAfterApply(&beforeValue, &afterValue, afterUnknownInterface)

	if err != nil {
		return fmt.Errorf("failed to remove known after apply fields: %w", err)
	}

	return nil
}

// Inserts the text "(known after apply)" in place of null values in the planned
// "after" values for fields that are known after apply. By default these are
// `null` which produces a bad diff, so we replace them with (known after apply)
// to more accurately mirror what Terraform does in the CLI
func insertKnownAfterApply(before, after *structpb.Value, afterUnknown interface{}) error {
	switch afterUnknown := afterUnknown.(type) {
	case map[string]interface{}:
		for k, v := range afterUnknown {
			if v == true {
				if afterFields := after.GetStructValue().GetFields(); afterFields != nil {
					// Insert this in the after fields even if it doesn't exist.
					// This is because sometimes you will get a plan that only
					// has a before value for a know after apply field, so we
					// want to still make sure it shows up
					afterFields[k] = &structpb.Value{
						Kind: &structpb.Value_StringValue{
							StringValue: KnownAfterApply,
						},
					}
				}
			} else if v == false {
				// Do nothing
				continue
			} else {
				// Recurse into the nested fields
				err := insertKnownAfterApply(before.GetStructValue().GetFields()[k], after.GetStructValue().GetFields()[k], v)
				if err != nil {
					return err
				}
			}
		}
	case []interface{}:
		for i, v := range afterUnknown {
			if v == true {
				// If this value in a slice is true, set the corresponding value
				// in after to (know after apply)
				if after.GetListValue() != nil && len(after.GetListValue().GetValues()) > i {
					after.GetListValue().Values[i] = &structpb.Value{
						Kind: &structpb.Value_StringValue{
							StringValue: KnownAfterApply,
						},
					}
				}
			} else if v == false {
				// Do nothing
				continue
			} else {
				// Make sure that the before and after both actually have a
				// valid list item at this position, if they don't we can just
				// pass `nil` to the `removeUnknownFields` function and it'll
				// handle it
				beforeListValues := before.GetListValue().GetValues()
				afterListValues := after.GetListValue().GetValues()
				var nestedBeforeValue *structpb.Value
				var nestedAfterValue *structpb.Value

				if len(beforeListValues) > i {
					nestedBeforeValue = beforeListValues[i]
				}

				if len(afterListValues) > i {
					nestedAfterValue = afterListValues[i]
				}

				err := insertKnownAfterApply(nestedBeforeValue, nestedAfterValue, v)
				if err != nil {
					return err
				}
			}
		}
	default:
		return nil
	}

	return nil
}
