package manual

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	IAMServiceAccount = shared.NewItemType(gcpshared.GCP, gcpshared.IAM, gcpshared.ServiceAccount)

	IAMServiceAccountLookupByEmailOrUniqueID = shared.NewItemTypeLookup("email or unique_id", IAMServiceAccount)
)
