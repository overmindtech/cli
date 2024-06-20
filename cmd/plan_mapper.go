package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/cmd/datamaps"
	"github.com/overmindtech/sdp-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MapStatus int

const (
	MapStatusSuccess MapStatus = iota
	MapStatusNotEnoughInfo
	MapStatusUnsupported
)

type PlannedChangeMapResult struct {
	// The name of the resource in the Terraform plan
	TerraformName string

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

func mappedItemDiffsFromPlanFile(ctx context.Context, fileName string, lf log.Fields) (*PlanMappingResult, error) {
	// read results from `terraform show -json ${tfplan file}`
	planJSON, err := os.ReadFile(fileName)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithFields(lf).Error("Failed to read terraform plan")
		return nil, err
	}

	return mappedItemDiffsFromPlan(ctx, planJSON, fileName, lf)
}

// mappedItemDiffsFromPlan takes a plan JSON, file name, and log fields as input
// and returns the mapping results and an error. It parses the plan JSON,
// extracts resource changes, and creates mapped item differences for each
// resource change. It also generates mapping queries based on the resource type
// and current resource values. The function categorizes the mapped item
// differences into supported and unsupported changes. Finally, it logs the
// number of supported and unsupported changes and returns the mapped item
// differences.
func mappedItemDiffsFromPlan(ctx context.Context, planJson []byte, fileName string, lf log.Fields) (*PlanMappingResult, error) {
	// Check that we haven't been passed a state file
	if isStateFile(planJson) {
		return nil, fmt.Errorf("'%v' appears to be a state file, not a plan file", fileName)
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

		// Load mappings for this type. These mappings tell us how to create an
		// SDP query that will return this resource
		awsMappings := datamaps.AwssourceData[resourceChange.Type]
		k8sMappings := datamaps.K8ssourceData[resourceChange.Type]
		mappings := append(awsMappings, k8sMappings...)

		if len(mappings) == 0 {
			log.WithContext(ctx).WithFields(lf).WithField("terraform-address", resourceChange.Address).Debug("Skipping unmapped resource")
			results.Results = append(results.Results, PlannedChangeMapResult{
				TerraformName: resourceChange.Address,
				Status:        MapStatusUnsupported,
				Message:       "unsupported",
				MappedItemDiff: &sdp.MappedItemDiff{
					Item:         itemDiff,
					MappingQuery: nil, // unmapped item has no mapping query
				},
			})
			continue
		}

		for _, mapData := range mappings {
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
					WithField("terraform-query-field", mapData.QueryField).Warn("Skipping resource without values")
				continue
			}

			query, ok := currentResource.AttributeValues.Dig(mapData.QueryField)
			if !ok {
				log.WithContext(ctx).
					WithFields(lf).
					WithField("terraform-address", resourceChange.Address).
					WithField("terraform-query-field", mapData.QueryField).Warn("Adding unmapped resource")
				results.Results = append(results.Results, PlannedChangeMapResult{
					TerraformName: resourceChange.Address,
					Status:        MapStatusNotEnoughInfo,
					Message:       fmt.Sprintf("missing %v", mapData.QueryField),
					MappedItemDiff: &sdp.MappedItemDiff{
						Item:         itemDiff,
						MappingQuery: nil, // unmapped item has no mapping query
					},
				})
				continue
			}

			// Create the map that variables will pull data from
			dataMap := make(map[string]any)

			// Populate resource values
			dataMap["values"] = currentResource.AttributeValues

			if overmindMappingsOutput, ok := plan.PlannedValues.Outputs["overmind_mappings"]; ok {
				configResource := plan.Config.RootModule.DigResource(resourceChange.Address)

				if configResource == nil {
					log.WithContext(ctx).
						WithFields(lf).
						WithField("terraform-address", resourceChange.Address).
						Debug("Skipping provider mapping for resource without config")
				} else {
					// Look up the provider config key in the mappings
					mappings := make(map[string]map[string]string)

					err = json.Unmarshal(overmindMappingsOutput.Value, &mappings)

					if err != nil {
						log.WithContext(ctx).
							WithFields(lf).
							WithField("terraform-address", resourceChange.Address).
							WithError(err).
							Error("Failed to parse overmind_mappings output")
					} else {
						// We need to split out the module section of the name
						// here. If the resource isn't in a module, the
						// ProviderConfigKey will be something like
						// "kubernetes", however if it's in a module it's be
						// something like "module.something:kubernetes"
						providerName := extractProviderNameFromConfigKey(configResource.ProviderConfigKey)
						currentProviderMappings, ok := mappings[providerName]

						if ok {
							log.WithContext(ctx).
								WithFields(lf).
								WithField("terraform-address", resourceChange.Address).
								WithField("provider-config-key", configResource.ProviderConfigKey).
								Debug("Found provider mappings")

							// We have mappings for this provider, so set them
							// in the `provider_mapping` value
							dataMap["provider_mapping"] = currentProviderMappings
						}
					}
				}
			}

			// Interpolate variables in the scope
			scope, err := InterpolateScope(mapData.Scope, dataMap)

			if err != nil {
				log.WithContext(ctx).WithError(err).Debugf("Could not find scope mapping variables %v, adding them will result in better results. Error: ", mapData.Scope)
				scope = "*"
			}

			u := uuid.New()
			newQuery := &sdp.Query{
				Type:               mapData.Type,
				Method:             mapData.Method,
				Query:              fmt.Sprintf("%v", query),
				Scope:              scope,
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
				UUID:               u[:],
				Deadline:           timestamppb.New(time.Now().Add(60 * time.Second)),
			}

			// cleanup item metadata from mapping query
			if itemDiff.GetBefore() != nil {
				itemDiff.Before.Type = newQuery.GetType()
				if newQuery.GetScope() != "*" {
					itemDiff.Before.Scope = newQuery.GetScope()
				}
			}

			// cleanup item metadata from mapping query
			if itemDiff.GetAfter() != nil {
				itemDiff.After.Type = newQuery.GetType()
				if newQuery.GetScope() != "*" {
					itemDiff.After.Scope = newQuery.GetScope()
				}
			}

			results.Results = append(results.Results, PlannedChangeMapResult{
				TerraformName: resourceChange.Address,
				Status:        MapStatusSuccess,
				Message:       "mapped",
				MappedItemDiff: &sdp.MappedItemDiff{
					Item:         itemDiff,
					MappingQuery: newQuery,
				},
			})

			log.WithContext(ctx).WithFields(log.Fields{
				"scope":  newQuery.GetScope(),
				"type":   newQuery.GetType(),
				"query":  newQuery.GetQuery(),
				"method": newQuery.GetMethod().String(),
			}).Debug("Mapped resource to query")
		}
	}

	return &results, nil
}
