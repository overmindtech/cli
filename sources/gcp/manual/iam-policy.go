package manual

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	IAMPolicy = shared.NewItemType(gcpshared.GCP, gcpshared.IAM, gcpshared.Policy)

	IAMPolicyLookupName = shared.NewItemTypeLookup("name", IAMPolicy)
)
