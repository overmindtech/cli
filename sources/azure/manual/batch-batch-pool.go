package manual

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v3"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var BatchBatchPoolLookupByName = shared.NewItemTypeLookup("name", azureshared.BatchBatchPool)

type batchBatchPoolWrapper struct {
	client clients.BatchPoolsClient
	*azureshared.MultiResourceGroupBase
}

// NewBatchBatchPool returns a SearchableWrapper for Azure Batch pools (child of Batch account).
func NewBatchBatchPool(client clients.BatchPoolsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &batchBatchPoolWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.BatchBatchPool,
		),
	}
}

func (b batchBatchPoolWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: accountName and poolName",
			Scope:       scope,
			ItemType:    b.Type(),
		}
	}
	accountName := queryParts[0]
	poolName := queryParts[1]

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}
	resp, err := b.client.Get(ctx, rgScope.ResourceGroup, accountName, poolName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	return b.azurePoolToSDPItem(&resp.Pool, accountName, poolName, scope)
}

func (b batchBatchPoolWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BatchAccountLookupByName,
		BatchBatchPoolLookupByName,
	}
}

func (b batchBatchPoolWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: accountName",
			Scope:       scope,
			ItemType:    b.Type(),
		}
	}
	accountName := queryParts[0]

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}
	pager := b.client.ListByBatchAccount(ctx, rgScope.ResourceGroup, accountName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, b.Type())
		}

		for _, pool := range page.Value {
			if pool == nil || pool.Name == nil {
				continue
			}
			item, sdpErr := b.azurePoolToSDPItem(pool, accountName, *pool.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (b batchBatchPoolWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: accountName"), scope, b.Type()))
		return
	}
	accountName := queryParts[0]

	rgScope, err := b.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, b.Type()))
		return
	}
	pager := b.client.ListByBatchAccount(ctx, rgScope.ResourceGroup, accountName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, b.Type()))
			return
		}
		for _, pool := range page.Value {
			if pool == nil || pool.Name == nil {
				continue
			}
			item, sdpErr := b.azurePoolToSDPItem(pool, accountName, *pool.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (b batchBatchPoolWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			BatchAccountLookupByName,
		},
	}
}

