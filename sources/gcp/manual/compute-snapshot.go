package manual

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeSnapshotLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeSnapshot)

type computeSnapshotWrapper struct {
	client gcpshared.ComputeSnapshotsClient

	*gcpshared.ProjectBase
}

// NewComputeSnapshot creates a new computeSnapshotWrapper instance
func NewComputeSnapshot(client gcpshared.ComputeSnapshotsClient, projectID string) sources.ListableWrapper {
	return &computeSnapshotWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
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

// PotentialLinks returns the potential links for the compute snapshot wrapper
func (c computeSnapshotWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeInstantSnapshot,
		gcpshared.ComputeLicense,
		gcpshared.ComputeDisk,
		gcpshared.CloudKMSCryptoKeyVersion,
		gcpshared.ComputeResourcePolicy,
	)
}

// TerraformMappings returns the Terraform mappings for the compute snapshot wrapper
func (c computeSnapshotWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_snapshot#argument-reference
			TerraformQueryMap: "google_compute_snapshot.name",
		},
	}
}

// GetLookups returns the lookups for the compute snapshot wrapper
func (c computeSnapshotWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeSnapshotLookupByName,
	}
}

// Get retrieves a compute snapshot by its name
func (c computeSnapshotWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetSnapshotRequest{
		Project:  c.ProjectID(),
		Snapshot: queryParts[0],
	}

	snapshot, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeSnapshotToSDPItem(snapshot)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute snapshots and converts them to sdp.Items.
func (c computeSnapshotWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListSnapshotsRequest{
		Project: c.ProjectID(),
	})

	var items []*sdp.Item
	for {
		snapshot, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeSnapshotToSDPItem(snapshot)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// gcpComputeSnapshotToSDPItem converts a GCP Snapshot to an SDP Item, linking GCP resource fields.
func (c computeSnapshotWrapper) gcpComputeSnapshotToSDPItem(snapshot *computepb.Snapshot) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           c.DefaultScope(),
		Tags:            snapshot.GetLabels(),
	}

	// The resource URLs for the licenses associated with this snapshot.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/licenses/{license}
	// https://cloud.google.com/compute/docs/reference/rest/v1/licenses/get
	// Caution This resource is intended for use only by third-party partners who are creating Cloud Marketplace images.
	for _, license := range snapshot.GetLicenses() {
		licenseName := gcpshared.LastPathComponent(license)
		if licenseName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeLicense.String(),
					Method: sdp.QueryMethod_GET,
					Query:  licenseName,
					Scope:  c.ProjectID(),
				},
				// While most licenses are created and managed by GCP, the license can be created by the user https://cloud.google.com/compute/docs/reference/rest/v1/licenses/insert.
				// If the license used to create the snapshot is a custom license created by the user then deleting it would leave the snapshot in an inconsistent state.
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
		}
	}

	// The resource URL for the source instant snapshot of this snapshot.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instantSnapshots/{instantSnapshot}
	// https://cloud.google.com/compute/docs/reference/rest/v1/instantSnapshots/get
	if sourceInstantSnapshot := snapshot.GetSourceInstantSnapshot(); sourceInstantSnapshot != "" {
		instantSnapshotName := gcpshared.LastPathComponent(sourceInstantSnapshot)
		if instantSnapshotName != "" {
			zone := gcpshared.ExtractPathParam("zones", sourceInstantSnapshot)
			if zone != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeInstantSnapshot.String(),
						Method: sdp.QueryMethod_GET,
						Query:  instantSnapshotName,
						Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}

		}

		// The customer provided encryption key when creating Snapshot from Instant Snapshot; appears in the following format:
		// "sourceInstantSnapshotEncryptionKey.kmsKeyName": "projects/ kms_project_id/locations/ region/keyRings/ key_region/cryptoKeys/key /cryptoKeyVersions/1"
		// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
		// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
		// sourceInstantSnapshotEncryptionKey.kmsKeyName -> CloudKMSCryptoKeyVersion
		if sourceInstantSnapshotEncryptionKey := snapshot.GetSourceInstantSnapshotEncryptionKey(); sourceInstantSnapshotEncryptionKey != nil {
			if keyName := sourceInstantSnapshotEncryptionKey.GetKmsKeyName(); keyName != "" {
				// Parsing them all together to improve readability
				location := gcpshared.ExtractPathParam("locations", keyName)
				keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
				cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
				cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

				// Validate all parts before proceeding, a bit less performatic if any is missing but readability is improved
				if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
							Scope:  c.ProjectID(),
						},
						//If the key is deleted the snapshot cannot be decrypted or used
						//Deleting the snapshot does not affect the key
						BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
					})
				}
			}
		}
	}

	// The resource URL for the source disk of this snapshot.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/disks/{disk}
	// https://cloud.google.com/compute/docs/reference/rest/v1/disks/get
	// The source disk is the disk from which this snapshot was created. Deleting the disk does not impact the snapshot,
	// but the snapshot cannot be restored to the point where it was taken if the snapshot is deleted.
	if disk := snapshot.GetSourceDisk(); disk != "" {
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

		// The customer-supplied encryption key of the source disk; appears in the following format:
		// "sourceDiskEncryptionKey.kmsKeyName": "projects/ kms_project_id/locations/ region/keyRings/ key_region/cryptoKeys/key /cryptoKeyVersions/1
		// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
		// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
		// sourceDiskEncryptionKey.kmsKeyName -> CloudKMSCryptoKeyVersion
		if sourceDiskEncryptionKey := snapshot.GetSourceDiskEncryptionKey(); sourceDiskEncryptionKey != nil {
			if keyName := sourceDiskEncryptionKey.GetKmsKeyName(); keyName != "" {

				// Parsing them all together to improve readability
				location := gcpshared.ExtractPathParam("locations", keyName)
				keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
				cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
				cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

				// Validate all parts before proceeding, a bit less performatic if any is missing but readability is improved
				if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
							Scope:  c.ProjectID(),
						},
						//Deleting a key might break the disk’s ability to function and have its data read
						//Deleting a disk in GCP does not affect its associated encryption key
						BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
					})
				}
			}
		}
	}

	// The URL of the resource policy which created this scheduled snapshot; this is a type of Resource Policy.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/resourcePolicies/{resourcePolicy}
	// https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies
	if sourceSnapshotSchedulePolicy := snapshot.GetSourceSnapshotSchedulePolicy(); sourceSnapshotSchedulePolicy != "" {
		snapshotSchedulePolicyName := gcpshared.LastPathComponent(sourceSnapshotSchedulePolicy)
		if snapshotSchedulePolicyName != "" {
			region := gcpshared.ExtractPathParam("regions", sourceSnapshotSchedulePolicy)
			if region != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeResourcePolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  snapshotSchedulePolicyName,
						Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
					},
					// Existing snapshot remains available even if the source policy is deleted.
					// However, new snapshots will not be created automatically unless the policy is recreated or replaced.
					//If snapshot is deleted the policy remains unaffected.
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
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
