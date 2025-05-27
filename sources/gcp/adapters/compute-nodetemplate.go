package adapters

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeNodeTemplate = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.NodeTemplate)

	ComputeNodeTemplateLookupByName = shared.NewItemTypeLookup("name", ComputeNodeTemplate)
)
