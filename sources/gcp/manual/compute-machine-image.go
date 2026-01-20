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
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeMachineImageLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeMachineImage)

type computeMachineImageWrapper struct {
	client gcpshared.ComputeMachineImageClient
	*gcpshared.ProjectBase
}

// NewComputeMachineImage creates a new computeMachineImageWrapper instance.
func NewComputeMachineImage(client gcpshared.ComputeMachineImageClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeMachineImageWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeMachineImage,
		),
	}
}

func (c computeMachineImageWrapper) IAMPermissions() []string {
	return []string{
		"compute.machineImages.get",
		"compute.machineImages.list",
	}
}

func (c computeMachineImageWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeMachineImageWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeNetwork,
		gcpshared.ComputeSubnetwork,
		gcpshared.ComputeNetworkAttachment,
		gcpshared.ComputeDisk,
		gcpshared.ComputeImage,
		gcpshared.ComputeSnapshot,
		gcpshared.CloudKMSCryptoKeyVersion,
		gcpshared.ComputeInstance,
		gcpshared.IAMServiceAccount,
		gcpshared.ComputeAcceleratorType,
		stdlib.NetworkIP,
	)
}

func (c computeMachineImageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_machine_image.name",
		},
	}
}

func (c computeMachineImageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeMachineImageLookupByName,
	}
}

func (c computeMachineImageWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetMachineImageRequest{
		Project:      location.ProjectID,
		MachineImage: queryParts[0],
	}

	machineImage, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeMachineImageToSDPItem(ctx, machineImage, location)
}

