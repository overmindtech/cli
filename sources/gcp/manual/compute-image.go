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

// NewComputeImage creates a new computeImageWrapper instance.
func NewComputeImage(client gcpshared.ComputeImagesClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeImageWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
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

func (c computeImageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_image.name",
		},
	}
}

func (c computeImageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeImageLookupByName,
	}
}

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

func (c computeImageWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetImageRequest{
		Project: location.ProjectID,
		Image:   queryParts[0],
	}

	image, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeImageToSDPItem(ctx, image, location)
}

func (c computeImageWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListImagesRequest{
		Project: location.ProjectID,
	})

	var items []*sdp.Item
	for {
		image, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeImageToSDPItem(ctx, image, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeImageWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListImagesRequest{
		Project: location.ProjectID,
	})

	for {
		image, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeImageToSDPItem(ctx, image, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeImageWrapper) gcpComputeImageToSDPItem(ctx context.Context, image *computepb.Image, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           location.ToScope(),
		Tags:            image.GetLabels(),
	}

	switch image.GetStatus() {
	case computepb.Image_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.Image_FAILED.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.Image_PENDING.String(), computepb.Image_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Image_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	}

	// Link to source disk
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
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Link to source snapshot
	if sourceSnapshot := image.GetSourceSnapshot(); sourceSnapshot != "" {
		snapshotName := gcpshared.LastPathComponent(sourceSnapshot)
		if snapshotName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeSnapshot.String(),
					Method: sdp.QueryMethod_GET,
					Query:  snapshotName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to source image
	if sourceImage := image.GetSourceImage(); sourceImage != "" {
		imageName := gcpshared.LastPathComponent(sourceImage)
		if imageName != "" {
			projectID := gcpshared.ExtractPathParam("projects", sourceImage)
			scope := location.ProjectID
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
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to licenses
	for _, license := range image.GetLicenses() {
		licenseName := gcpshared.LastPathComponent(license)
		if licenseName != "" {
			projectID := gcpshared.ExtractPathParam("projects", license)
			scope := location.ProjectID
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
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to raw disk storage bucket
	if rawDisk := image.GetRawDisk(); rawDisk != nil {
		if rawDiskSource := rawDisk.GetSource(); rawDiskSource != "" {
			blastPropagation := &sdp.BlastPropagation{In: true, Out: false}
			if linkFunc, ok := gcpshared.ManualAdapterLinksByAssetType[gcpshared.StorageBucket]; ok {
				linkedQuery := linkFunc(location.ProjectID, location.ToScope(), rawDiskSource, blastPropagation)
				if linkedQuery != nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, linkedQuery)
				}
			}
		}
	}

	// Link to image encryption key
	if imageEncryptionKey := image.GetImageEncryptionKey(); imageEncryptionKey != nil {
		c.addKMSKeyLinks(sdpItem, imageEncryptionKey.GetKmsKeyName(), imageEncryptionKey.GetKmsKeyServiceAccount(), location)
	}

	// Link to source image encryption key
	if sourceImageEncryptionKey := image.GetSourceImageEncryptionKey(); sourceImageEncryptionKey != nil {
		c.addKMSKeyLinks(sdpItem, sourceImageEncryptionKey.GetKmsKeyName(), sourceImageEncryptionKey.GetKmsKeyServiceAccount(), location)
	}

	// Link to source snapshot encryption key
	if sourceSnapshotEncryptionKey := image.GetSourceSnapshotEncryptionKey(); sourceSnapshotEncryptionKey != nil {
		c.addKMSKeyLinks(sdpItem, sourceSnapshotEncryptionKey.GetKmsKeyName(), sourceSnapshotEncryptionKey.GetKmsKeyServiceAccount(), location)
	}

	// Link to replacement image
	if deprecated := image.GetDeprecated(); deprecated != nil {
		if replacement := deprecated.GetReplacement(); replacement != "" {
			replacementImageName := gcpshared.LastPathComponent(replacement)
			if replacementImageName != "" {
				projectID := gcpshared.ExtractPathParam("projects", replacement)
				scope := location.ProjectID
				if projectID != "" {
					scope = projectID
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeImage.String(),
						Method: sdp.QueryMethod_GET,
						Query:  replacementImageName,
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

	return sdpItem, nil
}

func (c computeImageWrapper) addKMSKeyLinks(sdpItem *sdp.Item, keyName, kmsKeyServiceAccount string, location gcpshared.LocationInfo) {
	if keyName != "" {
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
		} else if loc != "" && keyRing != "" && cryptoKey != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.CloudKMSCryptoKey.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(loc, keyRing, cryptoKey),
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	if kmsKeyServiceAccount != "" {
		serviceAccountEmail := kmsKeyServiceAccount
		if strings.Contains(kmsKeyServiceAccount, "/serviceAccounts/") {
			serviceAccountEmail = gcpshared.LastPathComponent(kmsKeyServiceAccount)
		}
		if serviceAccountEmail != "" {
			projectID := location.ProjectID
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
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}
}
