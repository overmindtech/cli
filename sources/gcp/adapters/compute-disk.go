package adapters

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeDisk     = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.Disk)
	ComputeDiskType = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.DiskType)

	ComputeDiskLookupByName = shared.NewItemTypeLookup("name", ComputeDisk)
)

type computeDiskWrapper struct {
	client gcpshared.ComputeDiskClient

	*gcpshared.ZoneBase
}

// NewComputeDisk creates a new computeDiskWrapper
func NewComputeDisk(client gcpshared.ComputeDiskClient, projectID, zone string) sources.ListableWrapper {
	return &computeDiskWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			zone,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			ComputeDisk,
		),
	}
}

// PotentialLinks returns the potential links for the compute instance wrapper
func (c computeDiskWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		ComputeResourcePolicy,
		ComputeDisk,
		ComputeImage,
		ComputeSnapshot,
		ComputeInstantSnapshot,
		ComputeDiskType,
		ComputeInstance,
		CloudKMSCryptoKeyVersion,
	)
}

// TerraformMappings returns the Terraform mappings for the compute disk wrapper
func (c computeDiskWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_disk.name",
		},
	}
}

// GetLookups returns the lookups for the compute disk wrapper
func (c computeDiskWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeDiskLookupByName,
	}
}

// Get retrieves a compute disk by its name
func (c computeDiskWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetDiskRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
		Disk:    queryParts[0],
	}

	disk, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeDiskToSDPItem(disk)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute disks and converts them to sdp.Items.
