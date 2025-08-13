package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
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

// NewComputeInstance creates a new computeInstanceWrapper instance
func NewComputeInstance(client gcpshared.ComputeInstanceClient, projectID, zone string) sources.ListableWrapper {
	return &computeInstanceWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			zone,
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

// PotentialLinks returns the potential links for the compute instance wrapper
func (c computeInstanceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		stdlib.NetworkIP,
		gcpshared.ComputeDisk,
		gcpshared.ComputeSubnetwork,
		gcpshared.ComputeNetwork,
	)
}

// TerraformMappings returns the Terraform mappings for the compute instance wrapper
func (c computeInstanceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance#argument-reference
			TerraformQueryMap: "google_compute_instance.name",
		},
	}
}

// GetLookups returns the lookups for the compute instance wrapper
// This defines how the source can be queried for specific item
// In this case, it will be: gcp-compute-engine-instance-name
func (c computeInstanceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstanceLookupByName,
	}
}

// Get retrieves a compute instance by its name
func (c computeInstanceWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetInstanceRequest{
		Project:  c.ProjectID(),
		Zone:     c.Zone(),
		Instance: queryParts[0],
	}

	instance, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeInstanceToSDPItem(instance)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute instances and converts them to sdp.Items.
func (c computeInstanceWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListInstancesRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	var items []*sdp.Item
	for {
		instance, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeInstanceToSDPItem(instance)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeInstanceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream) {
	it := c.client.List(ctx, &computepb.ListInstancesRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	for {
		instance, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err))
			return
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeInstanceToSDPItem(instance)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		stream.SendItem(item)
	}
}

func (c computeInstanceWrapper) gcpComputeInstanceToSDPItem(instance *computepb.Instance) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           c.DefaultScope(),
		Tags:            instance.GetLabels(),
	}

	for _, disk := range instance.GetDisks() {
		if disk.GetSource() != "" {
			// Specifies a valid partial or full URL to an existing Persistent Disk resource.
			// "source": "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-c/disks/integration-test-instance"
			// last part is the disk name
			if strings.Contains(disk.GetSource(), "/") {
				diskNameParts := strings.Split(disk.GetSource(), "/")
				diskName := diskNameParts[len(diskNameParts)-1]
				zone := gcpshared.ExtractPathParam("zones", disk.GetSource())
				if zone != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeDisk.String(),
							Method: sdp.QueryMethod_GET,
							Query:  diskName,
							Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
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

			if subnetwork := networkInterface.GetSubnetwork(); subnetwork != "" {
				// The URL of the Subnetwork resource for this instance.
				// If the network resource is in legacy mode, do not specify this field.
				// If the network is in auto subnet mode, specifying the subnetwork is optional.
				// If the network is in custom subnet mode, specifying the subnetwork is required.
				// If you specify this field, you can specify the subnetwork as a full or partial URL. For example,
				// the following are all valid URLs:
				//	- https://www.googleapis.com/compute/v1/projects/project/regions/region/subnetworks/subnetwork
				//	- regions/region/subnetworks/subnetwork
				// "subnetwork": "https://www.googleapis.com/compute/v1/projects/project-test/regions/us-central1/subnetworks/default"
				// last part is the subnetwork name
				if strings.Contains(subnetwork, "/") {
					subnetworkNameParts := strings.Split(subnetwork, "/")
					subnetworkName := subnetworkNameParts[len(subnetworkNameParts)-1]
					region := gcpshared.ExtractPathParam("regions", subnetwork)
					if region != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   gcpshared.ComputeSubnetwork.String(),
								Method: sdp.QueryMethod_GET,
								Query:  subnetworkName,
								// This is a regional resource
								Scope: gcpshared.RegionalScope(c.ProjectID(), region),
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
				// URL of the VPC network resource for this instance.
				// When creating an instance, if neither the network nor the subnetwork is specified,
				// the default network global/networks/default is used.
				// If the selected project doesn't have the default network, you must specify a network or subnet.
				// If the network is not specified but the subnetwork is specified, the network is inferred.
				// If you specify this property, you can specify the network as a full or partial URL.
				// For example, the following are all valid URLs:
				//	- https://www.googleapis.com/compute/v1/projects/project/global/networks/network
				//	- projects/project/global/networks/network
				//	- global/networks/default
				//
				// "network": "https://www.googleapis.com/compute/v1/projects/project-test/global/networks/default"
				if strings.Contains(network, "/") {
					networkNameParts := strings.Split(network, "/")
					networkName := networkNameParts[len(networkNameParts)-1]
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.ComputeNetwork.String(),
							Method: sdp.QueryMethod_GET,
							Query:  networkName,
							// This is a global resource
							Scope: c.ProjectID(),
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

	// The status of the instance.
	//	For more information about the status of the instance, see Instance life cycle.
	// Check the Status enum for the list of possible values.
	// https://cloud.google.com/compute/docs/instances/instance-lifecycle
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
	}

	return sdpItem, nil
}
