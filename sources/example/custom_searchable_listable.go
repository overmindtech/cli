package example

import (
	"context"
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/shared"
)

type customComputeInstanceWrapper struct {
	client ExternalAPIClient

	*shared.Base
}

// NewCustomSearchableListable creates a new customComputeInstanceWrapper instance
func NewCustomSearchableListable(client ExternalAPIClient, projectID, zone string) sources.SearchableListableWrapper {
	return &customComputeInstanceWrapper{
		client: client,
		Base: shared.NewBase(
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			ComputeInstance,
			[]string{projectID, fmt.Sprintf("%s.%s", projectID, zone)}, // example custom scopes
		),
	}
}

// AdapterMetadata returns the adapter metadata for the ExternalType.
// This method allows providing custom metadata for the adapter.
func (d *customComputeInstanceWrapper) AdapterMetadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            ComputeInstance.String(),
		Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		PotentialLinks:  []string{ComputeDisk.String()},
		DescriptiveName: "Custom descriptive name",
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get:               true,
			GetDescription:    "Get a compute instance by ID",
			List:              true,
			ListDescription:   "List all compute instances",
			Search:            true,
			SearchDescription: "Search for compute instances by {compute status id} or {compute disk name|compute status id}",
		},
		TerraformMappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "example_resource.name",
			},
		},
	}
}

// PotentialLinks returns the potential links for the ExternalType.
// This should include all the item types that are added as linked items in the externalTypeToSDPItem method
func (d *customComputeInstanceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(ComputeDisk)
}

// GetLookups returns the sources.ItemTypeLookups for the Get operation
// This is used for input validation and constructing the human readable get query description.
func (d *customComputeInstanceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstanceLookupByID,
	}
}

// Get retrieves a specific ExternalType by unique attribute and converts it to a sdp.Item
func (d *customComputeInstanceWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	external, err := d.client.Get(ctx, queryParts[0])
	if err != nil {
		return nil, queryError(err)
	}

	return d.externalTypeToSDPItem(external)
}

// List retrieves all ExternalType and converts them to sdp.Items
func (d *customComputeInstanceWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	externals, err := d.client.List(ctx)
	if err != nil {
		return nil, queryError(err)
	}

	return d.mapper(externals)
}

// SearchLookups returns the ItemTypeLookups for the Search operation
// This is used for input validation and constructing the human-readable search query description.
// An item can be searched via multiple lookups.
// Each variant should be added as a separate sources.ItemTypeLookups
// In this example, we have two lookups:
// 1. Simple Key: ComputeDiskLookupByName: searching this item type by compute disk name
// 2. Composite Key: ComputeDiskLookupByName|ComputeStatusLookupByID: searching this item type by
// compute disk name and compute status ID
func (d *customComputeInstanceWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeStatusLookupByID,
		},
		{
			ComputeDiskLookupByName,
			ComputeStatusLookupByID,
		},
	}
}

// Search retrieves ExternalType by a search query and converts them to sdp.Items
func (d *customComputeInstanceWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	var err error
	var externals []*ExternalType
	switch len(queryParts) {
	case 1:
		externals, err = d.client.Search(ctx, queryParts[0])
	case 2:
		externals, err = d.client.Search(ctx, queryParts[0], queryParts[1])
	}
	if err != nil {
		return nil, queryError(err)
	}

	// We don't need to check if the length of the query is different from 1 or 2.
	// This is validated in the backend when converting this to an adapter.

	return d.mapper(externals)
}

// externalTypeToSDPItem converts an ExternalType to a sdp.Item
// This is where we define the linked items and how they are linked.
// All the linked items should be added to the PotentialLinks method!
func (d *customComputeInstanceWrapper) externalTypeToSDPItem(external *ExternalType) (*sdp.Item, *sdp.QueryError) {
	sdpItem := &sdp.Item{
		Type:            external.Type,
		UniqueAttribute: external.UniqueAttribute,
		Tags:            external.Tags,
		LinkedItemQueries: []*sdp.LinkedItemQuery{
			{
				Query: &sdp.Query{
					Type:   ComputeDisk.String(),
					Method: sdp.QueryMethod_GET,
					Query:  external.LinkedItemID,
					Scope:  d.Scopes()[0],
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			},
		},
	}

	return sdpItem, nil
}

// mapper converts a slice of ExternalType to a slice of sdp.Item
func (d *customComputeInstanceWrapper) mapper(externalItems []*ExternalType) ([]*sdp.Item, *sdp.QueryError) {
	sdpItems := make([]*sdp.Item, len(externalItems))
	for i, item := range externalItems {
		var err *sdp.QueryError
		sdpItems[i], err = d.externalTypeToSDPItem(item)
		if err != nil {
			return nil, err
		}
	}

	return sdpItems, nil
}
