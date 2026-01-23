package manual

import (
	"context"
	"errors"

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

var ComputeReservationLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeReservation)

type computeReservationWrapper struct {
	client gcpshared.ComputeReservationClient
	*gcpshared.ZoneBase
}

// NewComputeReservation creates a new computeReservationWrapper.
func NewComputeReservation(client gcpshared.ComputeReservationClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeReservationWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeReservation,
		),
	}
}

func (c computeReservationWrapper) IAMPermissions() []string {
	return []string{
		"compute.reservations.get",
		"compute.reservations.list",
	}
}

func (c computeReservationWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeReservationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeRegionCommitment,
		gcpshared.ComputeAcceleratorType,
		gcpshared.ComputeResourcePolicy,
	)
}

func (c computeReservationWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_reservation.name",
		},
	}
}

func (c computeReservationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeReservationLookupByName,
	}
}

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute reservations since they use aggregatedList
func (c computeReservationWrapper) SupportsWildcardScope() bool {
	return true
}

func (c computeReservationWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetReservationRequest{
		Project:     location.ProjectID,
		Zone:        location.Zone,
		Reservation: queryParts[0],
	}

	reservation, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeReservationToSDPItem(ctx, reservation, location)
}

func (c computeReservationWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeReservationWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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

	it := c.client.List(ctx, &computepb.ListReservationsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		reservation, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeReservationToSDPItem(ctx, reservation, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// listAggregatedStream uses AggregatedList to stream all reservations across all zones
func (c computeReservationWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := gcpshared.GetProjectIDsFromLocations(c.Locations())

	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListReservationsRequest{
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

				// Process reservations in this scope
				if pair.Value != nil && pair.Value.GetReservations() != nil {
					for _, reservation := range pair.Value.GetReservations() {
						item, sdpErr := c.gcpComputeReservationToSDPItem(ctx, reservation, scopeLocation)
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

func (c computeReservationWrapper) gcpComputeReservationToSDPItem(ctx context.Context, reservation *computepb.Reservation, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(reservation)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeReservation.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
	}

	// Link commitment
	if commitmentURL := reservation.GetCommitment(); commitmentURL != "" {
		commitmentName := gcpshared.LastPathComponent(commitmentURL)
		if commitmentName != "" {
			scope, err := gcpshared.ExtractScopeFromURI(ctx, commitmentURL)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeRegionCommitment.String(),
						Method: sdp.QueryMethod_GET,
						Query:  commitmentName,
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

	// Link accelerator types
	if reservation.GetSpecificReservation() != nil && reservation.GetSpecificReservation().GetInstanceProperties() != nil {
		for _, accelerator := range reservation.GetSpecificReservation().GetInstanceProperties().GetGuestAccelerators() {
			if accelerator != nil && accelerator.GetAcceleratorType() != "" {
				acceleratorType := accelerator.GetAcceleratorType()
				acceleratorName := gcpshared.LastPathComponent(acceleratorType)
				if acceleratorName != "" {
					scope, err := gcpshared.ExtractScopeFromURI(ctx, acceleratorType)
					if err == nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeAcceleratorType.String(),
								Method: sdp.QueryMethod_GET,
								Query:  acceleratorName,
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
		}
	}

	// Link resource policies
	for _, policyURL := range reservation.GetResourcePolicies() {
		if policyURL != "" {
			policyName := gcpshared.LastPathComponent(policyURL)
			if policyName != "" {
				scope, err := gcpshared.ExtractScopeFromURI(ctx, policyURL)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeResourcePolicy.String(),
							Method: sdp.QueryMethod_GET,
							Query:  policyName,
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
	}

	switch reservation.GetStatus() {
	case computepb.Reservation_CREATING.String(),
		computepb.Reservation_DELETING.String(),
		computepb.Reservation_UPDATING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Reservation_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	}

	return sdpItem, nil
}
