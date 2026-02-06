package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeImageLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeImage)

type computeImageWrapper struct {
	client clients.ImagesClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeImage(client clients.ImagesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListStreamableWrapper {
	return &computeImageWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeImage,
		),
	}
}

func (c computeImageWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, image := range page.Value {
			if image.Name == nil {
				continue
			}
			item, sdpErr := c.azureImageToSDPItem(image, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeImageWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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
		for _, image := range page.Value {
			if image.Name == nil {
				continue
			}
			item, sdpErr := c.azureImageToSDPItem(image, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c computeImageWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 1 and be the image name"), scope, c.Type())
	}
	imageName := queryParts[0]
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	image, err := c.client.Get(ctx, rgScope.ResourceGroup, imageName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureImageToSDPItem(&image.Image, scope)
}

func (c computeImageWrapper) azureImageToSDPItem(image *armcompute.Image, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(image, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeImage.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(image.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link resources from StorageProfile
	if image.Properties != nil && image.Properties.StorageProfile != nil {
		storageProfile := image.Properties.StorageProfile

		// Link to OS Disk resources
		if storageProfile.OSDisk != nil {
			osDisk := storageProfile.OSDisk

			// Link to Managed Disk from OSDisk.ManagedDisk.ID
			// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disks/get
			if osDisk.ManagedDisk != nil && osDisk.ManagedDisk.ID != nil && *osDisk.ManagedDisk.ID != "" {
				diskName := azureshared.ExtractResourceName(*osDisk.ManagedDisk.ID)
				if diskName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(*osDisk.ManagedDisk.ID)
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
							In:  true,  // If source disk is deleted/modified → image becomes invalid (In: true)
							Out: false, // If image is deleted → source disk remains (Out: false)
						},
					})
				}
			}

			// Link to Snapshot from OSDisk.Snapshot.ID
			// Reference: https://learn.microsoft.com/en-us/rest/api/compute/snapshots/get
			if osDisk.Snapshot != nil && osDisk.Snapshot.ID != nil && *osDisk.Snapshot.ID != "" {
				snapshotName := azureshared.ExtractResourceName(*osDisk.Snapshot.ID)
				if snapshotName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(*osDisk.Snapshot.ID)
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
							In:  true,  // If source snapshot is deleted/modified → image becomes invalid (In: true)
							Out: false, // If image is deleted → source snapshot remains (Out: false)
						},
					})
				}
			}

			// Link to Storage Account from OSDisk.BlobUri
			// Reference: https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/get-properties
			if osDisk.BlobURI != nil && *osDisk.BlobURI != "" {
				blobURI := *osDisk.BlobURI
				storageAccountName := azureshared.ExtractStorageAccountNameFromBlobURI(blobURI)
				if storageAccountName != "" {
					// For blob URIs, we use the current scope since storage accounts are typically in the same subscription
					// If the blob URI contains resource ID information, we could extract it, but blob URIs typically don't
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.StorageAccount.String(),
							Method: sdp.QueryMethod_GET,
							Query:  storageAccountName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Storage Account is deleted/modified → image blob becomes inaccessible (In: true)
							Out: false, // If image is deleted → Storage Account remains (Out: false)
						},
					})
				}

				// Link to stdlib.NetworkHTTP if blob URI is HTTP/HTTPS
				if strings.HasPrefix(blobURI, "http://") || strings.HasPrefix(blobURI, "https://") {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkHTTP.String(),
							Method: sdp.QueryMethod_SEARCH,
							Query:  blobURI,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // If HTTP endpoint is unavailable → image cannot be accessed (In: true)
							Out: true, // If image is deleted → HTTP endpoint may still be used by other resources (Out: true)
						},
					})
				}

				// Link to DNS name (standard library) from BlobURI
				dnsName := azureshared.ExtractDNSFromURL(blobURI)
				if dnsName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkDNS.String(),
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

			// Link to Disk Encryption Set from OSDisk.DiskEncryptionSet.ID
			// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
			if osDisk.DiskEncryptionSet != nil && osDisk.DiskEncryptionSet.ID != nil && *osDisk.DiskEncryptionSet.ID != "" {
				encryptionSetName := azureshared.ExtractResourceName(*osDisk.DiskEncryptionSet.ID)
				if encryptionSetName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(*osDisk.DiskEncryptionSet.ID)
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
							In:  true,  // If Disk Encryption Set is deleted/modified → image encryption is affected (In: true)
							Out: false, // If image is deleted → Disk Encryption Set remains (Out: false)
						},
					})
				}
			}
		}

		// Link to Data Disk resources
		if storageProfile.DataDisks != nil {
			for _, dataDisk := range storageProfile.DataDisks {
				if dataDisk == nil {
					continue
				}

				// Link to Managed Disk from DataDisk.ManagedDisk.ID
				// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disks/get
				if dataDisk.ManagedDisk != nil && dataDisk.ManagedDisk.ID != nil && *dataDisk.ManagedDisk.ID != "" {
					diskName := azureshared.ExtractResourceName(*dataDisk.ManagedDisk.ID)
					if diskName != "" {
						extractedScope := azureshared.ExtractScopeFromResourceID(*dataDisk.ManagedDisk.ID)
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
								In:  true,  // If source disk is deleted/modified → image becomes invalid (In: true)
								Out: false, // If image is deleted → source disk remains (Out: false)
							},
						})
					}
				}

				// Link to Snapshot from DataDisk.Snapshot.ID
				// Reference: https://learn.microsoft.com/en-us/rest/api/compute/snapshots/get
				if dataDisk.Snapshot != nil && dataDisk.Snapshot.ID != nil && *dataDisk.Snapshot.ID != "" {
					snapshotName := azureshared.ExtractResourceName(*dataDisk.Snapshot.ID)
					if snapshotName != "" {
						extractedScope := azureshared.ExtractScopeFromResourceID(*dataDisk.Snapshot.ID)
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
								In:  true,  // If source snapshot is deleted/modified → image becomes invalid (In: true)
								Out: false, // If image is deleted → source snapshot remains (Out: false)
							},
						})
					}
				}

				// Link to Storage Account from DataDisk.BlobUri
				// Reference: https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/get-properties
				if dataDisk.BlobURI != nil && *dataDisk.BlobURI != "" {
					blobURI := *dataDisk.BlobURI
					storageAccountName := azureshared.ExtractStorageAccountNameFromBlobURI(blobURI)
					if storageAccountName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.StorageAccount.String(),
								Method: sdp.QueryMethod_GET,
								Query:  storageAccountName,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Storage Account is deleted/modified → image blob becomes inaccessible (In: true)
								Out: false, // If image is deleted → Storage Account remains (Out: false)
							},
						})
					}

					// Link to stdlib.NetworkHTTP if blob URI is HTTP/HTTPS
					if strings.HasPrefix(blobURI, "http://") || strings.HasPrefix(blobURI, "https://") {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkHTTP.String(),
								Method: sdp.QueryMethod_SEARCH,
								Query:  blobURI,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // If HTTP endpoint is unavailable → image cannot be accessed (In: true)
								Out: true, // If image is deleted → HTTP endpoint may still be used by other resources (Out: true)
							},
						})
					}

					// Link to DNS name (standard library) from BlobURI
					dnsName := azureshared.ExtractDNSFromURL(blobURI)
					if dnsName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkDNS.String(),
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

				// Link to Disk Encryption Set from DataDisk.DiskEncryptionSet.ID
				// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
				if dataDisk.DiskEncryptionSet != nil && dataDisk.DiskEncryptionSet.ID != nil && *dataDisk.DiskEncryptionSet.ID != "" {
					encryptionSetName := azureshared.ExtractResourceName(*dataDisk.DiskEncryptionSet.ID)
					if encryptionSetName != "" {
						extractedScope := azureshared.ExtractScopeFromResourceID(*dataDisk.DiskEncryptionSet.ID)
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
								In:  true,  // If Disk Encryption Set is deleted/modified → image encryption is affected (In: true)
								Out: false, // If image is deleted → Disk Encryption Set remains (Out: false)
							},
						})
					}
				}
			}
		}
	}

	// Link to Source Virtual Machine from Properties.SourceVirtualMachine.ID
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get
	if image.Properties != nil && image.Properties.SourceVirtualMachine != nil && image.Properties.SourceVirtualMachine.ID != nil && *image.Properties.SourceVirtualMachine.ID != "" {
		vmName := azureshared.ExtractResourceName(*image.Properties.SourceVirtualMachine.ID)
		if vmName != "" {
			extractedScope := azureshared.ExtractScopeFromResourceID(*image.Properties.SourceVirtualMachine.ID)
			if extractedScope == "" {
				extractedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeVirtualMachine.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vmName,
					Scope:  extractedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If source VM is deleted/modified → image source becomes invalid (In: true)
					Out: false, // If image is deleted → source VM remains (Out: false)
				},
			})
		}
	}

	return sdpItem, nil
}

func (c computeImageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeImageLookupByName,
	}
}

func (c computeImageWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeDisk,
		azureshared.ComputeSnapshot,
		azureshared.ComputeDiskEncryptionSet,
		azureshared.ComputeVirtualMachine,
		azureshared.StorageAccount,
		stdlib.NetworkHTTP,
		stdlib.NetworkDNS,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/image
func (c computeImageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_image.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (c computeImageWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/images/read",
	}
}

func (c computeImageWrapper) PredefinedRole() string {
	return "Reader"
}
