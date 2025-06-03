package manual

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeResourcePolicy = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.ResourcePolicy)