func (c computeDiskWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListDisksRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	var items []*sdp.Item
	for {
		disk, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeDiskToSDPItem(disk)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// gcpComputeDiskToSDPItem converts a GCP Disk to an SDP Item, linking GCP resource fields.
func (c computeDiskWrapper) gcpComputeDiskToSDPItem(disk *computepb.Disk) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(disk, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            ComputeDisk.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            disk.GetLabels(),
	}

	// The resource URL for the disk type associated with this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/diskTypes/{diskType}
	// https://cloud.google.com/compute/docs/reference/rest/v1/diskTypes/get
	if diskType := disk.GetType(); diskType != "" {
		if strings.Contains(diskType, "/") {
			diskTypeName := gcpshared.LastPathComponent(diskType)
			if diskTypeName != "" {
				zone := gcpshared.ExtractPathParam("zones", diskType)
				if zone != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   ComputeDiskType.String(),
							Method: sdp.QueryMethod_GET,
							Query:  diskTypeName,
							Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
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

	// The resource URL for the image used to create this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/images/{image}
	// https://cloud.google.com/compute/docs/reference/rest/v1/images/get
	if sourceImage := disk.GetSourceImage(); sourceImage != "" {
		if strings.Contains(sourceImage, "/") {
			imageName := gcpshared.LastPathComponent(sourceImage)
			if imageName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   ComputeImage.String(),
						Method: sdp.QueryMethod_GET,
						Query:  imageName,
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// The resource URL for the snapshot used to create this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/snapshots/{snapshot}
	// https://cloud.google.com/compute/docs/reference/rest/v1/snapshots/get
	if sourceSnapshot := disk.GetSourceSnapshot(); sourceSnapshot != "" {
		if strings.Contains(sourceSnapshot, "/") {
			snapshotName := gcpshared.LastPathComponent(sourceSnapshot)
			if snapshotName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   ComputeSnapshot.String(),
						Method: sdp.QueryMethod_GET,
						Query:  snapshotName,
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}

		}
	}

	// The resource URL for the instant snapshot used to create this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instantSnapshots/{instantSnapshot}
	// https://cloud.google.com/compute/docs/reference/rest/v1/instantSnapshots/get
	if sourceInstantSnapshot := disk.GetSourceInstantSnapshot(); sourceInstantSnapshot != "" {
		if strings.Contains(sourceInstantSnapshot, "/") {
			instantSnapshotName := gcpshared.LastPathComponent(sourceInstantSnapshot)
			if instantSnapshotName != "" {
				zone := gcpshared.ExtractPathParam("zones", sourceInstantSnapshot)
				if zone != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   ComputeInstantSnapshot.String(),
							Method: sdp.QueryMethod_GET,
							Query:  instantSnapshotName,
							Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
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

	// The resource URL for the source disk used to create this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/disks/{disk}
	// https://cloud.google.com/compute/docs/reference/rest/v1/disks/get
	if sourceDisk := disk.GetSourceDisk(); sourceDisk != "" {
		if strings.Contains(sourceDisk, "/") {
			sourceDiskName := gcpshared.LastPathComponent(sourceDisk)
			if sourceDiskName != "" {
				zone := gcpshared.ExtractPathParam("zones", sourceDisk)
				if zone != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   ComputeDisk.String(),
							Method: sdp.QueryMethod_GET,
							Query:  sourceDiskName,
							Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
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

	// The resource URLs for the resource policies associated with this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/resourcePolicies/{resourcePolicy}
	// https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies/get
	for _, rp := range disk.GetResourcePolicies() {
		if strings.Contains(rp, "/") {
			rpName := gcpshared.LastPathComponent(rp)
			if rpName != "" {
				region := gcpshared.ExtractPathParam("regions", rp)
				if region != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   ComputeResourcePolicy.String(),
							Method: sdp.QueryMethod_GET,
							Query:  rpName,
							Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
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
	// The resource URLs for the users (instances) using this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{instance}
	// https://cloud.google.com/compute/docs/reference/rest/v1/instances/get
	for _, instance := range disk.GetUsers() {
		if strings.Contains(instance, "/") {
			instanceName := gcpshared.LastPathComponent(instance)
			if instanceName != "" {
				zone := gcpshared.ExtractPathParam("zones", instance)
				if zone != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   ComputeInstance.String(),
							Method: sdp.QueryMethod_GET,
							Query:  instanceName,
							Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  false,
							Out: true,
						},
					})
				}
			}
		}
	}

	// The Encryption keys associated with this disk; appears in the following format:
	// "diskEncryptionKey.kmsKeyName": "projects/kms_project_id/locations/region/keyRings/key_region/cryptoKeys/key/cryptoKeysVersions/version
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
	// DiskEncryptionKey.kmsKeyName -> CloudKMSCryptoKeyVersion
	if diskEncryptionKey := disk.GetDiskEncryptionKey(); diskEncryptionKey != nil {
		if keyName := diskEncryptionKey.GetKmsKeyName(); keyName != "" {

			// Parsing them all together to improve readability
			location := gcpshared.ExtractPathParam("locations", keyName)
			keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
			cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
			cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

			// Validate all parts before proceeding, a bit less performatic if any is missing but readability is improved
			if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudKMSCryptoKeyVersion.String(),
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

	// The customer-supplied encryption key of the source image; appears in the following format:
	// "sourceImageEncryptionKey.kmsKeyName": ""projects/ kms_project_id/locations/ region/keyRings/ key_region/cryptoKeys/key /cryptoKeyVersions/1"
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
	// SourceImageEncryptionKey.kmsKeyName -> CloudKMSCryptoKeyVersion
	if sourceImageEncryptionKey := disk.GetSourceImageEncryptionKey(); sourceImageEncryptionKey != nil {
		if keyName := sourceImageEncryptionKey.GetKmsKeyName(); keyName != "" {

			// Parsing them all together to improve readability
			location := gcpshared.ExtractPathParam("locations", keyName)
			keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
			cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
			cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

			// Validate all parts before proceeding, a bit less performatic if any is missing but readability is improved
			if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
						Scope:  c.ProjectID(),
					},
					//Deleting a key might break the disk’s ability to function and have its data read
					//Deleting a disk in GCP does not affect its source image's encryption key
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// The customer-supplied encryption key of the source snapshot; appears in the following format:
	// "sourceImageEncryptionKey.kmsKeyName": "projects/ kms_project_id/locations/ region/keyRings/ key_region/cryptoKeys/key /cryptoKeyVersions/1"
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
	// SourceSnapshotEncryptionKey.kmsKeyName -> CloudKMSCryptoKeyVersion
	if sourceSnapshotEncryptionKey := disk.GetSourceSnapshotEncryptionKey(); sourceSnapshotEncryptionKey != nil {
		if keyName := sourceSnapshotEncryptionKey.GetKmsKeyName(); keyName != "" {

			// Parsing them all together to improve readability
			location := gcpshared.ExtractPathParam("locations", keyName)
			keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
			cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
			cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

			// Validate all parts before proceeding, a bit less performatic if any is missing but readability is improved
			if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
						Scope:  c.ProjectID(),
					},
					//Deleting a key might break the disk’s ability to function and have its data read
					//Deleting a disk in GCP does not affect its source image's encryption key
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// The URL of the DiskConsistencyGroupPolicy for a secondary disk that was created using a consistency group; this is a type of Resource Policy.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/resourcePolicies/{resourcePolicy}
	// https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies
	if sourceConsistencyGroupPolicy := disk.GetSourceConsistencyGroupPolicy(); sourceConsistencyGroupPolicy != "" {
		if strings.Contains(sourceConsistencyGroupPolicy, "/") {
			rpName := gcpshared.LastPathComponent(sourceConsistencyGroupPolicy)
			if rpName != "" {
				region := gcpshared.ExtractPathParam("regions", sourceConsistencyGroupPolicy)
				if region != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   ComputeResourcePolicy.String(),
							Method: sdp.QueryMethod_GET,
							Query:  rpName,
							Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
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

	// The following seem like possible links: SourceStorageObject. Nauany hasn't identified a way to get this from any of the APIs.

	switch disk.GetStatus() {
	case computepb.Disk_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.Disk_CREATING.String(),
		computepb.Disk_RESTORING.String(),
		computepb.Disk_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Disk_FAILED.String(),
		computepb.Disk_UNAVAILABLE.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.Disk_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	}

	return sdpItem, nil
}
