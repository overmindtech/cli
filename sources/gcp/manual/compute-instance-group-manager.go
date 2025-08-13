package manual

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeInstanceGroupManagerLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeInstanceGroupManager)
)

type computeInstanceGroupManagerWrapper struct {
	client gcpshared.ComputeInstanceGroupManagerClient

	*gcpshared.ZoneBase
}

// NewComputeInstanceGroupManager creates a new computeInstanceGroupManagerWrapper
func NewComputeInstanceGroupManager(client gcpshared.ComputeInstanceGroupManagerClient, projectID, zone string) sources.ListableWrapper {
	return &computeInstanceGroupManagerWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			zone,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeInstanceGroupManager,
		),
	}
}

func (c computeInstanceGroupManagerWrapper) IAMPermissions() []string {
	return []string{
		"compute.instanceGroupManagers.get",
		"compute.instanceGroupManagers.list",
	}
}

// PotentialLinks returns the potential links for the compute instance group manager wrapper
func (c computeInstanceGroupManagerWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeInstanceTemplate,
		gcpshared.ComputeRegionInstanceTemplate,
		gcpshared.ComputeInstanceGroup,
		gcpshared.ComputeTargetPool,
		gcpshared.ComputeResourcePolicy,
		gcpshared.ComputeAutoscaler,
	)
}

// TerraformMappings returns the Terraform mappings for the compute instance group manager wrapper
func (c computeInstanceGroupManagerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance_group_manager#argument-reference
			TerraformQueryMap: "google_compute_instance_group_manager.name",
		},
	}
}

// GetLookups returns the lookups for the compute instance group manager wrapper
func (c computeInstanceGroupManagerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstanceGroupManagerLookupByName,
	}
}

// Get retrieves a compute instance group manager by its name
func (c computeInstanceGroupManagerWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetInstanceGroupManagerRequest{
		Project:              c.ProjectID(),
		Zone:                 c.Zone(),
		InstanceGroupManager: queryParts[0],
	}

	instanceGroupManager, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpInstanceGroupManagerToSDPItem(instanceGroupManager)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute instance group managers and converts them to sdp.Items.
func (c computeInstanceGroupManagerWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListInstanceGroupManagersRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	var items []*sdp.Item
	for {
		instanceGroupManager, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpInstanceGroupManagerToSDPItem(instanceGroupManager)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeInstanceGroupManagerWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream) {
	it := c.client.List(ctx, &computepb.ListInstanceGroupManagersRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	for {
		instanceGroupManager, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err))
			return
		}

		item, sdpErr := c.gcpInstanceGroupManagerToSDPItem(instanceGroupManager)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		stream.SendItem(item)
	}
}

func (c computeInstanceGroupManagerWrapper) gcpInstanceGroupManagerToSDPItem(instanceGroupManager *computepb.InstanceGroupManager) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(instanceGroupManager)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeInstanceGroupManager.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
	}

	//Deleting the Instance Group Manager:
	//If the IGM is deleted, the associated instances are also deleted, but the instance template remains unaffected.
	//The instance template can still be used by other IGMs or for creating standalone instances.
	//Deleting an instance template also doesn't not delete the IGM.
	if instanceTemplate := instanceGroupManager.GetInstanceTemplate(); instanceTemplate != "" {
		instanceTemplateName := gcpshared.LastPathComponent(instanceTemplate)
		region := gcpshared.ExtractPathParam("regions", instanceTemplate)
		//Set type as ComputeRegionInstanceTemplate if this is a regional template
		if region != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeRegionInstanceTemplate.String(),
					Method: sdp.QueryMethod_GET,
					Query:  instanceTemplateName,
					Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
			//Set type as ComputeInstanceTemplate if this is a global template
		} else {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeInstanceTemplate.String(),
					Method: sdp.QueryMethod_GET,
					Query:  instanceTemplateName,
					Scope:  c.ProjectID(),
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
		}
	}

	if group := instanceGroupManager.GetInstanceGroup(); group != "" {
		instanceGroupName := gcpshared.LastPathComponent(group)
		zone := gcpshared.ExtractPathParam("zones", group)
		if zone != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeInstanceGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  instanceGroupName,
					Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		}
	}

	for _, targetPool := range instanceGroupManager.GetTargetPools() {
		targetPoolName := gcpshared.LastPathComponent(targetPool)
		region := gcpshared.ExtractPathParam("regions", targetPool)
		if region != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeTargetPool.String(),
					Method: sdp.QueryMethod_GET,
					Query:  targetPoolName,
					Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
			})
		}
	}

	if instanceGroupManager.GetResourcePolicies() != nil {
		resourcePolicy := instanceGroupManager.GetResourcePolicies().GetWorkloadPolicy()
		//Deleting the  Instance Group Manager does not affect the the Resource Policy.
		//Deleting the Resource Policy doesn't stop the Instance Group Manager from running but makes it lose the policy’s scheduled effects.
		if resourcePolicy != "" {
			resourcePolicyName := gcpshared.LastPathComponent(string(resourcePolicy))
			if resourcePolicyName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeResourcePolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  resourcePolicyName,
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// Autoscalers set the Instance Group Manager target size
	// InstanceGroupManagers orphans the autoscaler when deleted
	if status := instanceGroupManager.GetStatus(); status != nil {
		if autoscalerURL := status.GetAutoscaler(); autoscalerURL != "" {
			autoscalerName := gcpshared.LastPathComponent(autoscalerURL)
			zone := gcpshared.ExtractPathParam("zones", autoscalerURL)
			if autoscalerName != "" && zone != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeAutoscaler.String(),
						Method: sdp.QueryMethod_GET,
						Query:  autoscalerName,
						Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
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