func (c computeMachineImageWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListMachineImagesRequest{
		Project: location.ProjectID,
	})

	var items []*sdp.Item
	for {
		machineImage, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeMachineImageToSDPItem(ctx, machineImage, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeMachineImageWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListMachineImagesRequest{
		Project: location.ProjectID,
	})

	for {
		machineImage, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeMachineImageToSDPItem(ctx, machineImage, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeMachineImageWrapper) gcpComputeMachineImageToSDPItem(ctx context.Context, machineImage *computepb.MachineImage, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(machineImage, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeMachineImage.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            machineImage.GetLabels(),
	}

	if instanceProperties := machineImage.GetInstanceProperties(); instanceProperties != nil {
		for _, networkInterface := range instanceProperties.GetNetworkInterfaces() {
			if network := networkInterface.GetNetwork(); network != "" {
				networkName := gcpshared.LastPathComponent(network)
				if networkName != "" {
					scope, err := gcpshared.ExtractScopeFromURI(ctx, network)
					if err == nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeNetwork.String(),
								Method: sdp.QueryMethod_GET,
								Query:  networkName,
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

			if subnet := networkInterface.GetSubnetwork(); subnet != "" {
				subnetworkName := gcpshared.LastPathComponent(subnet)
				if subnetworkName != "" {
					scope, err := gcpshared.ExtractScopeFromURI(ctx, subnet)
					if err == nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeSubnetwork.String(),
								Method: sdp.QueryMethod_GET,
								Query:  subnetworkName,
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

			if networkAttachment := networkInterface.GetNetworkAttachment(); networkAttachment != "" {
				networkAttachmentName := gcpshared.LastPathComponent(networkAttachment)
				if networkAttachmentName != "" {
					scope, err := gcpshared.ExtractScopeFromURI(ctx, networkAttachment)
					if err == nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeNetworkAttachment.String(),
								Method: sdp.QueryMethod_GET,
								Query:  networkAttachmentName,
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

			if networkIP := networkInterface.GetNetworkIP(); networkIP != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  networkIP,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}

			if ipv6Address := networkInterface.GetIpv6Address(); ipv6Address != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  ipv6Address,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}

			for _, accessConfig := range networkInterface.GetAccessConfigs() {
				if natIP := accessConfig.GetNatIP(); natIP != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  natIP,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					})
				}
			}

			for _, ipv6AccessConfig := range networkInterface.GetIpv6AccessConfigs() {
				if externalIpv6 := ipv6AccessConfig.GetExternalIpv6(); externalIpv6 != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  externalIpv6,
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

		for _, disk := range instanceProperties.GetDisks() {
			if diskSource := disk.GetSource(); diskSource != "" {
				if strings.Contains(diskSource, "/") {
					diskName := gcpshared.LastPathComponent(diskSource)
					if diskName != "" {
						scope, err := gcpshared.ExtractScopeFromURI(ctx, diskSource)
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

							if sourceDiskEncryptionKey := disk.GetDiskEncryptionKey(); sourceDiskEncryptionKey != nil {
								c.addKMSKeyLink(sdpItem, sourceDiskEncryptionKey.GetKmsKeyName(), location)
							}
						}
					}
				}
			}

			if initializeParams := disk.GetInitializeParams(); initializeParams != nil {
				if sourceImage := initializeParams.GetSourceImage(); sourceImage != "" {
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

				if sourceSnapshot := initializeParams.GetSourceSnapshot(); sourceSnapshot != "" {
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

				if sourceImageEncryptionKey := initializeParams.GetSourceImageEncryptionKey(); sourceImageEncryptionKey != nil {
					c.addKMSKeyLink(sdpItem, sourceImageEncryptionKey.GetKmsKeyName(), location)
				}

				if sourceSnapshotEncryptionKey := initializeParams.GetSourceSnapshotEncryptionKey(); sourceSnapshotEncryptionKey != nil {
					c.addKMSKeyLink(sdpItem, sourceSnapshotEncryptionKey.GetKmsKeyName(), location)
				}
			}
		}

		for _, serviceAccount := range instanceProperties.GetServiceAccounts() {
			if email := serviceAccount.GetEmail(); email != "" {
				saEmail := email
				if strings.Contains(email, "/") {
					saEmail = gcpshared.LastPathComponent(email)
				}
				if saEmail != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.IAMServiceAccount.String(),
							Method: sdp.QueryMethod_GET,
							Query:  saEmail,
							Scope:  location.ProjectID,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					})
				}
			}
		}

		for _, accelerator := range instanceProperties.GetGuestAccelerators() {
			if acceleratorType := accelerator.GetAcceleratorType(); acceleratorType != "" {
				acceleratorTypeName := gcpshared.LastPathComponent(acceleratorType)
				if acceleratorTypeName != "" {
					scope, err := gcpshared.ExtractScopeFromURI(ctx, acceleratorType)
					if err == nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeAcceleratorType.String(),
								Method: sdp.QueryMethod_GET,
								Query:  acceleratorTypeName,
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

	if machineImageEncryptionKey := machineImage.GetMachineImageEncryptionKey(); machineImageEncryptionKey != nil {
		c.addKMSKeyLink(sdpItem, machineImageEncryptionKey.GetKmsKeyName(), location)
	}

	if sourceInstance := machineImage.GetSourceInstance(); sourceInstance != "" {
		sourceInstanceName := gcpshared.LastPathComponent(sourceInstance)
		if sourceInstanceName != "" {
			scope, err := gcpshared.ExtractScopeFromURI(ctx, sourceInstance)
			if err == nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeInstance.String(),
						Method: sdp.QueryMethod_GET,
						Query:  sourceInstanceName,
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

	for _, savedDisk := range machineImage.GetSavedDisks() {
		if sourceDisk := savedDisk.GetSourceDisk(); sourceDisk != "" {
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
	}

	switch machineImage.GetStatus() {
	case computepb.MachineImage_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	case computepb.MachineImage_CREATING.String(),
		computepb.MachineImage_DELETING.String(),
		computepb.MachineImage_UPLOADING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.MachineImage_INVALID.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	}

	return sdpItem, nil
}

func (c computeMachineImageWrapper) addKMSKeyLink(sdpItem *sdp.Item, keyName string, location gcpshared.LocationInfo) {
	if keyName == "" {
		return
	}
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
	}
}
