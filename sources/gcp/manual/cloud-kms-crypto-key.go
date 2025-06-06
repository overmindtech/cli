package manual

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	CloudKMSCryptoKey = shared.NewItemType(gcpshared.GCP, gcpshared.CloudKMS, gcpshared.CryptoKey)

	CloudKMSCryptoKeyLookupByName = shared.NewItemTypeLookup("name", CloudKMSCryptoKey)
)
