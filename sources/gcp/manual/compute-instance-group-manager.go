package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/proto"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeInstanceGroupManagerLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeInstanceGroupManager)

type computeInstanceGroupManagerWrapper struct {
	client gcpshared.ComputeInstanceGroupManagerClient
	*gcpshared.ZoneBase
}

// NewComputeInstanceGroupManager creates a new computeInstanceGroupManagerWrapper.
func NewComputeInstanceGroupManager(client gcpshared.ComputeInstanceGroupManagerClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeInstanceGroupManagerWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
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

func (c computeInstanceGroupManagerWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
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
		gcpshared.ComputeHealthCheck,
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

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute instance group managers since they use aggregatedList
func (c computeInstanceGroupManagerWrapper) SupportsWildcardScope() bool {
	return true
}

// Get retrieves a compute instance group manager by its name
func (c computeInstanceGroupManagerWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetInstanceGroupManagerRequest{
		Project:              location.ProjectID,
		Zone:                 location.Zone,
		InstanceGroupManager: queryParts[0],
	}

	igm, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpInstanceGroupManagerToSDPItem(ctx, igm, location)
}

func (c computeInstanceGroupManagerWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeInstanceGroupManagerWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	// Handle wildcard scope with AggregatedList
	if scope == "*" {
		c.listAggregatedStream(ctx, stream, cache, cacheKey)
		return
	}

	// Handle specific scope with per-zone List
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListInstanceGroupManagersRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		igm, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpInstanceGroupManagerToSDPItem(ctx, igm, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// listAggregatedStream uses AggregatedList to stream all instance group managers across all zones
func (c computeInstanceGroupManagerWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := gcpshared.GetProjectIDsFromLocations(c.Locations())

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListInstanceGroupManagersRequest{
				Project:              projectID,
				ReturnPartialSuccess: proto.Bool(true), // Handle partial failures gracefully
			})

			for {
				pair, iterErr := it.Next()
				if errors.Is(iterErr, iterator.Done) {
					break
				}
				if iterErr != nil {
					stream.SendError(gcpshared.QueryError(iterErr, projectID, c.Type()))
					return iterErr
				}

				// Parse scope from pair.Key (e.g., "zones/us-central1-a")
				scopeLocation, err := gcpshared.ParseAggregatedListScope(projectID, pair.Key)
				if err != nil {
					continue // Skip unparseable scopes
				}

				// Only process if this scope is in our adapter's configured locations
				if !gcpshared.HasLocationInSlices(scopeLocation, c.Locations()) {
					continue
				}

				// Process instance group managers in this scope
				if pair.Value != nil && pair.Value.GetInstanceGroupManagers() != nil {
					for _, igm := range pair.Value.GetInstanceGroupManagers() {
						item, sdpErr := c.gcpInstanceGroupManagerToSDPItem(ctx, igm, scopeLocation)
						if sdpErr != nil {
							stream.SendError(sdpErr)
							continue
						}

						cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
						stream.SendItem(item)
					}
				}
			}

			return nil
		})
	}

	// Wait for all goroutines to complete
	_ = p.Wait()
}

func (c computeInstanceGroupManagerWrapper) gcpInstanceGroupManagerToSDPItem(ctx context.Context, instanceGroupManager *computepb.InstanceGroupManager, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(instanceGroupManager, "")
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
