package manual

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	CloudKMSEKMConnection             = shared.NewItemType(gcpshared.GCP, gcpshared.CloudKMS, gcpshared.CloudKMSEKMConnection)
	CloudKMSEKMConnectionLookupByName = shared.NewItemTypeLookup("name", CloudKMSEKMConnection)
)
