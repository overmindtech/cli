package shared

import (
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

type Impact struct {
	ToSDPItemType   shared.ItemType
	Description     string
	IsParentToChild bool
}

var (
	IPImpactBothWays = &Impact{
		Description:   "IP addresses and DNS names are tightly coupled with the source type. The linker automatically detects whether the value is an IP address or DNS name and creates the appropriate link. You can use either stdlib.NetworkIP or stdlib.NetworkDNS in blast propagation - both will automatically detect the actual type.",
		ToSDPItemType: stdlib.NetworkIP,
	}
	SecurityPolicyImpactInOnly = &Impact{
		Description:   "Any change on the security policy impacts the source, but not the other way around.",
		ToSDPItemType: ComputeSecurityPolicy,
	}
	CryptoKeyImpactInOnly = &Impact{
		Description:   "If the crypto key is updated: The source may not be able to access encrypted data. If the source is updated: The crypto key remains unaffected.",
		ToSDPItemType: CloudKMSCryptoKey,
	}
	CryptoKeyVersionImpactInOnly = &Impact{
		Description:   "If the crypto key version is updated: The source may not be able to access encrypted data. If the source is updated: The crypto key version remains unaffected.",
		ToSDPItemType: CloudKMSCryptoKeyVersion,
	}
	IAMServiceAccountImpactInOnly = &Impact{
		Description:   "If the service account is updated: The source may not be able to access encrypted data. If the source is updated: The service account remains unaffected.",
		ToSDPItemType: IAMServiceAccount,
	}
	ResourcePolicyImpactInOnly = &Impact{
		Description:   "If the resource policy is updated: The source may not be able to access the resource as expected. If the source is updated: The resource policy remains unaffected.",
		ToSDPItemType: ComputeResourcePolicy,
	}
	ComputeNetworkImpactInOnly = &Impact{
		Description:   "If the Compute Network is updated: The source may lose connectivity or fail to run as expected. If the source is updated: The network remains unaffected.",
		ToSDPItemType: ComputeNetwork,
	}
	ComputeSubnetworkImpactInOnly = &Impact{
		Description:   "If the Compute Subnetwork is updated: The source may lose connectivity or fail to run as expected. If the source is updated: The subnetwork remains unaffected.",
		ToSDPItemType: ComputeSubnetwork,
	}
)

// BlastPropagations maps item types to their blast propagation rules.
// This map is populated during source initiation by individual adapter files.
var BlastPropagations = map[shared.ItemType]map[string]*Impact{}
