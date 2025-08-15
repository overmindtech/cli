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

// NewComputeReservation creates a new computeReservationWrapper
func NewComputeReservation(client gcpshared.ComputeReservationClient, projectID, zone string) sources.ListableWrapper {
	return &computeReservationWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			zone,
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

// PotentialLinks returns the potential links for the compute reservation wrapper
func (c computeReservationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeRegionCommitment,
		gcpshared.ComputeMachineType,
		gcpshared.ComputeAcceleratorType,
		gcpshared.ComputeResourcePolicy,
	)
}

// TerraformMappings returns the Terraform mappings for the compute reservation wrapper
func (c computeReservationWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_node_group#argument-reference
			TerraformQueryMap: "google_compute_reservation.name",
		},
	}
}

// GetLookups returns the lookups for the compute reservation wrapper
func (c computeReservationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeReservationLookupByName,
	}
}

// Get retrieves a compute reservation by its name
func (c computeReservationWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetReservationRequest{
		Project:     c.ProjectID(),
		Zone:        c.Zone(),
		Reservation: queryParts[0],
	}

	reservation, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	item, sdpErr := c.gcpComputeReservationToSDPItem(reservation)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute reservations and converts them to sdp.Items.
func (c computeReservationWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListReservationsRequest{
		Project: c.ProjectID(), Zone: c.Zone(),
	})

	var items []*sdp.Item
	for {
		reservation, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		item, sdpErr := c.gcpComputeReservationToSDPItem(reservation)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream lists compute reservations and sends them as items to the stream.
func (c computeReservationWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListReservationsRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	for {
		reservation, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeReservationToSDPItem(reservation)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// gcpComputeReservationToSDPItem converts a GCP Reservation to an SDP Item
func (c computeReservationWrapper) gcpComputeReservationToSDPItem(reservation *computepb.Reservation) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           c.Scopes()[0],
	}

	// Reservations and commitments are linked; reservations can have multiple commitments and
	// commitments do not need to be linked to reservations.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/commitments/{commitment}
	// https://cloud.google.com/compute/docs/reference/rest/v1/regionCommitments/get
	if commitmentURL := reservation.GetCommitment(); commitmentURL != "" {
		commitmentName := gcpshared.LastPathComponent(commitmentURL)
		if commitmentName != "" {
			region := gcpshared.ExtractPathParam("regions", commitmentURL)
			if region != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeRegionCommitment.String(),
						Method: sdp.QueryMethod_GET,
						Query:  commitmentName,
						Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
					},
					// Deleting a reservation does not affect the commitment.
					// But deleting a commitment may affect the reservation and cause unexpected cost increases if you were relying on the discount.
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Reservations are used to guarantee availability of resources in a specific zone,
	// and are defined in terms of machine types and other instance properties.
	// MachineTypes describe the hardware configuration (vCPUs, memory) for VM instances.
	// A reservation must specify a MachineType or equivalent custom configuration.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/machineTypes/{machineType}
	// https://cloud.google.com/compute/docs/reference/rest/v1/machineTypes/get
	if reservation.GetSpecificReservation() != nil && reservation.GetSpecificReservation().GetInstanceProperties() != nil {
		if machineType := reservation.GetSpecificReservation().GetInstanceProperties().GetMachineType(); machineType != "" {
			machineTypeName := gcpshared.LastPathComponent(machineType)
			if machineTypeName != "" {
				zone := gcpshared.ExtractPathParam("zones", machineType)
				if zone != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeMachineType.String(),
							Method: sdp.QueryMethod_GET,
							Query:  machineTypeName,
							Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
						},
						//Not too sure about this one; while deleting a machine type doesn't necessarily delete the reservation,
						// it seems like deprecration of a machine type may make existing reservations unusable. Will set In: true for now to be sure.
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	// Accelerators are a type of resource the reservation optionally targets.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/acceleratorTypes/{acceleratorType}
	// https://cloud.google.com/compute/docs/reference/rest/v1/acceleratorTypes/get
	if reservation.GetSpecificReservation() != nil && reservation.GetSpecificReservation().GetInstanceProperties() != nil && len(reservation.GetSpecificReservation().GetInstanceProperties().GetGuestAccelerators()) != 0 {
		for _, accelerator := range reservation.GetSpecificReservation().GetInstanceProperties().GetGuestAccelerators() {
			if accelerator != nil && accelerator.GetAcceleratorType() != "" {
				acceleratorType := accelerator.GetAcceleratorType()
				acceleratorName := gcpshared.LastPathComponent(acceleratorType)
				if acceleratorName != "" {
					zone := gcpshared.ExtractPathParam("zones", acceleratorType)
					if zone != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeAcceleratorType.String(),
								Method: sdp.QueryMethod_GET,
								Query:  acceleratorName,
								Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
							},
							//Not too sure about this one; while deleting an accelerator type doesn't necessarily delete the reservation,
							// it seems like deprecration of an accelerator type may make existing reservations unusable. Will set In: true for now to be sure.
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

	// ResourcePolicies define additional behavior for resources, such as schedules or maintenance windows.
	// Reservations can reference resource policies to control instance creation timing or other lifecycle rules.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/resourcePolicies/{resourcePolicy}
	// https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies/get
	if reservation.GetResourcePolicies() != nil {
		for _, policyURL := range reservation.GetResourcePolicies() {
			if policyURL != "" {
				policyName := gcpshared.LastPathComponent(policyURL)
				if policyName != "" {
					region := gcpshared.ExtractPathParam("regions", policyURL)
					if region != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeResourcePolicy.String(),
								Method: sdp.QueryMethod_GET,
								Query:  policyName,
								Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
							},
							//Not too sure about this one; while deleting an policies type doesn't necessarily delete the reservation,
							// it seems like deleting a policy may cause errors.
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

	// The status of the reservation.
	//	For more information about the status of the reservation, see Reservation life cycle.
	// Check the Status enum for the list of possible values.
	//https://cloud.google.com/compute/docs/reference/rest/v1/reservations#:~:text=from%20this%20reservation.-,status,-enum
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
