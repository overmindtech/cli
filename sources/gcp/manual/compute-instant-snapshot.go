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

var ComputeInstantSnapshotLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeInstantSnapshot)

type computeInstantSnapshotWrapper struct {
	client gcpshared.ComputeInstantSnapshotsClient

	*gcpshared.ZoneBase
}

// NewComputeInstantSnapshot creates a new computeInstantSnapshotWrapper instance
func NewComputeInstantSnapshot(client gcpshared.ComputeInstantSnapshotsClient, projectID, zone string) sources.ListableWrapper {
	return &computeInstantSnapshotWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			zone,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			gcpshared.ComputeInstantSnapshot,
		),
	}
}

func (c computeInstantSnapshotWrapper) IAMPermissions() []string {
	return []string{
		"compute.instantSnapshots.get",
		"compute.instantSnapshots.list",
	}
}

func (c computeInstantSnapshotWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

// PotentialLinks returns the potential links for the compute snapshot wrapper
func (c computeInstantSnapshotWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeDisk,
	)
}

// TerraformMappings returns the Terraform mappings for the compute instant snapshot wrapper
func (c computeInstantSnapshotWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance_group_manager#argument-reference
			TerraformQueryMap: "google_compute_instant_snapshot.name",
		},
	}
}

// GetLookups returns the lookups for the compute instant snapshot wrapper
func (c computeInstantSnapshotWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstantSnapshotLookupByName,
	}
}

// Get retrieves a compute instant snapshot by its name
func (c computeInstantSnapshotWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetInstantSnapshotRequest{
		Project:         c.ProjectID(),
		Zone:            c.Zone(),
		InstantSnapshot: queryParts[0],
	}

	instantSnapshot, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	item, sdpErr := c.gcpComputeInstantSnapshotToSDPItem(instantSnapshot)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute instant snapshots and converts them to sdp.Items.
func (c computeInstantSnapshotWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListInstantSnapshotsRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	var items []*sdp.Item
	for {
		instantSnapshot, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeInstantSnapshotToSDPItem(instantSnapshot)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream lists compute instant snapshots and sends them to the stream.
func (c computeInstantSnapshotWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListInstantSnapshotsRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	for {
		instantSnapshot, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeInstantSnapshotToSDPItem(instantSnapshot)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// gcpComputeInstantSnapshotToSDPItem converts a GCP Instant Snapshot to an SDP Item
func (c computeInstantSnapshotWrapper) gcpComputeInstantSnapshotToSDPItem(instantSnapshot *computepb.InstantSnapshot) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(instantSnapshot, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeInstantSnapshot.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            instantSnapshot.GetLabels(),
	}

	if disk := instantSnapshot.GetSourceDisk(); disk != "" {
		zone := gcpshared.ExtractPathParam("zones", disk)
		if zone != "" {
			diskName := gcpshared.LastPathComponent(disk)
			if diskName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeDisk.String(),
						Method: sdp.QueryMethod_GET,
						Query:  diskName,
						Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
					},
					//Disk cannot be restored to the point where the snapshot was taken if the snapshot is deleted.
					//Deleting disk does not impact the snapshot.
					BlastPropagation: &sdp.BlastPropagation{In: false, Out: true},
				})
			}
		}
	}

	switch instantSnapshot.GetStatus() {
	case computepb.InstantSnapshot_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.InstantSnapshot_CREATING.String(),
		computepb.InstantSnapshot_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.InstantSnapshot_FAILED.String(),
		computepb.InstantSnapshot_UNAVAILABLE.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.InstantSnapshot_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	default:
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	}

	return sdpItem, nil
}
