package manual

import (
	"context"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// InstanceGroupManagerToSDPItem converts a GCP InstanceGroupManager to an SDP Item.
// This function is shared between zonal and regional instance group manager adapters.
// The itemType parameter determines which Overmind type the SDP item will have.
func InstanceGroupManagerToSDPItem(ctx context.Context, instanceGroupManager *computepb.InstanceGroupManager, location gcpshared.LocationInfo, itemType shared.ItemType) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(instanceGroupManager, "")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            itemType.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
	}

	// Deleting the Instance Group Manager:
	// If the IGM is deleted, the associated instances are also deleted, but the instance template remains unaffected.
	// The instance template can still be used by other IGMs or for creating standalone instances.
	// Deleting an instance template also doesn't not delete the IGM.

	// Link instance template
	if instanceTemplate := instanceGroupManager.GetInstanceTemplate(); instanceTemplate != "" {
		instanceTemplateName := gcpshared.LastPathComponent(instanceTemplate)
		scope, err := gcpshared.ExtractScopeFromURI(ctx, instanceTemplate)
		if err == nil && instanceTemplateName != "" {
			templateType := gcpshared.ComputeInstanceTemplate
			if strings.Contains(instanceTemplate, "/regions/") {
				templateType = gcpshared.ComputeRegionInstanceTemplate
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   templateType.String(),
					Method: sdp.QueryMethod_GET,
					Query:  instanceTemplateName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
		}
	}

	// Link instance group
	if group := instanceGroupManager.GetInstanceGroup(); group != "" {
		instanceGroupName := gcpshared.LastPathComponent(group)
		scope, err := gcpshared.ExtractScopeFromURI(ctx, group)
		if err == nil && instanceGroupName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeInstanceGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  instanceGroupName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		}
	}

	// Link zone (for zonal instance group managers)
	if zone := instanceGroupManager.GetZone(); zone != "" {
		zoneName := gcpshared.LastPathComponent(zone)
		if zoneName != "" && location.ProjectID != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeZone.String(),
					Method: sdp.QueryMethod_GET,
					Query:  zoneName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
		}
	}

	// Link region (for regional instance group managers)
	if region := instanceGroupManager.GetRegion(); region != "" {
		regionName := gcpshared.LastPathComponent(region)
		if regionName != "" && location.ProjectID != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeRegion.String(),
					Method: sdp.QueryMethod_GET,
					Query:  regionName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
		}
	}

	// Link zones from distribution policy (for regional MIGs with explicit zone distribution)
	if distributionPolicy := instanceGroupManager.GetDistributionPolicy(); distributionPolicy != nil {
		for _, zoneConfig := range distributionPolicy.GetZones() {
			if zoneURL := zoneConfig.GetZone(); zoneURL != "" {
				zoneName := gcpshared.LastPathComponent(zoneURL)
				if zoneName != "" && location.ProjectID != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeZone.String(),
							Method: sdp.QueryMethod_GET,
							Query:  zoneName,
							Scope:  location.ProjectID,
						},
						BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
					})
				}
			}
		}
	}

	// Link target pools
	for _, targetPool := range instanceGroupManager.GetTargetPools() {
		targetPoolName := gcpshared.LastPathComponent(targetPool)
		scope, err := gcpshared.ExtractScopeFromURI(ctx, targetPool)
		if err == nil && targetPoolName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeTargetPool.String(),
					Method: sdp.QueryMethod_GET,
					Query:  targetPoolName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		}
	}

	// Link resource policies from ResourcePolicies.WorkloadPolicy
	if resourcePolicies := instanceGroupManager.GetResourcePolicies(); resourcePolicies != nil {
		if workloadPolicy := resourcePolicies.GetWorkloadPolicy(); workloadPolicy != "" {
			resourcePolicyName := gcpshared.LastPathComponent(workloadPolicy)
			scope, err := gcpshared.ExtractScopeFromURI(ctx, workloadPolicy)
			if err == nil && resourcePolicyName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeResourcePolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  resourcePolicyName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// Link to instance templates in versions array (used for canary/rolling deployments)
	// If versions are defined, they override the top-level instanceTemplate
	// Each version can have its own template, so we need to link all of them
	for _, version := range instanceGroupManager.GetVersions() {
		if versionTemplate := version.GetInstanceTemplate(); versionTemplate != "" {
			versionTemplateName := gcpshared.LastPathComponent(versionTemplate)
			scope, err := gcpshared.ExtractScopeFromURI(ctx, versionTemplate)
			if err == nil && versionTemplateName != "" {
				templateType := gcpshared.ComputeInstanceTemplate
				if strings.Contains(versionTemplate, "/regions/") {
					templateType = gcpshared.ComputeRegionInstanceTemplate
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   templateType.String(),
						Method: sdp.QueryMethod_GET,
						Query:  versionTemplateName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// Link to health checks used in auto-healing policies
	// Auto-healing policies use health checks to determine if instances are healthy
	// If the health check is deleted or updated, auto-healing may fail
	for _, autoHealingPolicy := range instanceGroupManager.GetAutoHealingPolicies() {
		if healthCheckURL := autoHealingPolicy.GetHealthCheck(); healthCheckURL != "" {
			healthCheckName := gcpshared.LastPathComponent(healthCheckURL)
			scope, err := gcpshared.ExtractScopeFromURI(ctx, healthCheckURL)
			if err == nil && healthCheckName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeHealthCheck.String(),
						Method: sdp.QueryMethod_GET,
						Query:  healthCheckName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Autoscalers set the Instance Group Manager target size
	// InstanceGroupManagers orphans the autoscaler when deleted
	if status := instanceGroupManager.GetStatus(); status != nil {
		if autoscalerURL := status.GetAutoscaler(); autoscalerURL != "" {
			autoscalerName := gcpshared.LastPathComponent(autoscalerURL)
			scope, err := gcpshared.ExtractScopeFromURI(ctx, autoscalerURL)
			if err == nil && autoscalerName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeAutoscaler.String(),
						Method: sdp.QueryMethod_GET,
						Query:  autoscalerName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
				})
			}
		}
	}

	switch {
	case instanceGroupManager.GetStatus() != nil && instanceGroupManager.GetStatus().GetIsStable():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	default:
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	}

	return sdpItem, nil
}
