package shared

import (
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

type Impact struct {
	ToSDPItemType    shared.ItemType
	Description      string
	BlastPropagation *sdp.BlastPropagation
	IsParentToChild  bool
}

var (
	ImpactInOnly   = &sdp.BlastPropagation{In: true}
	impactBothWays = &sdp.BlastPropagation{In: true, Out: true}
)

var (
	IPImpactBothWays = &Impact{
		Description:      "IP addresses are tightly coupled with the source type.",
		ToSDPItemType:    stdlib.NetworkIP,
		BlastPropagation: impactBothWays,
	}
	SecurityPolicyImpactInOnly = &Impact{
		Description:      "Any change on the security policy impacts the source, but not the other way around.",
		ToSDPItemType:    ComputeSecurityPolicy,
		BlastPropagation: ImpactInOnly,
	}
	CryptoKeyImpactInOnly = &Impact{
		Description:      "If the crypto key is updated: The source may not be able to access encrypted data. If the source is updated: The crypto key remains unaffected.",
		ToSDPItemType:    CloudKMSCryptoKey,
		BlastPropagation: ImpactInOnly,
	}
	CryptoKeyVersionImpactInOnly = &Impact{
		Description:      "If the crypto key version is updated: The source may not be able to access encrypted data. If the source is updated: The crypto key version remains unaffected.",
		ToSDPItemType:    CloudKMSCryptoKeyVersion,
		BlastPropagation: ImpactInOnly,
	}
	IAMServiceAccountImpactInOnly = &Impact{
		Description:      "If the service account is updated: The source may not be able to access encrypted data. If the source is updated: The service account remains unaffected.",
		ToSDPItemType:    IAMServiceAccount,
		BlastPropagation: ImpactInOnly,
	}
	ResourcePolicyImpactInOnly = &Impact{
		Description:      "If the resource policy is updated: The source may not be able to access the resource as expected. If the source is updated: The resource policy remains unaffected.",
		ToSDPItemType:    ComputeResourcePolicy,
		BlastPropagation: ImpactInOnly,
	}
	ComputeNetworkImpactInOnly = &Impact{
		Description:      "If the Compute Network is updated: The source may lose connectivity or fail to run as expected. If the source is updated: The network remains unaffected.",
		ToSDPItemType:    ComputeNetwork,
		BlastPropagation: ImpactInOnly,
	}
	ComputeSubnetworkImpactInOnly = &Impact{
		Description:      "If the Compute Subnetwork is updated: The source may lose connectivity or fail to run as expected. If the source is updated: The subnetwork remains unaffected.",
		ToSDPItemType:    ComputeSubnetwork,
		BlastPropagation: ImpactInOnly,
	}
)

// BlastPropagations maps item types to their blast propagation rules.
// This map is populated during source initiation by individual adapter files.
var BlastPropagations = map[shared.ItemType]map[string]*Impact{}
