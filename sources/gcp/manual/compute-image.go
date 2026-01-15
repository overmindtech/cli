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

var ComputeImageLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeImage)

type computeImageWrapper struct {
	client gcpshared.ComputeImagesClient

	*gcpshared.ProjectBase
}

// NewComputeImage creates a new computeImageWrapper instance
func NewComputeImage(client gcpshared.ComputeImagesClient, projectID string) sources.ListableWrapper {
	return &computeImageWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeImage,
		),
	}
}

func (c computeImageWrapper) IAMPermissions() []string {
	return []string{
		"compute.images.get",
		"compute.images.list",
	}
}

func (c computeImageWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

// TerraformMappings returns the Terraform mappings for the compute image wrapper
func (c computeImageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_health_check#argument-reference
			TerraformQueryMap: "google_compute_image.name",
		},
	}
}

// GetLookups returns the lookups for the compute image wrapper
func (c computeImageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeImageLookupByName,
	}
}

// Get retrieves a compute image by its name
func (c computeImageWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetImageRequest{
		Project: c.ProjectID(),
		Image:   queryParts[0],
	}

	image, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeImageToSDPItem(ctx, image)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil

}

// List lists compute images and converts them to sdp.Items.
func (c computeImageWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListImagesRequest{
		Project: c.ProjectID(),
	})

	var items []*sdp.Item
	for {
		image, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeImageToSDPItem(ctx, image)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream lists compute images and sends them as items to the provided stream.
func (c computeImageWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListImagesRequest{
		Project: c.ProjectID(),
	})

	for {
		image, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeImageToSDPItem(ctx, image)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// PotentialLinks returns the potential links for the compute image wrapper
func (c computeImageWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeDisk,
		gcpshared.ComputeSnapshot,
		gcpshared.ComputeImage,
		gcpshared.ComputeLicense,
		gcpshared.StorageBucket,
		gcpshared.CloudKMSCryptoKey,
		gcpshared.CloudKMSCryptoKeyVersion,
		gcpshared.IAMServiceAccount,
	)
}

func (c computeImageWrapper) gcpComputeImageToSDPItem(ctx context.Context, image *computepb.Image) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(image, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeImage.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            image.GetLabels(),
	}

	switch image.GetStatus() {
	case computepb.Image_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.Image_FAILED.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.Image_PENDING.String(),
		computepb.Image_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Image_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	}

	// The URL of the disk used to create this image.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/disks/{disk}
	// https://cloud.google.com/compute/docs/reference/rest/v1/disks/get
	// If the source disk is deleted or updated: The image may become invalid or fail to create new instances. If the image is updated: The source disk remains unaffected.
	if sourceDisk := image.GetSourceDisk(); sourceDisk != "" {
		diskName := gcpshared.LastPathComponent(sourceDisk)
		if diskName != "" {
			scope, err := gcpshared.ExtractScopeFromURI(ctx, sourceDisk)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeDisk.String(),
						Method: sdp.QueryMethod_GET,
						Query:  diskName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// The URL of the snapshot used to create this image.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/snapshots/{snapshot}
	// https://cloud.google.com/compute/docs/reference/rest/v1/snapshots/get
	// If the source snapshot is deleted or updated: The image may become invalid or fail to create new instances. If the image is updated: The source snapshot remains unaffected.
	if sourceSnapshot := image.GetSourceSnapshot(); sourceSnapshot != "" {
		snapshotName := gcpshared.LastPathComponent(sourceSnapshot)
		if snapshotName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeSnapshot.String(),
					Method: sdp.QueryMethod_GET,
					Query:  snapshotName,
					Scope:  c.ProjectID(),
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
		}
	}

	// The URL of source image used to create this image.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/images/{image}
	// https://cloud.google.com/compute/docs/reference/rest/v1/images/get
	// If the source image is deleted or updated: The image may become invalid or fail to create new instances. If the image is updated: The source image remains unaffected.
	if sourceImage := image.GetSourceImage(); sourceImage != "" {
		imageName := gcpshared.LastPathComponent(sourceImage)
		if imageName != "" {
			// Source image can be from a different project, extract project ID from URI
			projectID := gcpshared.ExtractPathParam("projects", sourceImage)
			scope := c.ProjectID()
			if projectID != "" {
				scope = projectID
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeImage.String(),
					Method: sdp.QueryMethod_GET,
					Query:  imageName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
		}
	}

	// The resource URLs for the licenses associated with this image.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/licenses/{license}
	// https://cloud.google.com/compute/docs/reference/rest/v1/licenses/get
	// If the license is deleted or updated: The image may become invalid or fail to create new instances. If the image is updated: The license remains unaffected.
	for _, license := range image.GetLicenses() {
		licenseName := gcpshared.LastPathComponent(license)
		if licenseName != "" {
			// License can be from a different project, extract project ID from URI
			projectID := gcpshared.ExtractPathParam("projects", license)
			scope := c.ProjectID()
			if projectID != "" {
				scope = projectID
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeLicense.String(),
					Method: sdp.QueryMethod_GET,
					Query:  licenseName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
			})
		}
	}

	// The full GCS URL of raw disk archive used to create this image.
	// Format: gs://bucket-name/path/to/object or https://storage.googleapis.com/bucket-name/path/to/object
	// GET https://storage.googleapis.com/storage/v1/b/{bucket}
	// https://cloud.google.com/storage/docs/json_api/v1/buckets/get
	// If the Storage Bucket is deleted or inaccessible: The image may fail to be created or restored. If the image is updated: The bucket remains unaffected.
	if rawDisk := image.GetRawDisk(); rawDisk != nil {
		if rawDiskSource := rawDisk.GetSource(); rawDiskSource != "" {
			blastPropagation := &sdp.BlastPropagation{
				In:  true,
				Out: false,
			}
			if linkFunc, ok := gcpshared.ManualAdapterLinksByAssetType[gcpshared.StorageBucket]; ok {
				linkedQuery := linkFunc(c.ProjectID(), c.DefaultScope(), rawDiskSource, blastPropagation)
				if linkedQuery != nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, linkedQuery)
				}
			}
		}
	}

	// The customer-supplied encryption key for the image; appears in the following format:
	// "imageEncryptionKey.kmsKeyName": "projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{cryptoKey}/cryptoKeyVersions/{version}"
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
	// If the crypto key version is deleted or updated: The image may not be able to be decrypted or used. If the image is updated: The crypto key version remains unaffected.
	if imageEncryptionKey := image.GetImageEncryptionKey(); imageEncryptionKey != nil {
		if keyName := imageEncryptionKey.GetKmsKeyName(); keyName != "" {
			location := gcpshared.ExtractPathParam("locations", keyName)
			keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
			cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
			cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

			// If version is included, link to CryptoKeyVersion; otherwise link to CryptoKey
			if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			} else if location != "" && keyRing != "" && cryptoKey != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKey.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey),
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}

		// The service account used for KMS operations on the image.
		// GET https://iam.googleapis.com/v1/projects/{project}/serviceAccounts/{serviceAccount}
		// https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts/get
		// If the service account is deleted or updated: The image may fail to perform KMS operations. If the image is updated: The service account remains unaffected.
		if kmsKeyServiceAccount := imageEncryptionKey.GetKmsKeyServiceAccount(); kmsKeyServiceAccount != "" {
			// Extract email from service account format: projects/{project}/serviceAccounts/{email} or just the email
			serviceAccountEmail := kmsKeyServiceAccount
			if strings.Contains(kmsKeyServiceAccount, "/serviceAccounts/") {
				serviceAccountEmail = gcpshared.LastPathComponent(kmsKeyServiceAccount)
			}
			if serviceAccountEmail != "" {
				// Service account can be from a different project, extract project ID from URI
				projectID := c.ProjectID()
				if strings.Contains(kmsKeyServiceAccount, "/projects/") {
					extractedProjectID := gcpshared.ExtractPathParam("projects", kmsKeyServiceAccount)
					if extractedProjectID != "" {
						projectID = extractedProjectID
					}
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.IAMServiceAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  serviceAccountEmail,
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// The customer-supplied encryption key of the source image; appears in the following format:
	// "sourceImageEncryptionKey.kmsKeyName": "projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{cryptoKey}/cryptoKeyVersions/{version}"
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
	// If the crypto key version is deleted or updated: The image may not be able to access the source image. If the image is updated: The crypto key version remains unaffected.
	if sourceImageEncryptionKey := image.GetSourceImageEncryptionKey(); sourceImageEncryptionKey != nil {
		if keyName := sourceImageEncryptionKey.GetKmsKeyName(); keyName != "" {
			location := gcpshared.ExtractPathParam("locations", keyName)
			keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
			cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
			cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

			if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			} else if location != "" && keyRing != "" && cryptoKey != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKey.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey),
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}

		// The service account used for KMS operations on the source image.
		// GET https://iam.googleapis.com/v1/projects/{project}/serviceAccounts/{serviceAccount}
		// https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts/get
		// If the service account is deleted or updated: The image may fail to perform KMS operations on the source image. If the image is updated: The service account remains unaffected.
		if kmsKeyServiceAccount := sourceImageEncryptionKey.GetKmsKeyServiceAccount(); kmsKeyServiceAccount != "" {
			serviceAccountEmail := kmsKeyServiceAccount
			if strings.Contains(kmsKeyServiceAccount, "/serviceAccounts/") {
				serviceAccountEmail = gcpshared.LastPathComponent(kmsKeyServiceAccount)
			}
			if serviceAccountEmail != "" {
				projectID := c.ProjectID()
				if strings.Contains(kmsKeyServiceAccount, "/projects/") {
					extractedProjectID := gcpshared.ExtractPathParam("projects", kmsKeyServiceAccount)
					if extractedProjectID != "" {
						projectID = extractedProjectID
					}
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.IAMServiceAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  serviceAccountEmail,
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// The customer-supplied encryption key of the source snapshot; appears in the following format:
	// "sourceSnapshotEncryptionKey.kmsKeyName": "projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{cryptoKey}/cryptoKeyVersions/{version}"
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
	// If the crypto key version is deleted or updated: The image may not be able to access the source snapshot. If the image is updated: The crypto key version remains unaffected.
	if sourceSnapshotEncryptionKey := image.GetSourceSnapshotEncryptionKey(); sourceSnapshotEncryptionKey != nil {
		if keyName := sourceSnapshotEncryptionKey.GetKmsKeyName(); keyName != "" {
			location := gcpshared.ExtractPathParam("locations", keyName)
			keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
			cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
			cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

			if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			} else if location != "" && keyRing != "" && cryptoKey != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKey.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey),
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}

		// The service account used for KMS operations on the source snapshot.
		// GET https://iam.googleapis.com/v1/projects/{project}/serviceAccounts/{serviceAccount}
		// https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts/get
		// If the service account is deleted or updated: The image may fail to perform KMS operations on the source snapshot. If the image is updated: The service account remains unaffected.
		if kmsKeyServiceAccount := sourceSnapshotEncryptionKey.GetKmsKeyServiceAccount(); kmsKeyServiceAccount != "" {
			serviceAccountEmail := kmsKeyServiceAccount
			if strings.Contains(kmsKeyServiceAccount, "/serviceAccounts/") {
				serviceAccountEmail = gcpshared.LastPathComponent(kmsKeyServiceAccount)
			}
			if serviceAccountEmail != "" {
				projectID := c.ProjectID()
				if strings.Contains(kmsKeyServiceAccount, "/projects/") {
					extractedProjectID := gcpshared.ExtractPathParam("projects", kmsKeyServiceAccount)
					if extractedProjectID != "" {
						projectID = extractedProjectID
					}
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.IAMServiceAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  serviceAccountEmail,
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// The URL of suggested replacement image when this image is deprecated.
	// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/images/{image}
	// https://cloud.google.com/compute/docs/reference/rest/v1/images/get
	// If the replacement image is deleted or updated: The deprecation path may break. If the image is updated: The replacement image remains unaffected.
	if deprecated := image.GetDeprecated(); deprecated != nil {
		if replacement := deprecated.GetReplacement(); replacement != "" {
			replacementImageName := gcpshared.LastPathComponent(replacement)
			if replacementImageName != "" {
				// Replacement image can be from a different project, extract project ID from URI
				projectID := c.ProjectID()
				if strings.Contains(replacement, "/projects/") {
					extractedProjectID := gcpshared.ExtractPathParam("projects", replacement)
					if extractedProjectID != "" {
						projectID = extractedProjectID
					}
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeImage.String(),
						Method: sdp.QueryMethod_GET,
						Query:  replacementImageName,
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	return sdpItem, nil
}
