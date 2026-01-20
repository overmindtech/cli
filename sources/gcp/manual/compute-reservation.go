package manual

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

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
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListReservationsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	var items []*sdp.Item
	for {
		reservation, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeReservationToSDPItem(ctx, reservation, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeReservationWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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
