package manual

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeInstanceLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeInstance)
var ComputeInstanceLookupByNetworkTag = shared.NewItemTypeLookup("networkTag", gcpshared.ComputeInstance)

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
		gcpshared.ComputeInstanceTemplate,
		gcpshared.ComputeRegionInstanceTemplate,
		gcpshared.ComputeInstanceGroupManager,
		gcpshared.ComputeFirewall,
		gcpshared.ComputeRoute,
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

// SupportsWildcardScope implements the WildcardScopeAdapter interface
// Always returns true for compute instances since they use aggregatedList
func (c computeInstanceWrapper) SupportsWildcardScope() bool {
	return true
}

func (c computeInstanceWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{ComputeInstanceLookupByNetworkTag},
	}
}

// Search finds compute instances by network tag. The engine routes
// project-scoped SEARCH queries to zonal scopes via substring matching, so
// scope is a zonal scope like "project.zone". We list all instances via
// AggregatedList and filter to the matching zone + tag.
func (c computeInstanceWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	tag := queryParts[0]

	allItems, qErr := c.List(ctx, "*")
	if qErr != nil {
		return nil, qErr
	}

	var matched []*sdp.Item
	for _, item := range allItems {
		if item.GetScope() != scope {
			continue
		}

		tagsVal, err := item.GetAttributes().Get("tags")
		if err != nil {
			continue
		}
		tagsMap, ok := tagsVal.(map[string]any)
		if !ok {
			continue
		}
		itemsVal, ok := tagsMap["items"]
		if !ok {
			continue
		}
		itemsList, ok := itemsVal.([]any)
		if !ok {
			continue
		}
		for _, t := range itemsList {
			if s, ok := t.(string); ok && s == tag {
				matched = append(matched, item)
				break
			}
		}
	}

	return matched, nil
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
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.ListStream(ctx, stream, cache, cacheKey, scope)
	})
}

func (c computeInstanceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	// Handle wildcard scope with AggregatedList
	if scope == "*" {
		c.listAggregatedStream(ctx, stream, cache, cacheKey)
		return
	}

	// Handle specific scope with per-zone List
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

	itemsSent := 0
	var hadError bool
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
			hadError = true
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
		itemsSent++
	}

	if itemsSent == 0 && !hadError {
		notFoundErr := &sdp.QueryError{
			ErrorType:     sdp.QueryError_NOTFOUND,
			ErrorString:   "no compute instances found in scope " + scope,
			Scope:         scope,
			SourceName:    c.Name(),
			ItemType:      c.Type(),
			ResponderName: c.Name(),
		}
		cache.StoreUnavailableItem(ctx, notFoundErr, shared.DefaultCacheDuration, cacheKey)
	}
}

// listAggregatedStream uses AggregatedList to stream all instances across all zones
func (c computeInstanceWrapper) listAggregatedStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	// Get all unique project IDs
	projectIDs := gcpshared.GetProjectIDsFromLocations(c.Locations())

	var itemsSent atomic.Int32
	var hadError atomic.Bool
	// Use a pool with 10x concurrency to parallelize AggregatedList calls
	p := pool.New().WithMaxGoroutines(10).WithContext(ctx)

	for _, projectID := range projectIDs {
		p.Go(func(ctx context.Context) error {
			it := c.client.AggregatedList(ctx, &computepb.AggregatedListInstancesRequest{
				Project:              projectID,
				ReturnPartialSuccess: new(true), // Handle partial failures gracefully
			})

			for {
				pair, iterErr := it.Next()
				if errors.Is(iterErr, iterator.Done) {
					break
				}
				if iterErr != nil {
					stream.SendError(gcpshared.QueryError(iterErr, projectID, c.Type()))
					hadError.Store(true)
					return iterErr
				}

				// Parse scope from pair.Key (e.g., "zones/us-central1-a")
				scopeLocation, err := gcpshared.ParseAggregatedListScope(projectID, pair.Key)
				if err != nil {
					continue // Skip unparseable scopes
				}

				// Only process if this scope is in our adapter's configured locations
				if !gcpshared.HasLocationInSlices(scopeLocation, c.Locations()) {
					continue
				}

				// Process instances in this scope
				if pair.Value != nil && pair.Value.GetInstances() != nil {
					for _, instance := range pair.Value.GetInstances() {
						item, sdpErr := c.gcpComputeInstanceToSDPItem(ctx, instance, scopeLocation)
						if sdpErr != nil {
							stream.SendError(sdpErr)
							hadError.Store(true)
							continue
						}

						cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
						stream.SendItem(item)
						itemsSent.Add(1)
					}
				}
			}

			return nil
		})
	}

	// Wait for all goroutines to complete
	_ = p.Wait()

	if itemsSent.Load() == 0 && !hadError.Load() {
		notFoundErr := &sdp.QueryError{
			ErrorType:     sdp.QueryError_NOTFOUND,
			ErrorString:   "no compute instances found in scope *",
			Scope:         "*",
			SourceName:    c.Name(),
			ItemType:      c.Type(),
			ResponderName: c.Name(),
		}
		cache.StoreUnavailableItem(ctx, notFoundErr, shared.DefaultCacheDuration, cacheKey)
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
						// Use SEARCH for all image references - it handles both family and specific image formats
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeImage.String(),
								Method: sdp.QueryMethod_SEARCH,
								Query:  sourceImage, // Pass full URI so Search can detect format
								Scope:  scope,
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
						})
					} else {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.CloudKMSCryptoKey.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(loc, keyRing, cryptoKey),
								Scope:  location.ProjectID,
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
			})
		}
	}

	// Link to instance template and instance group manager from metadata
	if metadata := instance.GetMetadata(); metadata != nil {
		for _, item := range metadata.GetItems() {
			key := item.GetKey()
			value := item.GetValue()

			switch key {
			case "instance-template":
				// Link to instance template (global or regional)
				if value != "" {
					templateName := gcpshared.LastPathComponent(value)
					scope, err := gcpshared.ExtractScopeFromURI(ctx, value)
					if err == nil && templateName != "" {
						templateType := gcpshared.ComputeInstanceTemplate
						if strings.Contains(value, "/regions/") {
							templateType = gcpshared.ComputeRegionInstanceTemplate
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   templateType.String(),
								Method: sdp.QueryMethod_GET,
								Query:  templateName,
								Scope:  scope,
							},
						})
					}
				}
			case "created-by":
				// Link to instance group manager (zonal or regional)
				if value != "" {
					igmName := gcpshared.LastPathComponent(value)
					scope, err := gcpshared.ExtractScopeFromURI(ctx, value)
					if err == nil && igmName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeInstanceGroupManager.String(),
								Method: sdp.QueryMethod_GET,
								Query:  igmName,
								Scope:  scope,
							},
						})
					}
				}
			}
		}
	}

	// Link to firewalls and routes by network tag.
	// Tag-based SEARCH lists all firewalls/routes in scope then filters;
	// may be slow in very large projects.
	if tags := instance.GetTags(); tags != nil {
		for _, tag := range tags.GetItems() {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries,
				&sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeFirewall.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  tag,
						Scope:  location.ProjectID,
					},
				},
				&sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.ComputeRoute.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  tag,
						Scope:  location.ProjectID,
					},
				},
			)
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
