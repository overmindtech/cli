package adapters

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeRegionCommitment = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.RegionCommitment)
