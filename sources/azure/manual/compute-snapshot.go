package manual

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeSnapshotLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeSnapshot)

type computeSnapshotWrapper struct {
	client clients.SnapshotsClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeSnapshot(client clients.SnapshotsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &computeSnapshotWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ComputeSnapshot,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/snapshots/list-by-resource-group
func (c computeSnapshotWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, snapshot := range page.Value {
			if snapshot.Name == nil {
				continue
			}
			item, sdpErr := c.azureSnapshotToSDPItem(snapshot, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeSnapshotWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, snapshot := range page.Value {
			if snapshot.Name == nil {
				continue
			}
			item, sdpErr := c.azureSnapshotToSDPItem(snapshot, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/snapshots/get
func (c computeSnapshotWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the snapshot name"), scope, c.Type())
	}
	snapshotName := queryParts[0]
	if snapshotName == "" {
		return nil, azureshared.QueryError(errors.New("snapshotName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	result, err := c.client.Get(ctx, rgScope.ResourceGroup, snapshotName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureSnapshotToSDPItem(&result.Snapshot, scope)
}

func (c computeSnapshotWrapper) azureSnapshotToSDPItem(snapshot *armcompute.Snapshot, scope string) (*sdp.Item, *sdp.QueryError) {
	if snapshot.Name == nil {
		return nil, azureshared.QueryError(errors.New("snapshot name is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(snapshot, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeSnapshot.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(snapshot.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Health status from ProvisioningState
	if snapshot.Properties != nil && snapshot.Properties.ProvisioningState != nil {
		switch *snapshot.Properties.ProvisioningState {
		case "Succeeded":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case "Creating", "Updating", "Deleting":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Failed", "Canceled":
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link to Disk Access from Properties.DiskAccessID
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-accesses/get
	if snapshot.Properties != nil && snapshot.Properties.DiskAccessID != nil && *snapshot.Properties.DiskAccessID != "" {
		diskAccessName := azureshared.ExtractResourceName(*snapshot.Properties.DiskAccessID)
		if diskAccessName != "" {
			extractedScope := azureshared.ExtractScopeFromResourceID(*snapshot.Properties.DiskAccessID)
			if extractedScope == "" {
				extractedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeDiskAccess.String(),
					Method: sdp.QueryMethod_GET,
					Query:  diskAccessName,
					Scope:  extractedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Disk Access is deleted/modified → snapshot private endpoint access is affected
					Out: false, // If snapshot is deleted → Disk Access remains
				},
			})
		}
	}

	// Link to Disk Encryption Set from Properties.Encryption.DiskEncryptionSetID
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
	if snapshot.Properties != nil && snapshot.Properties.Encryption != nil && snapshot.Properties.Encryption.DiskEncryptionSetID != nil && *snapshot.Properties.Encryption.DiskEncryptionSetID != "" {
		encryptionSetName := azureshared.ExtractResourceName(*snapshot.Properties.Encryption.DiskEncryptionSetID)
		if encryptionSetName != "" {
			extractedScope := azureshared.ExtractScopeFromResourceID(*snapshot.Properties.Encryption.DiskEncryptionSetID)
			if extractedScope == "" {
				extractedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeDiskEncryptionSet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  encryptionSetName,
					Scope:  extractedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Disk Encryption Set is deleted/modified → snapshot encryption is affected
					Out: false, // If snapshot is deleted → Disk Encryption Set remains
				},
			})
		}
	}

	// Link to Disk Encryption Set from Properties.SecurityProfile.SecureVMDiskEncryptionSetID
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
	if snapshot.Properties != nil && snapshot.Properties.SecurityProfile != nil && snapshot.Properties.SecurityProfile.SecureVMDiskEncryptionSetID != nil && *snapshot.Properties.SecurityProfile.SecureVMDiskEncryptionSetID != "" {
		encryptionSetName := azureshared.ExtractResourceName(*snapshot.Properties.SecurityProfile.SecureVMDiskEncryptionSetID)
		if encryptionSetName != "" {
			extractedScope := azureshared.ExtractScopeFromResourceID(*snapshot.Properties.SecurityProfile.SecureVMDiskEncryptionSetID)
			if extractedScope == "" {
				extractedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeDiskEncryptionSet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  encryptionSetName,
					Scope:  extractedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Disk Encryption Set is deleted/modified → snapshot encryption is affected
					Out: false, // If snapshot is deleted → Disk Encryption Set remains
				},
			})
		}
	}

	// Link to source resources from Properties.CreationData
	if snapshot.Properties != nil && snapshot.Properties.CreationData != nil {
		creationData := snapshot.Properties.CreationData

		// Link to source Disk or Snapshot from SourceResourceID
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disks/get
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/snapshots/get
		if creationData.SourceResourceID != nil && *creationData.SourceResourceID != "" {
			sourceResourceID := *creationData.SourceResourceID
			sourceResourceIDLower := strings.ToLower(sourceResourceID)
			if strings.Contains(sourceResourceIDLower, "/disks/") {
				diskName := azureshared.ExtractResourceName(sourceResourceID)
				if diskName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(sourceResourceID)
					if extractedScope == "" {
						extractedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeDisk.String(),
							Method: sdp.QueryMethod_GET,
							Query:  diskName,
							Scope:  extractedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If source disk is deleted/modified → snapshot may be affected
							Out: false, // If snapshot is deleted → source disk remains
						},
					})
				}
			} else if strings.Contains(sourceResourceIDLower, "/snapshots/") {
				snapshotName := azureshared.ExtractResourceName(sourceResourceID)
				if snapshotName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(sourceResourceID)
					if extractedScope == "" {
						extractedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSnapshot.String(),
							Method: sdp.QueryMethod_GET,
							Query:  snapshotName,
							Scope:  extractedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If source snapshot is deleted/modified → this snapshot may be affected
							Out: false, // If this snapshot is deleted → source snapshot remains
						},
					})
				}
			}
		}

		// Link to Storage Account from StorageAccountID
		// Reference: https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/get-properties
		if creationData.StorageAccountID != nil && *creationData.StorageAccountID != "" {
			storageAccountName := azureshared.ExtractResourceName(*creationData.StorageAccountID)
			if storageAccountName != "" {
				extractedScope := azureshared.ExtractScopeFromResourceID(*creationData.StorageAccountID)
				if extractedScope == "" {
					extractedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.StorageAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  storageAccountName,
						Scope:  extractedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Storage Account is deleted/modified → snapshot import may fail
						Out: false, // If snapshot is deleted → Storage Account remains
					},
				})
			}
		}

		// Link to Storage Account and DNS from SourceURI (blob URI used for Import)
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/snapshots/create-or-update
		if creationData.SourceURI != nil && *creationData.SourceURI != "" {
			sourceURI := *creationData.SourceURI
			storageAccountName := azureshared.ExtractStorageAccountNameFromBlobURI(sourceURI)
			if storageAccountName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.StorageAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  storageAccountName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Storage Account is deleted/modified → snapshot import blob becomes inaccessible
						Out: false, // If snapshot is deleted → Storage Account remains
					},
				})

				containerName := azureshared.ExtractContainerNameFromBlobURI(sourceURI)
				if containerName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.StorageBlobContainer.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(storageAccountName, containerName),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If blob container is deleted/modified → snapshot import source is lost
							Out: false, // If snapshot is deleted → blob container remains
						},
					})
				}
			}

			if strings.HasPrefix(sourceURI, "http://") || strings.HasPrefix(sourceURI, "https://") {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkHTTP.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  sourceURI,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // If HTTP endpoint is unavailable → snapshot import source is lost
						Out: true, // Bidirectional: changes to either side may affect the other
					},
				})
			}

			host := azureshared.ExtractDNSFromURL(sourceURI)
			if host != "" {
				if net.ParseIP(host) != nil {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  host,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					})
				} else {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkDNS.String(),
							Method: sdp.QueryMethod_SEARCH,
							Query:  host,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					})
				}
			}
		}

		// Link to Image from ImageReference.ID
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/images/get
		if creationData.ImageReference != nil && creationData.ImageReference.ID != nil && *creationData.ImageReference.ID != "" {
			imageID := *creationData.ImageReference.ID
			imageIDLower := strings.ToLower(imageID)
			if strings.Contains(imageIDLower, "/images/") && !strings.Contains(imageIDLower, "/galleries/") {
				imageName := azureshared.ExtractResourceName(imageID)
				if imageName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(imageID)
					if extractedScope == "" {
						extractedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  imageName,
							Scope:  extractedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Image is deleted/modified → snapshot created from image may be affected
							Out: false, // If snapshot is deleted → Image remains
						},
					})
				}
			}
		}

		// Link to Gallery Image from GalleryImageReference
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/gallery-images/get
		if creationData.GalleryImageReference != nil {
			if creationData.GalleryImageReference.ID != nil && *creationData.GalleryImageReference.ID != "" {
				galleryImageID := *creationData.GalleryImageReference.ID
				parts := azureshared.ExtractPathParamsFromResourceID(galleryImageID, []string{"galleries", "images", "versions"})
				if len(parts) >= 3 {
					galleryName := parts[0]
					imageName := parts[1]
					version := parts[2]
					extractedScope := azureshared.ExtractScopeFromResourceID(galleryImageID)
					if extractedScope == "" {
						extractedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSharedGalleryImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(galleryName, imageName, version),
							Scope:  extractedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Gallery Image is deleted/modified → snapshot created from image may be affected
							Out: false, // If snapshot is deleted → Gallery Image remains
						},
					})
				}
			}

			if creationData.GalleryImageReference.SharedGalleryImageID != nil && *creationData.GalleryImageReference.SharedGalleryImageID != "" {
				sharedGalleryImageID := *creationData.GalleryImageReference.SharedGalleryImageID
				parts := azureshared.ExtractPathParamsFromResourceID(sharedGalleryImageID, []string{"galleries", "images", "versions"})
				if len(parts) >= 3 {
					galleryName := parts[0]
					imageName := parts[1]
					version := parts[2]
					extractedScope := azureshared.ExtractScopeFromResourceID(sharedGalleryImageID)
					if extractedScope == "" {
						extractedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSharedGalleryImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(galleryName, imageName, version),
							Scope:  extractedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Gallery Image is deleted/modified → snapshot created from image may be affected
							Out: false, // If snapshot is deleted → Gallery Image remains
						},
					})
				}
			}

			if creationData.GalleryImageReference.CommunityGalleryImageID != nil && *creationData.GalleryImageReference.CommunityGalleryImageID != "" {
				communityGalleryImageID := *creationData.GalleryImageReference.CommunityGalleryImageID
				parts := azureshared.ExtractPathParamsFromResourceID(communityGalleryImageID, []string{"Images", "Versions"})
				if len(parts) >= 2 {
					imageName := parts[0]
					version := parts[1]
					allParts := strings.Split(strings.Trim(communityGalleryImageID, "/"), "/")
					communityGalleryName := ""
					for i, part := range allParts {
						if strings.EqualFold(part, "CommunityGalleries") && i+1 < len(allParts) {
							communityGalleryName = allParts[i+1]
							break
						}
					}
					if communityGalleryName != "" {
						extractedScope := azureshared.ExtractScopeFromResourceID(communityGalleryImageID)
						if extractedScope == "" {
							extractedScope = scope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeCommunityGalleryImage.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(communityGalleryName, imageName, version),
								Scope:  extractedScope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Community Gallery Image is deleted/modified → snapshot may be affected
								Out: false, // If snapshot is deleted → Community Gallery Image remains
							},
						})
					}
				}
			}
		}

		// Link to Elastic SAN Volume Snapshot from ElasticSanResourceID
		// Reference: https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-snapshots/get
		if creationData.ElasticSanResourceID != nil && *creationData.ElasticSanResourceID != "" {
			elasticSanResourceID := *creationData.ElasticSanResourceID
			parts := azureshared.ExtractPathParamsFromResourceID(elasticSanResourceID, []string{"elasticSans", "volumegroups", "snapshots"})
			if len(parts) >= 3 {
				elasticSanName := parts[0]
				volumeGroupName := parts[1]
				esSnapshotName := parts[2]
				extractedScope := azureshared.ExtractScopeFromResourceID(elasticSanResourceID)
				if extractedScope == "" {
					extractedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ElasticSanVolumeSnapshot.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(elasticSanName, volumeGroupName, esSnapshotName),
						Scope:  extractedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Elastic SAN snapshot is deleted/modified → this snapshot may be affected
						Out: false, // If this snapshot is deleted → Elastic SAN snapshot remains
					},
				})
			}
		}
	}

	// Link to Key Vault resources from EncryptionSettingsCollection
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/vaults/get
	if snapshot.Properties != nil && snapshot.Properties.EncryptionSettingsCollection != nil && snapshot.Properties.EncryptionSettingsCollection.EncryptionSettings != nil {
		for _, encryptionSetting := range snapshot.Properties.EncryptionSettingsCollection.EncryptionSettings {
			if encryptionSetting == nil {
				continue
			}

			// Link to Key Vault from DiskEncryptionKey.SourceVault.ID
			if encryptionSetting.DiskEncryptionKey != nil && encryptionSetting.DiskEncryptionKey.SourceVault != nil && encryptionSetting.DiskEncryptionKey.SourceVault.ID != nil && *encryptionSetting.DiskEncryptionKey.SourceVault.ID != "" {
				vaultName := azureshared.ExtractResourceName(*encryptionSetting.DiskEncryptionKey.SourceVault.ID)
				if vaultName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(*encryptionSetting.DiskEncryptionKey.SourceVault.ID)
					if extractedScope == "" {
						extractedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  extractedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault is deleted/modified → snapshot encryption key access is affected
							Out: false, // If snapshot is deleted → Key Vault remains
						},
					})
				}
			}

			// Link to Key Vault Secret from DiskEncryptionKey.SecretURL
			if encryptionSetting.DiskEncryptionKey != nil && encryptionSetting.DiskEncryptionKey.SecretURL != nil && *encryptionSetting.DiskEncryptionKey.SecretURL != "" {
				secretURL := *encryptionSetting.DiskEncryptionKey.SecretURL
				vaultName := azureshared.ExtractVaultNameFromURI(secretURL)
				secretName := azureshared.ExtractSecretNameFromURI(secretURL)
				if vaultName != "" && secretName != "" {
					// Derive scope from the DiskEncryptionKey's SourceVault when available
					secretScope := scope
					if encryptionSetting.DiskEncryptionKey.SourceVault != nil && encryptionSetting.DiskEncryptionKey.SourceVault.ID != nil {
						if extracted := azureshared.ExtractScopeFromResourceID(*encryptionSetting.DiskEncryptionKey.SourceVault.ID); extracted != "" {
							secretScope = extracted
						}
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultSecret.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(vaultName, secretName),
							Scope:  secretScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault Secret is deleted/modified → snapshot encryption key is affected
							Out: false, // If snapshot is deleted → Key Vault Secret remains
						},
					})
				}

				secretHost := azureshared.ExtractDNSFromURL(secretURL)
				if secretHost != "" {
					if net.ParseIP(secretHost) != nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkIP.String(),
								Method: sdp.QueryMethod_GET,
								Query:  secretHost,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: true,
							},
						})
					} else {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkDNS.String(),
								Method: sdp.QueryMethod_SEARCH,
								Query:  secretHost,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: true,
							},
						})
					}
				}
			}

			// Link to Key Vault from KeyEncryptionKey.SourceVault.ID
			if encryptionSetting.KeyEncryptionKey != nil && encryptionSetting.KeyEncryptionKey.SourceVault != nil && encryptionSetting.KeyEncryptionKey.SourceVault.ID != nil && *encryptionSetting.KeyEncryptionKey.SourceVault.ID != "" {
				vaultName := azureshared.ExtractResourceName(*encryptionSetting.KeyEncryptionKey.SourceVault.ID)
				if vaultName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(*encryptionSetting.KeyEncryptionKey.SourceVault.ID)
					if extractedScope == "" {
						extractedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  extractedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault is deleted/modified → key encryption key access is affected
							Out: false, // If snapshot is deleted → Key Vault remains
						},
					})
				}
			}

			// Link to Key Vault Key from KeyEncryptionKey.KeyURL
			if encryptionSetting.KeyEncryptionKey != nil && encryptionSetting.KeyEncryptionKey.KeyURL != nil && *encryptionSetting.KeyEncryptionKey.KeyURL != "" {
				keyURL := *encryptionSetting.KeyEncryptionKey.KeyURL
				vaultName := azureshared.ExtractVaultNameFromURI(keyURL)
				keyName := azureshared.ExtractKeyNameFromURI(keyURL)
				if vaultName != "" && keyName != "" {
					// Derive scope from the KeyEncryptionKey's SourceVault when available
					keyScope := scope
					if encryptionSetting.KeyEncryptionKey.SourceVault != nil && encryptionSetting.KeyEncryptionKey.SourceVault.ID != nil {
						if extracted := azureshared.ExtractScopeFromResourceID(*encryptionSetting.KeyEncryptionKey.SourceVault.ID); extracted != "" {
							keyScope = extracted
						}
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultKey.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(vaultName, keyName),
							Scope:  keyScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault Key is deleted/modified → key encryption key is affected
							Out: false, // If snapshot is deleted → Key Vault Key remains
						},
					})
				}

				keyHost := azureshared.ExtractDNSFromURL(keyURL)
				if keyHost != "" {
					if net.ParseIP(keyHost) != nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkIP.String(),
								Method: sdp.QueryMethod_GET,
								Query:  keyHost,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: true,
							},
						})
					} else {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkDNS.String(),
								Method: sdp.QueryMethod_SEARCH,
								Query:  keyHost,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: true,
							},
						})
					}
				}
			}
		}
	}

	return sdpItem, nil
}

func (c computeSnapshotWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeSnapshotLookupByName,
	}
}

func (c computeSnapshotWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeDisk,
		azureshared.ComputeSnapshot,
		azureshared.ComputeDiskAccess,
		azureshared.ComputeDiskEncryptionSet,
		azureshared.ComputeImage,
		azureshared.ComputeSharedGalleryImage,
		azureshared.ComputeCommunityGalleryImage,
		azureshared.StorageAccount,
		azureshared.StorageBlobContainer,
		azureshared.ElasticSanVolumeSnapshot,
		azureshared.KeyVaultVault,
		azureshared.KeyVaultSecret,
		azureshared.KeyVaultKey,
		stdlib.NetworkDNS,
		stdlib.NetworkHTTP,
		stdlib.NetworkIP,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/snapshot
func (c computeSnapshotWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_snapshot.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (c computeSnapshotWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/snapshots/read",
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/compute
func (c computeSnapshotWrapper) PredefinedRole() string {
	return "Reader"
}
