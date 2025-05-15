package adapters

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var (
	ComputeAddress = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.Address)

	ComputeAddressLookupByName = shared.NewItemTypeLookup("name", ComputeAddress)
)

type computeAddressWrapper struct {
	client gcpshared.ComputeAddressClient

	*gcpshared.RegionBase
}

// NewComputeAddress creates a new computeAddressWrapper address
func NewComputeAddress(client gcpshared.ComputeAddressClient, projectID, region string) sources.ListableWrapper {
	return &computeAddressWrapper{
		client: client,
		RegionBase: gcpshared.NewRegionBase(
			projectID,
			region,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			ComputeAddress,
		),
	}
}

// PotentialLinks returns the potential links for the compute address wrapper
func (c computeAddressWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		stdlib.NetworkIP,
		ComputeAddress,
		ComputeSubnetwork,
		ComputeNetwork,
	)
}

// TerraformMappings returns the Terraform mappings for the compute address wrapper
func (c computeAddressWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_address.name",
		},
	}
}

// GetLookups returns the lookups for the compute address wrapper
// This defines how the source can be queried for specific item
// In this case, it will be: gcp-compute-engine-address-name
func (c computeAddressWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeAddressLookupByName,
	}
}

// Get retrieves a compute address by its name
func (c computeAddressWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetAddressRequest{
		Project: c.ProjectID(),
		Region:  c.Region(),
		Address: queryParts[0],
	}

	address, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeAddressToSDPItem(address)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute addresss and converts them to sdp.Items.
func (c computeAddressWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListAddressesRequest{
		Project: c.ProjectID(),
		Region:  c.Region(),
	})

	var items []*sdp.Item
	for {
		address, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeAddressToSDPItem(address)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeAddressWrapper) gcpComputeAddressToSDPItem(address *computepb.Address) (*sdp.Item, *sdp.QueryError) {
	// Convert the address to attributes
	attributes, err := shared.ToAttributesWithExclude(address, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            ComputeAddress.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.Scopes()[0],
		Tags:            address.GetLabels(),
	}

	if network := address.GetNetwork(); network != "" {
		if strings.Contains(network, "/") {
			networkNameParts := strings.Split(network, "/")
			networkName := networkNameParts[len(networkNameParts)-1]
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  networkName,
					Scope:  c.ProjectID(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	if subnetwork := address.GetSubnetwork(); subnetwork != "" {
		if strings.Contains(subnetwork, "/") {
			subnetworkNameParts := strings.Split(subnetwork, "/")
			subnetworkName := subnetworkNameParts[len(subnetworkNameParts)-1]
			region := gcpshared.ExtractRegion(subnetwork)
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   ComputeSubnetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  subnetworkName,
					Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	if ip := address.GetAddress(); ip != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkIP.String(),
				Method: sdp.QueryMethod_GET,
				Query:  ip,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	switch address.GetStatus() {
	case computepb.Address_RESERVING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Address_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.Address_RESERVED.String(),
		computepb.Address_IN_USE.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	}

	return sdpItem, nil
}