func (b batchBatchPoolWrapper) azurePoolToSDPItem(pool *armbatch.Pool, accountName, poolName, scope string) (*sdp.Item, *sdp.QueryError) {
	if pool.Name == nil {
		return nil, azureshared.QueryError(errors.New("pool name is nil"), scope, b.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(pool, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	if err := attributes.Set("uniqueAttr", shared.CompositeLookupKey(accountName, poolName)); err != nil {
		return nil, azureshared.QueryError(err, scope, b.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.BatchBatchPool.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(pool.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to parent Batch Account
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchAccount.String(),
			Method: sdp.QueryMethod_GET,
			Query:  accountName,
			Scope:  scope,
		},
	})

	// Link to public IPs when NetworkConfiguration.PublicIPAddressConfiguration.IPAddressIDs is set
	if pool.Properties != nil && pool.Properties.NetworkConfiguration != nil && pool.Properties.NetworkConfiguration.PublicIPAddressConfiguration != nil {
		for _, ipIDPtr := range pool.Properties.NetworkConfiguration.PublicIPAddressConfiguration.IPAddressIDs {
			if ipIDPtr == nil || *ipIDPtr == "" {
				continue
			}
			ipName := azureshared.ExtractResourceName(*ipIDPtr)
			if ipName == "" {
				continue
			}
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(*ipIDPtr); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkPublicIPAddress.String(),
					Method: sdp.QueryMethod_GET,
					Query:  ipName,
					Scope:  linkedScope,
				},
			})
		}
	}

	// Link to Subnet when NetworkConfiguration.SubnetID is set
	if pool.Properties != nil && pool.Properties.NetworkConfiguration != nil && pool.Properties.NetworkConfiguration.SubnetID != nil {
		subnetID := *pool.Properties.NetworkConfiguration.SubnetID
		scopeParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"subscriptions", "resourceGroups"})
		subnetParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
		if len(scopeParams) >= 2 && len(subnetParams) >= 2 {
			subnetScope := fmt.Sprintf("%s.%s", scopeParams[0], scopeParams[1])
			vnetName := subnetParams[0]
			subnetName := subnetParams[1]
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkSubnet.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(vnetName, subnetName),
					Scope:  subnetScope,
				},
			})
		}
	}

	// Link to user-assigned managed identities from Identity.UserAssignedIdentities map keys (resource IDs)
	if pool.Identity != nil && pool.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range pool.Identity.UserAssignedIdentities {
			if identityResourceID == "" {
				continue
			}
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName == "" {
				continue
			}
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					Method: sdp.QueryMethod_GET,
					Query:  identityName,
					Scope:  linkedScope,
				},
			})
		}
	}

	// Link to application packages referenced by the pool (Properties.ApplicationPackages)
	// ID can be .../batchAccounts/{account}/applications/{app}/versions/{version} (specific version)
	// or .../applications/{app} (default version); when default, use pkgRef.Version as fallback.
	if pool.Properties != nil && pool.Properties.ApplicationPackages != nil {
		for _, pkgRef := range pool.Properties.ApplicationPackages {
			if pkgRef == nil || pkgRef.ID == nil || *pkgRef.ID == "" {
				continue
			}
			var pkgAccountName, appName, version string
			params := azureshared.ExtractPathParamsFromResourceID(*pkgRef.ID, []string{"batchAccounts", "applications", "versions"})
			if len(params) >= 3 {
				pkgAccountName, appName, version = params[0], params[1], params[2]
			} else {
				paramsApp := azureshared.ExtractPathParamsFromResourceID(*pkgRef.ID, []string{"batchAccounts", "applications"})
				if len(paramsApp) < 2 {
					continue
				}
				pkgAccountName, appName = paramsApp[0], paramsApp[1]
				if pkgRef.Version != nil && *pkgRef.Version != "" {
					version = *pkgRef.Version
				} else {
					// Default version reference with no Version field: cannot form GET (adapter needs account|app|version)
					continue
				}
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.BatchBatchApplicationPackage.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(pkgAccountName, appName, version),
					Scope:  scope,
				},
			})
		}
	}

	// Link to certificates referenced by the pool (Properties.Certificates)
	// ID format: .../batchAccounts/{account}/certificates/{thumbprint}
	if pool.Properties != nil && pool.Properties.Certificates != nil {
		for _, certRef := range pool.Properties.Certificates {
			if certRef == nil || certRef.ID == nil || *certRef.ID == "" {
				continue
			}
			params := azureshared.ExtractPathParamsFromResourceID(*certRef.ID, []string{"batchAccounts", "certificates"})
			if len(params) < 2 {
				continue
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.BatchBatchCertificate.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(params[0], params[1]),
					Scope:  scope,
				},
			})
		}
	}

	// Link to storage accounts and IP/DNS from MountConfiguration
	seenIPs := make(map[string]struct{})
	seenDNS := make(map[string]struct{})
	if pool.Properties != nil && pool.Properties.MountConfiguration != nil {
		for _, mount := range pool.Properties.MountConfiguration {
			if mount == nil {
				continue
			}
			if mount.AzureBlobFileSystemConfiguration != nil {
				blobCfg := mount.AzureBlobFileSystemConfiguration
				if blobCfg.AccountName != nil && *blobCfg.AccountName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.StorageAccount.String(),
							Method: sdp.QueryMethod_GET,
							Query:  *blobCfg.AccountName,
							Scope:  scope,
						},
					})
				}
				if blobCfg.AccountName != nil && *blobCfg.AccountName != "" && blobCfg.ContainerName != nil && *blobCfg.ContainerName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.StorageBlobContainer.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(*blobCfg.AccountName, *blobCfg.ContainerName),
							Scope:  scope,
						},
					})
				}
				if blobCfg.IdentityReference != nil && blobCfg.IdentityReference.ResourceID != nil && *blobCfg.IdentityReference.ResourceID != "" {
					identityResourceID := *blobCfg.IdentityReference.ResourceID
					identityName := azureshared.ExtractResourceName(identityResourceID)
					if identityName != "" {
						linkedScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
							linkedScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
								Method: sdp.QueryMethod_GET,
								Query:  identityName,
								Scope:  linkedScope,
							},
						})
					}
				}
			}
			if mount.AzureFileShareConfiguration != nil {
				if mount.AzureFileShareConfiguration.AccountName != nil && *mount.AzureFileShareConfiguration.AccountName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.StorageAccount.String(),
							Method: sdp.QueryMethod_GET,
							Query:  *mount.AzureFileShareConfiguration.AccountName,
							Scope:  scope,
						},
					})
				}
				if mount.AzureFileShareConfiguration.AzureFileURL != nil && *mount.AzureFileShareConfiguration.AzureFileURL != "" {
					AppendURILinks(&sdpItem.LinkedItemQueries, *mount.AzureFileShareConfiguration.AzureFileURL, seenDNS, seenIPs)
				}
			}
			if mount.CifsMountConfiguration != nil && mount.CifsMountConfiguration.Source != nil && *mount.CifsMountConfiguration.Source != "" {
				appendMountSourceHostLink(&sdpItem.LinkedItemQueries, *mount.CifsMountConfiguration.Source, seenIPs, seenDNS)
			}
			if mount.NfsMountConfiguration != nil && mount.NfsMountConfiguration.Source != nil && *mount.NfsMountConfiguration.Source != "" {
				appendMountSourceHostLink(&sdpItem.LinkedItemQueries, *mount.NfsMountConfiguration.Source, seenIPs, seenDNS)
			}
		}
	}

	// Link to image reference from DeploymentConfiguration.VirtualMachineConfiguration.ImageReference
	// (custom image, shared gallery image, or community gallery image)
	if pool.Properties != nil && pool.Properties.DeploymentConfiguration != nil &&
		pool.Properties.DeploymentConfiguration.VirtualMachineConfiguration != nil {
		imageRef := pool.Properties.DeploymentConfiguration.VirtualMachineConfiguration.ImageReference
		if imageRef != nil {
			// ImageReference.ID: custom image or gallery image version path
			if imageRef.ID != nil && *imageRef.ID != "" {
				imageID := *imageRef.ID
				if strings.Contains(imageID, "/galleries/") && strings.Contains(imageID, "/images/") && strings.Contains(imageID, "/versions/") {
					params := azureshared.ExtractPathParamsFromResourceID(imageID, []string{"galleries", "images", "versions"})
					if len(params) == 3 {
						galleryName, imageName, versionName := params[0], params[1], params[2]
						linkScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(imageID); extractedScope != "" {
							linkScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeSharedGalleryImage.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(galleryName, imageName, versionName),
								Scope:  linkScope,
							},
						})
					}
				} else if strings.Contains(imageID, "/images/") {
					imageName := azureshared.ExtractResourceName(imageID)
					if imageName != "" {
						linkScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(imageID); extractedScope != "" {
							linkScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeImage.String(),
								Method: sdp.QueryMethod_GET,
								Query:  imageName,
								Scope:  linkScope,
							},
						})
					}
				}
			}
			// SharedGalleryImageID (path: .../sharedGalleries/{name}/images/{name}/versions/{name})
			if imageRef.SharedGalleryImageID != nil && *imageRef.SharedGalleryImageID != "" {
				sharedGalleryImageID := *imageRef.SharedGalleryImageID
				parts := azureshared.ExtractPathParamsFromResourceID(sharedGalleryImageID, []string{"sharedGalleries", "images", "versions"})
				if len(parts) >= 3 {
					galleryName, imageName, version := parts[0], parts[1], parts[2]
					linkScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(sharedGalleryImageID); extractedScope != "" {
						linkScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeSharedGalleryImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(galleryName, imageName, version),
							Scope:  linkScope,
						},
					})
				}
			}
			// CommunityGalleryImageID
			if imageRef.CommunityGalleryImageID != nil && *imageRef.CommunityGalleryImageID != "" {
				communityGalleryImageID := *imageRef.CommunityGalleryImageID
				parts := azureshared.ExtractPathParamsFromResourceID(communityGalleryImageID, []string{"CommunityGalleries", "Images", "Versions"})
				if len(parts) >= 3 {
					communityGalleryName, imageName, version := parts[0], parts[1], parts[2]
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeCommunityGalleryImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(communityGalleryName, imageName, version),
							Scope:  scope,
						},
					})
				}
			}
		}
		// Container registries (RegistryServer → DNS link; IdentityReference → managed identity link)
		vmConfig := pool.Properties.DeploymentConfiguration.VirtualMachineConfiguration
		if vmConfig.ContainerConfiguration != nil && vmConfig.ContainerConfiguration.ContainerRegistries != nil {
			for _, reg := range vmConfig.ContainerConfiguration.ContainerRegistries {
				if reg == nil {
					continue
				}
				if reg.RegistryServer != nil && *reg.RegistryServer != "" {
					host := strings.TrimSpace(*reg.RegistryServer)
					if host != "" {
						if net.ParseIP(host) != nil {
							if _, seen := seenIPs[host]; !seen {
								seenIPs[host] = struct{}{}
								sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
									Query: &sdp.Query{
										Type:   stdlib.NetworkIP.String(),
										Method: sdp.QueryMethod_GET,
										Query:  host,
										Scope:  "global",
									},
								})
							}
						} else {
							if _, seen := seenDNS[host]; !seen {
								seenDNS[host] = struct{}{}
								sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
									Query: &sdp.Query{
										Type:   stdlib.NetworkDNS.String(),
										Method: sdp.QueryMethod_SEARCH,
										Query:  host,
										Scope:  "global",
									},
								})
							}
						}
					}
				}
				if reg.IdentityReference != nil && reg.IdentityReference.ResourceID != nil && *reg.IdentityReference.ResourceID != "" {
					identityResourceID := *reg.IdentityReference.ResourceID
					identityName := azureshared.ExtractResourceName(identityResourceID)
					if identityName != "" {
						linkedScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
							linkedScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
								Method: sdp.QueryMethod_GET,
								Query:  identityName,
								Scope:  linkedScope,
							},
						})
					}
				}
			}
		}
	}

	// StartTask: ResourceFiles (HTTPUrl, StorageContainerURL → URI links; IdentityReference → managed identity), ContainerSettings.Registry (RegistryServer → DNS; IdentityReference → managed identity)
	if pool.Properties != nil && pool.Properties.StartTask != nil {
		startTask := pool.Properties.StartTask
		if startTask.ResourceFiles != nil {
			for _, rf := range startTask.ResourceFiles {
				if rf == nil {
					continue
				}
				if rf.HTTPURL != nil && *rf.HTTPURL != "" {
					AppendURILinks(&sdpItem.LinkedItemQueries, *rf.HTTPURL, seenDNS, seenIPs)
				}
				if rf.StorageContainerURL != nil && *rf.StorageContainerURL != "" {
					AppendURILinks(&sdpItem.LinkedItemQueries, *rf.StorageContainerURL, seenDNS, seenIPs)
				}
				if rf.IdentityReference != nil && rf.IdentityReference.ResourceID != nil && *rf.IdentityReference.ResourceID != "" {
					identityResourceID := *rf.IdentityReference.ResourceID
					identityName := azureshared.ExtractResourceName(identityResourceID)
					if identityName != "" {
						linkedScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
							linkedScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
								Method: sdp.QueryMethod_GET,
								Query:  identityName,
								Scope:  linkedScope,
							},
						})
					}
				}
			}
		}
		if startTask.ContainerSettings != nil && startTask.ContainerSettings.Registry != nil {
			reg := startTask.ContainerSettings.Registry
			if reg.RegistryServer != nil && *reg.RegistryServer != "" {
				host := strings.TrimSpace(*reg.RegistryServer)
				if host != "" {
					if net.ParseIP(host) != nil {
						if _, seen := seenIPs[host]; !seen {
							seenIPs[host] = struct{}{}
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   stdlib.NetworkIP.String(),
									Method: sdp.QueryMethod_GET,
									Query:  host,
									Scope:  "global",
								},
							})
						}
					} else {
						if _, seen := seenDNS[host]; !seen {
							seenDNS[host] = struct{}{}
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   stdlib.NetworkDNS.String(),
									Method: sdp.QueryMethod_SEARCH,
									Query:  host,
									Scope:  "global",
								},
							})
						}
					}
				}
			}
			if reg.IdentityReference != nil && reg.IdentityReference.ResourceID != nil && *reg.IdentityReference.ResourceID != "" {
				identityResourceID := *reg.IdentityReference.ResourceID
				identityName := azureshared.ExtractResourceName(identityResourceID)
				if identityName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
							Method: sdp.QueryMethod_GET,
							Query:  identityName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Map provisioning state to health
	if pool.Properties != nil && pool.Properties.ProvisioningState != nil {
		switch *pool.Properties.ProvisioningState {
		case armbatch.PoolProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armbatch.PoolProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return sdpItem, nil
}

func (b batchBatchPoolWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.BatchBatchAccount:                   true,
		azureshared.NetworkSubnet:                       true,
		azureshared.NetworkPublicIPAddress:              true,
		azureshared.ManagedIdentityUserAssignedIdentity: true,
		azureshared.BatchBatchApplicationPackage:        true,
		azureshared.BatchBatchCertificate:               true,
		azureshared.StorageAccount:                      true,
		azureshared.StorageBlobContainer:                true,
		azureshared.ComputeImage:                        true,
		azureshared.ComputeSharedGalleryImage:           true,
		azureshared.ComputeCommunityGalleryImage:        true,
		stdlib.NetworkIP:                                true,
		stdlib.NetworkDNS:                               true,
		stdlib.NetworkHTTP:                              true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/batch_pool
func (b batchBatchPoolWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_batch_pool.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute
func (b batchBatchPoolWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Batch/batchAccounts/pools/read",
	}
}

func (b batchBatchPoolWrapper) PredefinedRole() string {
	return "Azure Batch Account Reader"
}

// appendMountSourceHostLink extracts a host from a CIFS or NFS mount source (e.g. "\\server\share", "nfs://host/path", or "192.168.1.1") and appends a NetworkIP or NetworkDNS linked query with deduplication.
func appendMountSourceHostLink(queries *[]*sdp.LinkedItemQuery, source string, seenIPs, seenDNS map[string]struct{}) {
	if source == "" {
		return
	}
	var host string
	if after, ok := strings.CutPrefix(source, "\\\\"); ok {
		// UNC path: \\server\share
		rest := after
		if before, _, ok := strings.Cut(rest, "\\"); ok {
			host = before
		} else {
			host = rest
		}
	} else if strings.Contains(source, "://") {
		u, err := url.Parse(source)
		if err != nil || u.Host == "" {
			return
		}
		host = u.Hostname()
	} else {
		// NFS format: host:/path (e.g. 192.168.1.1:/vol1) — split on ":/" so host has no trailing colon
		if before, _, ok0 := strings.Cut(source, ":/"); ok0 {
			host = before
		} else if idx := strings.IndexAny(source, "/\\"); idx >= 0 {
			host = source[:idx]
		} else {
			host = source
		}
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return
	}
	if net.ParseIP(host) != nil {
		if _, seen := seenIPs[host]; !seen {
			seenIPs[host] = struct{}{}
			*queries = append(*queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkIP.String(),
					Method: sdp.QueryMethod_GET,
					Query:  host,
					Scope:  "global",
				},
			})
		}
	} else {
		if _, seen := seenDNS[host]; !seen {
			seenDNS[host] = struct{}{}
			*queries = append(*queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  host,
					Scope:  "global",
				},
			})
		}
	}
}
