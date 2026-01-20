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

// NewComputeInstantSnapshot creates a new computeInstantSnapshotWrapper instance.
func NewComputeInstantSnapshot(client gcpshared.ComputeInstantSnapshotsClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeInstantSnapshotWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
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

func (c computeInstantSnapshotWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeDisk,
	)
}

func (c computeInstantSnapshotWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_instant_snapshot.name",
		},
	}
}

func (c computeInstantSnapshotWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstantSnapshotLookupByName,
	}
}

func (c computeInstantSnapshotWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetInstantSnapshotRequest{
		Project:         location.ProjectID,
		Zone:            location.Zone,
		InstantSnapshot: queryParts[0],
	}

	instantSnapshot, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeInstantSnapshotToSDPItem(ctx, instantSnapshot, location)
}

func (c computeInstantSnapshotWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListInstantSnapshotsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	var items []*sdp.Item
	for {
		instantSnapshot, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeInstantSnapshotToSDPItem(ctx, instantSnapshot, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeInstantSnapshotWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListInstantSnapshotsRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		instantSnapshot, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeInstantSnapshotToSDPItem(ctx, instantSnapshot, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeInstantSnapshotWrapper) gcpComputeInstantSnapshotToSDPItem(ctx context.Context, instantSnapshot *computepb.InstantSnapshot, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           location.ToScope(),
		Tags:            instantSnapshot.GetLabels(),
	}

	// Link source disk
	if disk := instantSnapshot.GetSourceDisk(); disk != "" {
		diskName := gcpshared.LastPathComponent(disk)
		if diskName != "" {
			scope, err := gcpshared.ExtractScopeFromURI(ctx, disk)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeDisk.String(),
						Method: sdp.QueryMethod_GET,
						Query:  diskName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
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
