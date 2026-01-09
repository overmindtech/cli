package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeDiskLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeDisk)

type computeDiskWrapper struct {
	client clients.DisksClient
	*azureshared.ResourceGroupBase
}

func NewComputeDisk(client clients.DisksClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &computeDiskWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ComputeDisk,
		),
	}
}

func (c computeDiskWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := c.client.NewListByResourceGroupPager(c.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
		}
		for _, disk := range page.Value {
			if disk.Name == nil {
				continue
			}
			item, sdpErr := c.azureDiskToSDPItem(disk)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeDiskWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	pager := c.client.NewListByResourceGroupPager(c.ResourceGroup(), nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}
		for _, disk := range page.Value {
			if disk.Name == nil {
				continue
			}
			item, sdpErr := c.azureDiskToSDPItem(disk)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c computeDiskWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the disk name"), c.DefaultScope(), c.Type())
	}
	diskName := queryParts[0]
	disk, err := c.client.Get(ctx, c.ResourceGroup(), diskName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
	}
	return c.azureDiskToSDPItem(&disk.Disk)
}

func (c computeDiskWrapper) azureDiskToSDPItem(disk *armcompute.Disk) (*sdp.Item, *sdp.QueryError) {
	if disk.Name == nil {
		return nil, azureshared.QueryError(errors.New("name is nil"), c.DefaultScope(), c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(disk, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
	}
	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeDisk.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             c.DefaultScope(),
		Tags:              azureshared.ConvertAzureTags(disk.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to Virtual Machine from ManagedBy
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get
	if disk.ManagedBy != nil && *disk.ManagedBy != "" {
		vmName := azureshared.ExtractResourceName(*disk.ManagedBy)
		if vmName != "" {
			scope := c.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*disk.ManagedBy); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeVirtualMachine.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vmName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If VM is deleted/modified → disk becomes detached (In: true)
					Out: false, // If disk is deleted → VM remains but loses disk (Out: false)
				},
			})
		}
	}

	// Link to Virtual Machines from ManagedByExtended (for shared disks)
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get
	if disk.ManagedByExtended != nil {
		for _, vmID := range disk.ManagedByExtended {
			if vmID != nil && *vmID != "" {
				vmName := azureshared.ExtractResourceName(*vmID)
				if vmName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(*vmID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeVirtualMachine.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vmName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If VM is deleted/modified → disk becomes detached (In: true)
							Out: false, // If disk is deleted → VM remains but loses disk (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to Virtual Machines from ShareInfo
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get
	if disk.Properties != nil && disk.Properties.ShareInfo != nil {
		for _, shareInfo := range disk.Properties.ShareInfo {
			if shareInfo != nil && shareInfo.VMURI != nil && *shareInfo.VMURI != "" {
				vmName := azureshared.ExtractResourceName(*shareInfo.VMURI)
				if vmName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(*shareInfo.VMURI); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeVirtualMachine.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vmName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If VM is deleted/modified → disk becomes detached (In: true)
							Out: false, // If disk is deleted → VM remains but loses disk (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to Disk Access from Properties.DiskAccessID
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/diskaccesses/get
	if disk.Properties != nil && disk.Properties.DiskAccessID != nil && *disk.Properties.DiskAccessID != "" {
		diskAccessName := azureshared.ExtractResourceName(*disk.Properties.DiskAccessID)
		if diskAccessName != "" {
			scope := c.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*disk.Properties.DiskAccessID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeDiskAccess.String(),
					Method: sdp.QueryMethod_GET,
					Query:  diskAccessName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Disk Access is deleted/modified → disk private endpoint access is affected (In: true)
					Out: false, // If disk is deleted → Disk Access remains (Out: false)
				},
			})
		}
	}

	// Link to Disk Encryption Set from Properties.Encryption.DiskEncryptionSetID
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
	if disk.Properties != nil && disk.Properties.Encryption != nil && disk.Properties.Encryption.DiskEncryptionSetID != nil && *disk.Properties.Encryption.DiskEncryptionSetID != "" {
		encryptionSetName := azureshared.ExtractResourceName(*disk.Properties.Encryption.DiskEncryptionSetID)
		if encryptionSetName != "" {
			scope := c.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*disk.Properties.Encryption.DiskEncryptionSetID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeDiskEncryptionSet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  encryptionSetName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Disk Encryption Set is deleted/modified → disk encryption is affected (In: true)
					Out: false, // If disk is deleted → Disk Encryption Set remains (Out: false)
				},
			})
		}
	}

	// Link to Disk Encryption Set from Properties.SecurityProfile.SecureVMDiskEncryptionSetID
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
	if disk.Properties != nil && disk.Properties.SecurityProfile != nil && disk.Properties.SecurityProfile.SecureVMDiskEncryptionSetID != nil && *disk.Properties.SecurityProfile.SecureVMDiskEncryptionSetID != "" {
		encryptionSetName := azureshared.ExtractResourceName(*disk.Properties.SecurityProfile.SecureVMDiskEncryptionSetID)
		if encryptionSetName != "" {
			scope := c.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*disk.Properties.SecurityProfile.SecureVMDiskEncryptionSetID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeDiskEncryptionSet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  encryptionSetName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Disk Encryption Set is deleted/modified → disk encryption is affected (In: true)
					Out: false, // If disk is deleted → Disk Encryption Set remains (Out: false)
				},
			})
		}
	}

	// Link to source resources from Properties.CreationData
	if disk.Properties != nil && disk.Properties.CreationData != nil {
		creationData := disk.Properties.CreationData

		// Link to source Disk or Snapshot from SourceResourceID
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disks/get?view=rest-compute-2025-04-01&tabs=HTTP
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/snapshots/get?view=rest-compute-2025-04-01&tabs=HTTP
		if creationData.SourceResourceID != nil && *creationData.SourceResourceID != "" {
			sourceResourceID := *creationData.SourceResourceID
			// Determine if it's a disk or snapshot based on the resource type in the ID
			if strings.Contains(sourceResourceID, "/disks/") {
				diskName := azureshared.ExtractResourceName(sourceResourceID)
				if diskName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(sourceResourceID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeDisk.String(),
							Method: sdp.QueryMethod_GET,
							Query:  diskName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If source disk is deleted/modified → this disk may be affected (In: true)
							Out: false, // If this disk is deleted → source disk remains (Out: false)
						},
					})
				}
			} else if strings.Contains(sourceResourceID, "/snapshots/") {
				snapshotName := azureshared.ExtractResourceName(sourceResourceID)
				if snapshotName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(sourceResourceID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSnapshot.String(),
							Method: sdp.QueryMethod_GET,
							Query:  snapshotName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If source snapshot is deleted/modified → this disk may be affected (In: true)
							Out: false, // If this disk is deleted → source snapshot remains (Out: false)
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
				scope := c.DefaultScope()
				if extractedScope := azureshared.ExtractScopeFromResourceID(*creationData.StorageAccountID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.StorageAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  storageAccountName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Storage Account is deleted/modified → disk import may fail (In: true)
						Out: false, // If disk is deleted → Storage Account remains (Out: false)
					},
				})
			}
		}

		// Link to Image from ImageReference.ID
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/images/get
		if creationData.ImageReference != nil && creationData.ImageReference.ID != nil && *creationData.ImageReference.ID != "" {
			imageID := *creationData.ImageReference.ID
			// Check if it's a regular image or gallery image
			if strings.Contains(imageID, "/images/") && !strings.Contains(imageID, "/galleries/") {
				imageName := azureshared.ExtractResourceName(imageID)
				if imageName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(imageID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  imageName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Image is deleted/modified → disk created from image may be affected (In: true)
							Out: false, // If disk is deleted → Image remains (Out: false)
						},
					})
				}
			}
		}

		// Link to Gallery Image from GalleryImageReference
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/gallery-images/get
		if creationData.GalleryImageReference != nil {
			// Link from ID (shared gallery image)
			if creationData.GalleryImageReference.ID != nil && *creationData.GalleryImageReference.ID != "" {
				galleryImageID := *creationData.GalleryImageReference.ID
				// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/galleries/{galleryName}/images/{imageName}/versions/{version}
				parts := azureshared.ExtractPathParamsFromResourceID(galleryImageID, []string{"galleries", "images", "versions"})
				if len(parts) >= 3 {
					galleryName := parts[0]
					imageName := parts[1]
					version := parts[2]
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(galleryImageID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSharedGalleryImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(galleryName, imageName, version),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Gallery Image is deleted/modified → disk created from image may be affected (In: true)
							Out: false, // If disk is deleted → Gallery Image remains (Out: false)
						},
					})
				}
			}

			// Link from SharedGalleryImageID
			if creationData.GalleryImageReference.SharedGalleryImageID != nil && *creationData.GalleryImageReference.SharedGalleryImageID != "" {
				sharedGalleryImageID := *creationData.GalleryImageReference.SharedGalleryImageID
				// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/galleries/{galleryName}/images/{imageName}/versions/{version}
				parts := azureshared.ExtractPathParamsFromResourceID(sharedGalleryImageID, []string{"galleries", "images", "versions"})
				if len(parts) >= 3 {
					galleryName := parts[0]
					imageName := parts[1]
					version := parts[2]
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(sharedGalleryImageID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSharedGalleryImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(galleryName, imageName, version),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Gallery Image is deleted/modified → disk created from image may be affected (In: true)
							Out: false, // If disk is deleted → Gallery Image remains (Out: false)
						},
					})
				}
			}

			// Link from CommunityGalleryImageID
			if creationData.GalleryImageReference.CommunityGalleryImageID != nil && *creationData.GalleryImageReference.CommunityGalleryImageID != "" {
				communityGalleryImageID := *creationData.GalleryImageReference.CommunityGalleryImageID
				// Format: /CommunityGalleries/{communityGalleryName}/Images/{imageName}/Versions/{version}
				// Note: Community gallery images may not have subscription/resource group in the ID
				parts := azureshared.ExtractPathParamsFromResourceID(communityGalleryImageID, []string{"Images", "Versions"})
				if len(parts) >= 2 {
					imageName := parts[0]
					version := parts[1]
					// Extract community gallery name (before "Images")
					allParts := strings.Split(strings.Trim(communityGalleryImageID, "/"), "/")
					communityGalleryName := ""
					for i, part := range allParts {
						if part == "CommunityGalleries" && i+1 < len(allParts) {
							communityGalleryName = allParts[i+1]
							break
						}
					}
					if communityGalleryName != "" {
						scope := c.DefaultScope()
						if extractedScope := azureshared.ExtractScopeFromResourceID(communityGalleryImageID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeCommunityGalleryImage.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(communityGalleryName, imageName, version),
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Community Gallery Image is deleted/modified → disk created from image may be affected (In: true)
								Out: false, // If disk is deleted → Community Gallery Image remains (Out: false)
							},
						})
					}
				}
			}
		}

		// Link to Elastic SAN Volume Snapshot from ElasticSanResourceID
		// Reference: https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-snapshots/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/snapshots/{snapshotName}
		if creationData.ElasticSanResourceID != nil && *creationData.ElasticSanResourceID != "" {
			elasticSanResourceID := *creationData.ElasticSanResourceID
			// Elastic SAN snapshot IDs follow format:
			// /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/snapshots/{snapshotName}
			parts := azureshared.ExtractPathParamsFromResourceID(elasticSanResourceID, []string{"elasticSans", "volumegroups", "snapshots"})
			if len(parts) >= 3 {
				elasticSanName := parts[0]
				volumeGroupName := parts[1]
				snapshotName := parts[2]
				scope := c.DefaultScope()
				if extractedScope := azureshared.ExtractScopeFromResourceID(elasticSanResourceID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ElasticSanVolumeSnapshot.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(elasticSanName, volumeGroupName, snapshotName),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Elastic SAN snapshot is deleted/modified → disk created from snapshot may be affected (In: true)
						Out: false, // If disk is deleted → Elastic SAN snapshot remains (Out: false)
					},
				})
			}
		}
	}

	// Link to Key Vault resources from EncryptionSettingsCollection
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/vaults/get
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/get-secret
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keys/get-key
	if disk.Properties != nil && disk.Properties.EncryptionSettingsCollection != nil && disk.Properties.EncryptionSettingsCollection.EncryptionSettings != nil {
		for _, encryptionSetting := range disk.Properties.EncryptionSettingsCollection.EncryptionSettings {
			if encryptionSetting == nil {
				continue
			}

			// Link to Key Vault from DiskEncryptionKey.SourceVault.ID
			if encryptionSetting.DiskEncryptionKey != nil && encryptionSetting.DiskEncryptionKey.SourceVault != nil && encryptionSetting.DiskEncryptionKey.SourceVault.ID != nil && *encryptionSetting.DiskEncryptionKey.SourceVault.ID != "" {
				vaultName := azureshared.ExtractResourceName(*encryptionSetting.DiskEncryptionKey.SourceVault.ID)
				if vaultName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(*encryptionSetting.DiskEncryptionKey.SourceVault.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault is deleted/modified → disk encryption key access is affected (In: true)
							Out: false, // If disk is deleted → Key Vault remains (Out: false)
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
					// Key Vault URI doesn't contain resource group, use disk's scope as best effort
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultSecret.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(vaultName, secretName),
							Scope:  c.DefaultScope(), // Limitation: Key Vault URI doesn't contain resource group info
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault Secret is deleted/modified → disk encryption key is affected (In: true)
							Out: false, // If disk is deleted → Key Vault Secret remains (Out: false)
						},
					})
				}

				// Link to DNS name (standard library) from SecretURL
				dnsName := azureshared.ExtractDNSFromURL(secretURL)
				if dnsName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "dns",
							Method: sdp.QueryMethod_SEARCH,
							Query:  dnsName,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// DNS names are always linked
							In:  true,
							Out: true,
						},
					})
				}
			}

			// Link to Key Vault from KeyEncryptionKey.SourceVault.ID
			if encryptionSetting.KeyEncryptionKey != nil && encryptionSetting.KeyEncryptionKey.SourceVault != nil && encryptionSetting.KeyEncryptionKey.SourceVault.ID != nil && *encryptionSetting.KeyEncryptionKey.SourceVault.ID != "" {
				vaultName := azureshared.ExtractResourceName(*encryptionSetting.KeyEncryptionKey.SourceVault.ID)
				if vaultName != "" {
					scope := c.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(*encryptionSetting.KeyEncryptionKey.SourceVault.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault is deleted/modified → key encryption key access is affected (In: true)
							Out: false, // If disk is deleted → Key Vault remains (Out: false)
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
					// Key Vault URI doesn't contain resource group, use disk's scope as best effort
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultKey.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(vaultName, keyName),
							Scope:  c.DefaultScope(), // Limitation: Key Vault URI doesn't contain resource group info
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault Key is deleted/modified → key encryption key is affected (In: true)
							Out: false, // If disk is deleted → Key Vault Key remains (Out: false)
						},
					})
				}

				// Link to DNS name (standard library) from KeyURL
				dnsName := azureshared.ExtractDNSFromURL(keyURL)
				if dnsName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "dns",
							Method: sdp.QueryMethod_SEARCH,
							Query:  dnsName,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// DNS names are always linked
							In:  true,
							Out: true,
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (c computeDiskWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeDiskLookupByName,
	}
}

func (c computeDiskWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeVirtualMachine,
		azureshared.ComputeDisk,
		azureshared.ComputeSnapshot,
		azureshared.ComputeDiskAccess,
		azureshared.ComputeDiskEncryptionSet,
		azureshared.ComputeImage,
		azureshared.ComputeSharedGalleryImage,
		azureshared.ComputeCommunityGalleryImage,
		azureshared.StorageAccount,
		azureshared.ElasticSanVolumeSnapshot,
		azureshared.KeyVaultVault,
		azureshared.KeyVaultSecret,
		azureshared.KeyVaultKey,
		stdlib.NetworkDNS,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/managed_disk
func (c computeDiskWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_managed_disk.name",
		},
	}
}
