package adapters

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var CloudKMSCryptoKeyVersion = shared.NewItemType(gcpshared.GCP, gcpshared.CloudKMS, gcpshared.CryptoKeyVersion)
