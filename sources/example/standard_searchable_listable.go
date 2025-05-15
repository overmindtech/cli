package example

import (
	"context"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	exampleshared "github.com/overmindtech/cli/sources/example/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeInstance = shared.NewItemType(exampleshared.Source, exampleshared.Compute, exampleshared.Instance)
	ComputeDisk     = shared.NewItemType(exampleshared.Source, exampleshared.Compute, exampleshared.Disk)
	ComputeStatus   = shared.NewItemType(exampleshared.Source, exampleshared.Compute, exampleshared.Status)

	ComputeInstanceLookupByID = shared.NewItemTypeLookup("id", ComputeInstance)
	ComputeStatusLookupByID   = shared.NewItemTypeLookup("id", ComputeStatus)
	ComputeDiskLookupByName   = shared.NewItemTypeLookup("name", ComputeDisk)
)

// ExternalType is a placeholder for the external API type
// For example, this could be a struct that represents a compute instance from GCP
type ExternalType struct {
	Type            string
	UniqueAttribute string
	Tags            map[string]string
	LinkedItemID    string
}

// NotFoundError is a placeholder for external API error codes
type NotFoundError struct{}

func (e NotFoundError) Error() string {
	return "not found"
}

// ExternalAPIClient is an interface for the external API client
//
//go:generate mockgen -destination=./mocks/mock_external_api_client.go -package=mocks -source=standard_searchable_listable.go ExternalAPIClient
type ExternalAPIClient interface {
	Get(ctx context.Context, query string) (*ExternalType, error)
	List(ctx context.Context) ([]*ExternalType, error)
	Search(ctx context.Context, query ...string) ([]*ExternalType, error)
}

type computeInstanceWrapper struct {
	client ExternalAPIClient

	*Base
}

// NewStandardSearchableListable creates a new computeInstanceWrapper instance
func NewStandardSearchableListable(client ExternalAPIClient, projectID, zone string) sources.SearchableListableWrapper {
	return &computeInstanceWrapper{
		client: client,
		Base: NewBase(
			projectID,
			zone,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			ComputeInstance,
		),
	}
}

// TerraformMappings returns the Terraform mappings for the ExternalType
func (d *computeInstanceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "example_resource.name",
		},
	}
}

// PotentialLinks returns the potential links for the ExternalType.
// This should include all the item types that are added as linked items in the externalTypeToSDPItem method
func (d *computeInstanceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(ComputeDisk)
}

// GetLookups returns the sources.ItemTypeLookups for the Get operation
// This is used for input validation and constructing the human readable get query description.
func (d *computeInstanceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeInstanceLookupByID,
	}
}

// Get retrieves a specific ExternalType by unique attribute and converts it to a sdp.Item
func (d *computeInstanceWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	external, err := d.client.Get(ctx, queryParts[0])
	if err != nil {
		return nil, queryError(err)
	}

	return d.externalTypeToSDPItem(external)
}

// List retrieves all ExternalType and converts them to sdp.Items
func (d *computeInstanceWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
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
func (d *computeInstanceWrapper) SearchLookups() []sources.ItemTypeLookups {
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
func (d *computeInstanceWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
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
func (d *computeInstanceWrapper) externalTypeToSDPItem(external *ExternalType) (*sdp.Item, *sdp.QueryError) {
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
func (d *computeInstanceWrapper) mapper(externalItems []*ExternalType) ([]*sdp.Item, *sdp.QueryError) {
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
