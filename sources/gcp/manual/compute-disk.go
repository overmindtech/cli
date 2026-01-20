package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeDiskLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeDisk)

type computeDiskWrapper struct {
	client gcpshared.ComputeDiskClient
	*gcpshared.ZoneBase
}

// NewComputeDisk creates a new computeDiskWrapper.
func NewComputeDisk(client gcpshared.ComputeDiskClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeDiskWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			gcpshared.ComputeDisk,
		),
	}
}

func (c computeDiskWrapper) IAMPermissions() []string {
	return []string{
		"compute.disks.get",
		"compute.disks.list",
	}
}

func (c computeDiskWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeDiskWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeResourcePolicy,
		gcpshared.ComputeDisk,
		gcpshared.ComputeImage,
		gcpshared.ComputeSnapshot,
		gcpshared.ComputeInstantSnapshot,
		gcpshared.ComputeDiskType,
		gcpshared.ComputeInstance,
		gcpshared.CloudKMSCryptoKeyVersion,
		gcpshared.StorageBucket,
		gcpshared.ComputeStoragePool,
	)
}

func (c computeDiskWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_disk.name",
		},
	}
}

func (c computeDiskWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeDiskLookupByName,
	}
}

func (c computeDiskWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetDiskRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
		Disk:    queryParts[0],
	}

	disk, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeDiskToSDPItem(ctx, disk, location)
}

