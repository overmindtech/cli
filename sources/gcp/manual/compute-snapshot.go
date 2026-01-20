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

var ComputeSnapshotLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeSnapshot)

type computeSnapshotWrapper struct {
	client gcpshared.ComputeSnapshotsClient
	*gcpshared.ProjectBase
}

// NewComputeSnapshot creates a new computeSnapshotWrapper instance.
func NewComputeSnapshot(client gcpshared.ComputeSnapshotsClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeSnapshotWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			gcpshared.ComputeSnapshot,
		),
	}
}

func (c computeSnapshotWrapper) IAMPermissions() []string {
	return []string{
		"compute.snapshots.get",
		"compute.snapshots.list",
	}
}

func (c computeSnapshotWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeSnapshotWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeInstantSnapshot,
		gcpshared.ComputeLicense,
		gcpshared.ComputeDisk,
		gcpshared.CloudKMSCryptoKeyVersion,
		gcpshared.ComputeResourcePolicy,
	)
}

func (c computeSnapshotWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_snapshot.name",
		},
	}
}

func (c computeSnapshotWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeSnapshotLookupByName,
	}
}

func (c computeSnapshotWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetSnapshotRequest{
		Project:  location.ProjectID,
		Snapshot: queryParts[0],
	}

	snapshot, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeSnapshotToSDPItem(ctx, snapshot, location)
}

func (c computeSnapshotWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListSnapshotsRequest{
		Project: location.ProjectID,
	})

	var items []*sdp.Item
	for {
		snapshot, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeSnapshotToSDPItem(ctx, snapshot, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeSnapshotWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListSnapshotsRequest{
		Project: location.ProjectID,
	})

	for {
		snapshot, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeSnapshotToSDPItem(ctx, snapshot, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeSnapshotWrapper) gcpComputeSnapshotToSDPItem(ctx context.Context, snapshot *computepb.Snapshot, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(snapshot, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeSnapshot.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            snapshot.GetLabels(),
	}

	// Link to licenses
	for _, license := range snapshot.GetLicenses() {
		licenseName := gcpshared.LastPathComponent(license)
		if licenseName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeLicense.String(),
					Method: sdp.QueryMethod_GET,
					Query:  licenseName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to source instant snapshot
	if sourceInstantSnapshot := snapshot.GetSourceInstantSnapshot(); sourceInstantSnapshot != "" {
		instantSnapshotName := gcpshared.LastPathComponent(sourceInstantSnapshot)
		if instantSnapshotName != "" {
			scope, err := gcpshared.ExtractScopeFromURI(ctx, sourceInstantSnapshot)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeInstantSnapshot.String(),
						Method: sdp.QueryMethod_GET,
						Query:  instantSnapshotName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}

			if sourceInstantSnapshotEncryptionKey := snapshot.GetSourceInstantSnapshotEncryptionKey(); sourceInstantSnapshotEncryptionKey != nil {
				c.addKMSKeyLink(sdpItem, sourceInstantSnapshotEncryptionKey.GetKmsKeyName(), location)
			}
		}
	}

	// Link to source disk
	if disk := snapshot.GetSourceDisk(); disk != "" {
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

		if sourceDiskEncryptionKey := snapshot.GetSourceDiskEncryptionKey(); sourceDiskEncryptionKey != nil {
			c.addKMSKeyLink(sdpItem, sourceDiskEncryptionKey.GetKmsKeyName(), location)
		}
	}

	// Link to snapshot schedule policy
	if sourceSnapshotSchedulePolicy := snapshot.GetSourceSnapshotSchedulePolicy(); sourceSnapshotSchedulePolicy != "" {
		snapshotSchedulePolicyName := gcpshared.LastPathComponent(sourceSnapshotSchedulePolicy)
		if snapshotSchedulePolicyName != "" {
			scope, err := gcpshared.ExtractScopeFromURI(ctx, sourceSnapshotSchedulePolicy)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeResourcePolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  snapshotSchedulePolicyName,
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

	// Link to snapshot encryption key
	if snapshotEncryptionKey := snapshot.GetSnapshotEncryptionKey(); snapshotEncryptionKey != nil {
		c.addKMSKeyLink(sdpItem, snapshotEncryptionKey.GetKmsKeyName(), location)
	}

	switch snapshot.GetStatus() {
	case computepb.Snapshot_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.Snapshot_CREATING.String(),
		computepb.Snapshot_DELETING.String(),
		computepb.Snapshot_UPLOADING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Snapshot_FAILED.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.Snapshot_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	default:
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	}

	return sdpItem, nil
}

func (c computeSnapshotWrapper) addKMSKeyLink(sdpItem *sdp.Item, keyName string, location gcpshared.LocationInfo) {
	if keyName == "" {
		return
	}
	loc := gcpshared.ExtractPathParam("locations", keyName)
	keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
	cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
	cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

	if loc != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
				Method: sdp.QueryMethod_GET,
				Query:  shared.CompositeLookupKey(loc, keyRing, cryptoKey, cryptoKeyVersion),
				Scope:  location.ProjectID,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		})
	}
}
