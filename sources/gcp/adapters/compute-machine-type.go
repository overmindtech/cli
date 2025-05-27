package adapters

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeMachineType = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.MachineType)
