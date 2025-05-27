package adapters

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeAcceleratorType = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.AcceleratorType)
