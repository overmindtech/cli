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

var ComputeInstanceLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeInstance)

type computeInstanceWrapper struct {
	client gcpshared.ComputeInstanceClient
	*gcpshared.ZoneBase
}

// NewComputeInstance creates a new computeInstanceWrapper instance.
func NewComputeInstance(client gcpshared.ComputeInstanceClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeInstanceWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeInstance,
		),
	}
}

func (c computeInstanceWrapper) IAMPermissions() []string {
	return []string{
		"compute.instances.get",
		"compute.instances.list",
	}
}

func (c computeInstanceWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeInstanceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		stdlib.NetworkIP,
		stdlib.NetworkDNS,
		gcpshared.ComputeDisk,
		gcpshared.ComputeSubnetwork,
		gcpshared.ComputeNetwork,
		gcpshared.ComputeResourcePolicy,
		gcpshared.IAMServiceAccount,
		gcpshared.ComputeImage,
		gcpshared.ComputeSnapshot,
		gcpshared.CloudKMSCryptoKey,
		gcpshared.CloudKMSCryptoKeyVersion,
		gcpshared.ComputeZone,
	)
}

func (c computeInstanceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance#argument-reference
			TerraformQueryMap: "google_compute_instance.name",
		},
	}
}

func (c computeInstanceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstanceLookupByName,
	}
}

func (c computeInstanceWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetInstanceRequest{
		Project:  location.ProjectID,
		Zone:     location.Zone,
		Instance: queryParts[0],
	}

	instance, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeInstanceToSDPItem(ctx, instance, location)
}

func (c computeInstanceWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListInstancesRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	var items []*sdp.Item
	for {
		instance, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeInstanceToSDPItem(ctx, instance, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeInstanceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListInstancesRequest{
		Project: location.ProjectID,
		Zone:    location.Zone,
	})

	for {
		instance, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeInstanceToSDPItem(ctx, instance, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeInstanceWrapper) gcpComputeInstanceToSDPItem(ctx context.Context, instance *computepb.Instance, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(instance, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeInstance.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            instance.GetLabels(),
	}

	for _, disk := range instance.GetDisks() {
		if disk.GetSource() != "" {
			if strings.Contains(disk.GetSource(), "/") {
				diskNameParts := strings.Split(disk.GetSource(), "/")
				diskName := diskNameParts[len(diskNameParts)-1]
				scope, err := gcpshared.ExtractScopeFromURI(ctx, disk.GetSource())
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
							Out: true,
						},
					})
				}
			}
		}

		// Link to source image if disk is being initialized from an image
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

			// Link to source snapshot if disk is being initialized from a snapshot
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

			// Link to KMS key used to decrypt source image
			if sourceImageEncryptionKey := initializeParams.GetSourceImageEncryptionKey(); sourceImageEncryptionKey != nil {
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
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: false,
							},
						})
					}
				}
			}

			// Link to KMS key used to decrypt source snapshot
			if sourceSnapshotEncryptionKey := initializeParams.GetSourceSnapshotEncryptionKey(); sourceSnapshotEncryptionKey != nil {
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
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: false,
							},
						})
					}
				}
			}
		}

		// Link to KMS key used for disk encryption
		if diskEncryptionKey := disk.GetDiskEncryptionKey(); diskEncryptionKey != nil {
			if keyName := diskEncryptionKey.GetKmsKeyName(); keyName != "" {
				loc := gcpshared.ExtractPathParam("locations", keyName)
				keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
				cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
				cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

				if loc != "" && keyRing != "" && cryptoKey != "" {
					if cryptoKeyVersion != "" {
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
					} else {
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
			}
		}
	}

	if instance.GetNetworkInterfaces() != nil {
		for _, networkInterface := range instance.GetNetworkInterfaces() {
			if networkInterface.GetNetworkIP() != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  networkInterface.GetNetworkIP(),
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}

			if networkInterface.GetIpv6Address() != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  networkInterface.GetIpv6Address(),
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}

			// Link to external IPv4 address from access configs
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
				if externalIPv6 := accessConfig.GetExternalIpv6(); externalIPv6 != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  externalIPv6,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					})
				}
			}

			// Link to external IPv6 address from ipv6AccessConfigs
			for _, ipv6AccessConfig := range networkInterface.GetIpv6AccessConfigs() {
				if externalIPv6 := ipv6AccessConfig.GetExternalIpv6(); externalIPv6 != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  externalIPv6,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					})
				}
			}

			if subnetwork := networkInterface.GetSubnetwork(); subnetwork != "" {
				if strings.Contains(subnetwork, "/") {
					subnetworkName := gcpshared.LastPathComponent(subnetwork)
					region := gcpshared.ExtractPathParam("regions", subnetwork)
					if region != "" && subnetworkName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeSubnetwork.String(),
								Method: sdp.QueryMethod_GET,
								Query:  subnetworkName,
								Scope:  gcpshared.RegionalScope(location.ProjectID, region),
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,
								Out: false,
							},
						})
					}
				}
			}

			if network := networkInterface.GetNetwork(); network != "" {
				if strings.Contains(network, "/") {
					networkName := gcpshared.LastPathComponent(network)
					if networkName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeNetwork.String(),
								Method: sdp.QueryMethod_GET,
								Query:  networkName,
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
		}
	}

	// Link to resource policies
	for _, rp := range instance.GetResourcePolicies() {
		if strings.Contains(rp, "/") {
			parts := gcpshared.ExtractPathParams(rp, "regions", "resourcePolicies")
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				resourcePolicyName := parts[1]
				region := parts[0]
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeResourcePolicy.String(),
						Method: sdp.QueryMethod_GET,
						Query:  resourcePolicyName,
						Scope:  gcpshared.RegionalScope(location.ProjectID, region),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Link to service account
	for _, sa := range instance.GetServiceAccounts() {
		if email := sa.GetEmail(); email != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.IAMServiceAccount.String(),
					Method: sdp.QueryMethod_GET,
					Query:  email,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to zone
	if zoneURL := instance.GetZone(); zoneURL != "" {
		zoneName := gcpshared.LastPathComponent(zoneURL)
		if zoneName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeZone.String(),
					Method: sdp.QueryMethod_GET,
					Query:  zoneName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Set health based on status
	switch instance.GetStatus() {
	case computepb.Instance_RUNNING.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	case computepb.Instance_STOPPING.String(),
		computepb.Instance_SUSPENDING.String(),
		computepb.Instance_PROVISIONING.String(),
		computepb.Instance_STAGING.String(),
		computepb.Instance_REPAIRING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Instance_TERMINATED.String(),
		computepb.Instance_STOPPED.String(),
		computepb.Instance_SUSPENDED.String():
		// No health set for stopped/terminated instances
	}

	return sdpItem, nil
}