func (c computeDiskWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListDisksRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	var items []*sdp.Item
	for {
		disk, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeDiskToSDPItem(ctx, disk, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeDiskWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListDisksRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		disk, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeDiskToSDPItem(ctx, disk, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeDiskWrapper) gcpComputeDiskToSDPItem(ctx context.Context, disk *computepb.Disk, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(disk, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeDisk.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            disk.GetLabels(),
	}

	// The resource URL for the disk type associated with this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/diskTypes/{diskType}
	// https://cloud.google.com/compute/docs/reference/rest/v1/diskTypes/get
	if diskType := disk.GetType(); diskType != "" {
		if strings.Contains(diskType, "/") {
			diskTypeName := gcpshared.LastPathComponent(diskType)
			if diskTypeName != "" {
				scope, err := gcpshared.ExtractScopeFromURI(ctx, diskType)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeDiskType.String(),
							Method: sdp.QueryMethod_GET,
							Query:  diskTypeName,
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

	// The resource URL for the image used to create this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/images/{image}
	// https://cloud.google.com/compute/docs/reference/rest/v1/images/get
	if sourceImage := disk.GetSourceImage(); sourceImage != "" {
		if strings.Contains(sourceImage, "/") {
			imageName := gcpshared.LastPathComponent(sourceImage)
			if imageName != "" {
				scope, err := gcpshared.ExtractScopeFromURI(ctx, sourceImage)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  imageName,
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

	// The resource URL for the snapshot used to create this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/snapshots/{snapshot}
	// https://cloud.google.com/compute/docs/reference/rest/v1/snapshots/get
	if sourceSnapshot := disk.GetSourceSnapshot(); sourceSnapshot != "" {
		if strings.Contains(sourceSnapshot, "/") {
			snapshotName := gcpshared.LastPathComponent(sourceSnapshot)
			if snapshotName != "" {
				scope, err := gcpshared.ExtractScopeFromURI(ctx, sourceSnapshot)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeSnapshot.String(),
							Method: sdp.QueryMethod_GET,
							Query:  snapshotName,
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

	// The resource URL for the instant snapshot used to create this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instantSnapshots/{instantSnapshot}
	// https://cloud.google.com/compute/docs/reference/rest/v1/instantSnapshots/get
	if sourceInstantSnapshot := disk.GetSourceInstantSnapshot(); sourceInstantSnapshot != "" {
		if strings.Contains(sourceInstantSnapshot, "/") {
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
				scope, err := gcpshared.ExtractScopeFromURI(ctx, sourceDisk)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeDisk.String(),
							Method: sdp.QueryMethod_GET,
							Query:  sourceDiskName,
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

	// The resource URLs for the resource policies associated with this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/resourcePolicies/{resourcePolicy}
	// https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies/get
	for _, rp := range disk.GetResourcePolicies() {
		if strings.Contains(rp, "/") {
			rpName := gcpshared.LastPathComponent(rp)
			if rpName != "" {
				scope, err := gcpshared.ExtractScopeFromURI(ctx, rp)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeResourcePolicy.String(),
							Method: sdp.QueryMethod_GET,
							Query:  rpName,
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
	// The resource URLs for the users (instances) using this disk.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{instance}
	// https://cloud.google.com/compute/docs/reference/rest/v1/instances/get
	for _, instance := range disk.GetUsers() {
		if strings.Contains(instance, "/") {
			instanceName := gcpshared.LastPathComponent(instance)
			if instanceName != "" {
				scope, err := gcpshared.ExtractScopeFromURI(ctx, instance)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeInstance.String(),
							Method: sdp.QueryMethod_GET,
							Query:  instanceName,
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
	}

	// The Encryption keys associated with this disk; appears in the following format:
	// "diskEncryptionKey.kmsKeyName": "projects/kms_project_id/locations/region/keyRings/key_region/cryptoKeys/key/cryptoKeysVersions/version
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
	// DiskEncryptionKey.kmsKeyName -> CloudKMSCryptoKeyVersion
	if diskEncryptionKey := disk.GetDiskEncryptionKey(); diskEncryptionKey != nil {
		if keyName := diskEncryptionKey.GetKmsKeyName(); keyName != "" {
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
					// Deleting a key might break the disk’s ability to function and have its data read
					// Deleting a disk in GCP does not affect its associated encryption key
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
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
					// Deleting a key might break the disk’s ability to function and have its data read
					// Deleting a disk in GCP does not affect its source image's encryption key
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
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
					// Deleting a key might break the disk’s ability to function and have its data read
					// Deleting a disk in GCP does not affect its source image's encryption key
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
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
				scope, err := gcpshared.ExtractScopeFromURI(ctx, sourceConsistencyGroupPolicy)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeResourcePolicy.String(),
							Method: sdp.QueryMethod_GET,
							Query:  rpName,
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

	// The Cloud Storage URI for a disk image (tarball .tar.gz or .vmdk) used to create this disk.
	// Format: gs://bucket-name/path/to/object or https://storage.googleapis.com/bucket-name/path/to/object
	// GET https://storage.googleapis.com/storage/v1/b/{bucket}
	// https://cloud.google.com/storage/docs/json_api/v1/buckets/get
	// Note: Storage Bucket adapter only supports GET method (not SEARCH), so we extract the bucket name
	// and use GET. We reuse the existing StorageBucket manual adapter linker to avoid duplicating
	// GCS URI parsing logic, which handles various formats:
	// - //storage.googleapis.com/projects/PROJECT_ID/buckets/BUCKET_ID
	// - https://storage.googleapis.com/projects/PROJECT_ID/buckets/BUCKET_ID
	// - gs://bucket-name
	// - gs://bucket-name/path/to/file
	// - bucket-name (without gs:// prefix)
	if sourceStorageObject := disk.GetSourceStorageObject(); sourceStorageObject != "" {
		blastPropagation := &sdp.BlastPropagation{In: true, Out: false}
		if linkFunc, ok := gcpshared.ManualAdapterLinksByAssetType[gcpshared.StorageBucket]; ok {
			linkedQuery := linkFunc(location.ProjectID, location.ToScope(), sourceStorageObject, blastPropagation)
			if linkedQuery != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, linkedQuery)
			}
		}
	}

	// The storage pool to create new disk in. URL or partial resource path accepted.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/storagePools/{storagePool}
	// https://cloud.google.com/compute/docs/reference/rest/v1/storagePools/get
	if storagePool := disk.GetStoragePool(); storagePool != "" {
		if strings.Contains(storagePool, "/") {
			storagePoolName := gcpshared.LastPathComponent(storagePool)
			if storagePoolName != "" {
				scope, err := gcpshared.ExtractScopeFromURI(ctx, storagePool)
				if err == nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeStoragePool.String(),
							Method: sdp.QueryMethod_GET,
							Query:  storagePoolName,
							Scope:  scope,
						},
						// If the Storage Pool is deleted or updated: The disk may fail to operate correctly or become invalid. If the disk is updated: The Storage Pool remains unaffected.
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	// Link async primary disk
	if asyncPrimaryDisk := disk.GetAsyncPrimaryDisk(); asyncPrimaryDisk != nil {
		if primaryDisk := asyncPrimaryDisk.GetDisk(); primaryDisk != "" {
			if strings.Contains(primaryDisk, "/") {
				primaryDiskName := gcpshared.LastPathComponent(primaryDisk)
				if primaryDiskName != "" {
					scope, err := gcpshared.ExtractScopeFromURI(ctx, primaryDisk)
					if err == nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeDisk.String(),
								Method: sdp.QueryMethod_GET,
								Query:  primaryDiskName,
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

		if consistencyGroupPolicy := asyncPrimaryDisk.GetConsistencyGroupPolicy(); consistencyGroupPolicy != "" {
			if strings.Contains(consistencyGroupPolicy, "/") {
				policyName := gcpshared.LastPathComponent(consistencyGroupPolicy)
				if policyName != "" {
					scope, err := gcpshared.ExtractScopeFromURI(ctx, consistencyGroupPolicy)
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
	}

	// Link async secondary disks
	for _, asyncSecondaryDisk := range disk.GetAsyncSecondaryDisks() {
		if asyncReplicationDisk := asyncSecondaryDisk.GetAsyncReplicationDisk(); asyncReplicationDisk != nil {
			if secondaryDisk := asyncReplicationDisk.GetDisk(); secondaryDisk != "" {
				if strings.Contains(secondaryDisk, "/") {
					secondaryDiskName := gcpshared.LastPathComponent(secondaryDisk)
					if secondaryDiskName != "" {
						scope, err := gcpshared.ExtractScopeFromURI(ctx, secondaryDisk)
						if err == nil {
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   gcpshared.ComputeDisk.String(),
									Method: sdp.QueryMethod_GET,
									Query:  secondaryDiskName,
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

			if consistencyGroupPolicy := asyncReplicationDisk.GetConsistencyGroupPolicy(); consistencyGroupPolicy != "" {
				if strings.Contains(consistencyGroupPolicy, "/") {
					policyName := gcpshared.LastPathComponent(consistencyGroupPolicy)
					if policyName != "" {
						scope, err := gcpshared.ExtractScopeFromURI(ctx, consistencyGroupPolicy)
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
		}
	}

	// Set health status
	switch disk.GetStatus() {
	case computepb.Disk_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.Disk_CREATING.String(), computepb.Disk_RESTORING.String(), computepb.Disk_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Disk_FAILED.String(), computepb.Disk_UNAVAILABLE.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.Disk_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	}

	return sdpItem, nil
}
