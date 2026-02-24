package manual

import (
	"context"
	"errors"
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

var (
	ComputeGalleryApplicationVersionLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeGalleryApplicationVersion)
	ComputeGalleryApplicationLookupByName        = shared.NewItemTypeLookup("name", azureshared.ComputeGalleryApplication) //todo: move to its adapter file when created, this is just a placeholder
)

type computeGalleryApplicationVersionWrapper struct {
	client clients.GalleryApplicationVersionsClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeGalleryApplicationVersion(client clients.GalleryApplicationVersionsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &computeGalleryApplicationVersionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeGalleryApplicationVersion,
		),
	}
}

func (c computeGalleryApplicationVersionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 3 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 3 and be the gallery name, gallery application name, and gallery application version name"), scope, c.Type())
	}
	galleryName := queryParts[0]
	if galleryName == "" {
		return nil, azureshared.QueryError(errors.New("gallery name cannot be empty"), scope, c.Type())
	}
	galleryApplicationName := queryParts[1]
	if galleryApplicationName == "" {
		return nil, azureshared.QueryError(errors.New("gallery application name cannot be empty"), scope, c.Type())
	}
	galleryApplicationVersionName := queryParts[2]
	if galleryApplicationVersionName == "" {
		return nil, azureshared.QueryError(errors.New("gallery application version name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, galleryName, galleryApplicationName, galleryApplicationVersionName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureGalleryApplicationVersionToSDPItem(&resp.GalleryApplicationVersion, galleryName, galleryApplicationName, scope)
}

func (c computeGalleryApplicationVersionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 2 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 2 and be the gallery name and gallery application name"), scope, c.Type())
	}
	galleryName := queryParts[0]
	if galleryName == "" {
		return nil, azureshared.QueryError(errors.New("gallery name cannot be empty"), scope, c.Type())
	}
	galleryApplicationName := queryParts[1]
	if galleryApplicationName == "" {
		return nil, azureshared.QueryError(errors.New("gallery application name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByGalleryApplicationPager(rgScope.ResourceGroup, galleryName, galleryApplicationName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, galleryApplicationVersion := range page.Value {
			if galleryApplicationVersion == nil || galleryApplicationVersion.Name == nil {
				continue
			}
			item, sdpErr := c.azureGalleryApplicationVersionToSDPItem(galleryApplicationVersion, galleryName, galleryApplicationName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeGalleryApplicationVersionWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) != 2 {
		stream.SendError(azureshared.QueryError(errors.New("queryParts must be exactly 2 and be the gallery name and gallery application name"), scope, c.Type()))
		return
	}
	galleryName := queryParts[0]
	if galleryName == "" {
		stream.SendError(azureshared.QueryError(errors.New("gallery name cannot be empty"), scope, c.Type()))
		return
	}
	galleryApplicationName := queryParts[1]
	if galleryApplicationName == "" {
		stream.SendError(azureshared.QueryError(errors.New("gallery application name cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListByGalleryApplicationPager(rgScope.ResourceGroup, galleryName, galleryApplicationName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, galleryApplicationVersion := range page.Value {
			if galleryApplicationVersion == nil || galleryApplicationVersion.Name == nil {
				continue
			}
			item, sdpErr := c.azureGalleryApplicationVersionToSDPItem(galleryApplicationVersion, galleryName, galleryApplicationName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c computeGalleryApplicationVersionWrapper) azureGalleryApplicationVersionToSDPItem(
	galleryApplicationVersion *armcompute.GalleryApplicationVersion,
	galleryName,
	galleryApplicationName,
	scope string,
) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(galleryApplicationVersion, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	if galleryApplicationVersion.Name == nil {
		return nil, azureshared.QueryError(errors.New("gallery application version name is nil"), scope, c.Type())
	}
	galleryApplicationVersionName := *galleryApplicationVersion.Name
	if galleryApplicationVersionName == "" {
		return nil, azureshared.QueryError(errors.New("gallery application version name cannot be empty"), scope, c.Type())
	}
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(galleryName, galleryApplicationName, galleryApplicationVersionName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)

	// Parent Gallery: version depends on gallery; deleting version does not delete gallery
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeGallery.String(),
			Method: sdp.QueryMethod_GET,
			Query:  galleryName,
			Scope:  scope,
		},
	})

	// Parent Gallery Application: version depends on application; deleting version does not delete application
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeGalleryApplication.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(galleryName, galleryApplicationName),
			Scope:  scope,
		},
	})

	// MediaLink and DefaultConfigurationLink: add stdlib.NetworkHTTP, stdlib.NetworkDNS (hostname), stdlib.NetworkIP (when host is IP), azureshared.StorageAccount and azureshared.StorageBlobContainer (when Azure Blob) links.
	// Dedupe DNS by hostname, IP by address, StorageAccount by account name, and StorageBlobContainer by (account, container) so the same resource is not linked twice.
	linkedDNSHostnames := make(map[string]struct{})
	seenIPs := make(map[string]struct{})
	seenStorageAccounts := make(map[string]struct{})
	seenBlobContainers := make(map[string]struct{})
	if galleryApplicationVersion.Properties != nil && galleryApplicationVersion.Properties.PublishingProfile != nil && galleryApplicationVersion.Properties.PublishingProfile.Source != nil {
		src := galleryApplicationVersion.Properties.PublishingProfile.Source
		addBlobLinks := func(link string) {
			if link == "" || (!strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://")) {
				return
			}
			AppendURILinks(&linkedItemQueries, link, linkedDNSHostnames, seenIPs)
			if accountName := azureshared.ExtractStorageAccountNameFromBlobURI(link); accountName != "" {
				if _, seen := seenStorageAccounts[accountName]; !seen {
					seenStorageAccounts[accountName] = struct{}{}
					linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.StorageAccount.String(),
							Method: sdp.QueryMethod_GET,
							Query:  accountName,
							Scope:  scope,
						},
					})
				}
				containerName := azureshared.ExtractContainerNameFromBlobURI(link)
				if containerName != "" {
					containerKey := shared.CompositeLookupKey(accountName, containerName)
					if _, seen := seenBlobContainers[containerKey]; !seen {
						seenBlobContainers[containerKey] = struct{}{}
						linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.StorageBlobContainer.String(),
								Method: sdp.QueryMethod_GET,
								Query:  containerKey,
								Scope:  scope,
							},
						})
					}
				}
			}
		}
		if src.MediaLink != nil && *src.MediaLink != "" {
			addBlobLinks(*src.MediaLink)
		}
		if src.DefaultConfigurationLink != nil && *src.DefaultConfigurationLink != "" {
			defaultConfigLink := *src.DefaultConfigurationLink
			if strings.HasPrefix(defaultConfigLink, "http://") || strings.HasPrefix(defaultConfigLink, "https://") {
				sameAsMedia := src.MediaLink != nil && *src.MediaLink == defaultConfigLink
				if !sameAsMedia {
					addBlobLinks(defaultConfigLink)
				}
			}
		}
	}

	// Disk encryption sets from TargetRegions[].Encryption (OS and data disk); dedupe by ID
	seenEncryptionSetIDs := make(map[string]struct{})
	if galleryApplicationVersion.Properties != nil && galleryApplicationVersion.Properties.PublishingProfile != nil && galleryApplicationVersion.Properties.PublishingProfile.TargetRegions != nil {
		for _, tr := range galleryApplicationVersion.Properties.PublishingProfile.TargetRegions {
			if tr == nil || tr.Encryption == nil {
				continue
			}
			if tr.Encryption.OSDiskImage != nil && tr.Encryption.OSDiskImage.DiskEncryptionSetID != nil && *tr.Encryption.OSDiskImage.DiskEncryptionSetID != "" {
				id := *tr.Encryption.OSDiskImage.DiskEncryptionSetID
				if _, seen := seenEncryptionSetIDs[id]; !seen {
					seenEncryptionSetIDs[id] = struct{}{}
					name := azureshared.ExtractResourceName(id)
					if name != "" {
						linkScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(id); extractedScope != "" {
							linkScope = extractedScope
						}
						linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeDiskEncryptionSet.String(),
								Method: sdp.QueryMethod_GET,
								Query:  name,
								Scope:  linkScope,
							},
						})
					}
				}
			}
			if tr.Encryption.OSDiskImage != nil && tr.Encryption.OSDiskImage.SecurityProfile != nil && tr.Encryption.OSDiskImage.SecurityProfile.SecureVMDiskEncryptionSetID != nil && *tr.Encryption.OSDiskImage.SecurityProfile.SecureVMDiskEncryptionSetID != "" {
				id := *tr.Encryption.OSDiskImage.SecurityProfile.SecureVMDiskEncryptionSetID
				if _, seen := seenEncryptionSetIDs[id]; !seen {
					seenEncryptionSetIDs[id] = struct{}{}
					name := azureshared.ExtractResourceName(id)
					if name != "" {
						linkScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(id); extractedScope != "" {
							linkScope = extractedScope
						}
						linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeDiskEncryptionSet.String(),
								Method: sdp.QueryMethod_GET,
								Query:  name,
								Scope:  linkScope,
							},
						})
					}
				}
			}
			if tr.Encryption.DataDiskImages != nil {
				for _, ddi := range tr.Encryption.DataDiskImages {
					if ddi != nil && ddi.DiskEncryptionSetID != nil && *ddi.DiskEncryptionSetID != "" {
						id := *ddi.DiskEncryptionSetID
						if _, seen := seenEncryptionSetIDs[id]; !seen {
							seenEncryptionSetIDs[id] = struct{}{}
							name := azureshared.ExtractResourceName(id)
							if name != "" {
								linkScope := scope
								if extractedScope := azureshared.ExtractScopeFromResourceID(id); extractedScope != "" {
									linkScope = extractedScope
								}
								linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
									Query: &sdp.Query{
										Type:   azureshared.ComputeDiskEncryptionSet.String(),
										Method: sdp.QueryMethod_GET,
										Query:  name,
										Scope:  linkScope,
									},
								})
							}
						}
					}
				}
			}
		}
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeGalleryApplicationVersion.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(galleryApplicationVersion.Tags),
		LinkedItemQueries: linkedItemQueries,
	}
	return sdpItem, nil
}

func (c computeGalleryApplicationVersionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeGalleryLookupByName,
		ComputeGalleryApplicationLookupByName,
		ComputeGalleryApplicationVersionLookupByName,
	}
}

func (c computeGalleryApplicationVersionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeGalleryLookupByName,
			ComputeGalleryApplicationLookupByName,
		},
	}
}

func (c computeGalleryApplicationVersionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeGallery,
		azureshared.ComputeGalleryApplication,
		azureshared.ComputeDiskEncryptionSet,
		azureshared.StorageAccount,
		azureshared.StorageBlobContainer,
		stdlib.NetworkDNS,
		stdlib.NetworkHTTP,
		stdlib.NetworkIP,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/gallery_application_version
func (c computeGalleryApplicationVersionWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			//example id: /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Compute/galleries/gallery1/applications/galleryApplication1/versions/galleryApplicationVersion1
			TerraformQueryMap: "azurerm_gallery_application_version.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (c computeGalleryApplicationVersionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/galleries/applications/versions/read",
	}
}

func (c computeGalleryApplicationVersionWrapper) PredefinedRole() string {
	return "Reader"
}
