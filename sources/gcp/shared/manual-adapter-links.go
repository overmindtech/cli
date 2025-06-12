package shared

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func ZoneBaseLinkedItemQueryByName(sdpItem shared.ItemType) func(projectID, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	return func(projectID, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		zone := ExtractPathParam("zones", query)
		name := LastPathComponent(query)
		if projectID != "" && zone != "" && name != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   sdpItem.String(),
					Method: sdp.QueryMethod_GET,
					Query:  name,
					Scope:  fmt.Sprintf("%s.%s", projectID, zone),
				},
				BlastPropagation: blastPropagation,
			}
		}

		return nil
	}
}

func RegionBaseLinkedItemQueryByName(sdpItem shared.ItemType) func(projectID, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	return func(projectID, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		region := ExtractPathParam("regions", query)
		name := LastPathComponent(query)
		if projectID != "" && region != "" && name != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   sdpItem.String(),
					Method: sdp.QueryMethod_GET,
					Query:  name,
					Scope:  fmt.Sprintf("%s.%s", projectID, region),
				},
				BlastPropagation: blastPropagation,
			}
		}

		return nil
	}
}

func ProjectBaseLinkedItemQueryByName(sdpItem shared.ItemType) func(projectID, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	return func(projectID, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		name := LastPathComponent(query)
		if projectID != "" && name != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   sdpItem.String(),
					Method: sdp.QueryMethod_GET,
					Query:  name,
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			}
		}

		return nil
	}
}

// ManualAdapterGetLinksByAssetType defines how to link manually created adapters.
// Expects that the query will have all the necessary information to create the linked item query.
var ManualAdapterGetLinksByAssetType = map[shared.ItemType]func(projectID, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery{
	ComputeInstance:             ZoneBaseLinkedItemQueryByName(ComputeInstance),
	ComputeInstanceGroup:        ZoneBaseLinkedItemQueryByName(ComputeInstanceGroup),
	ComputeInstanceGroupManager: ZoneBaseLinkedItemQueryByName(ComputeInstanceGroupManager),
	ComputeAutoscaler:           ZoneBaseLinkedItemQueryByName(ComputeAutoscaler),
	ComputeDisk:                 ZoneBaseLinkedItemQueryByName(ComputeDisk),
	ComputeMachineType:          ZoneBaseLinkedItemQueryByName(ComputeMachineType),
	ComputeReservation:          ZoneBaseLinkedItemQueryByName(ComputeReservation),
	ComputeNodeGroup:            ZoneBaseLinkedItemQueryByName(ComputeNodeGroup),
	ComputeInstantSnapshot:      ZoneBaseLinkedItemQueryByName(ComputeInstantSnapshot),
	ComputeMachineImage:         ProjectBaseLinkedItemQueryByName(ComputeMachineImage),
	ComputeLicense:              ProjectBaseLinkedItemQueryByName(ComputeLicense),
	ComputeSecurityPolicy:       ProjectBaseLinkedItemQueryByName(ComputeSecurityPolicy),
	ComputeSnapshot:             ProjectBaseLinkedItemQueryByName(ComputeSnapshot),
	ComputeHealthCheck:          ProjectBaseLinkedItemQueryByName(ComputeHealthCheck),
	ComputeBackendService:       ProjectBaseLinkedItemQueryByName(ComputeBackendService),
	ComputeImage:                ProjectBaseLinkedItemQueryByName(ComputeImage),
	ComputeAddress:              RegionBaseLinkedItemQueryByName(ComputeAddress),
	ComputeForwardingRule:       RegionBaseLinkedItemQueryByName(ComputeForwardingRule),
	ComputeNodeTemplate:         RegionBaseLinkedItemQueryByName(ComputeNodeTemplate),
	CloudKMSCryptoKeyVersion: func(projectID, keyName string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		location := ExtractPathParam("locations", keyName)
		keyRing := ExtractPathParam("keyRings", keyName)
		cryptoKey := ExtractPathParam("cryptoKeys", keyName)
		cryptoKeyVersion := ExtractPathParam("cryptoKeyVersions", keyName)

		if projectID != "" && location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   CloudKMSCryptoKeyVersion.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			}
		}
		return nil
	},
	IAMServiceAccountKey: ProjectBaseLinkedItemQueryByName(IAMServiceAccountKey),
	IAMServiceAccount:    ProjectBaseLinkedItemQueryByName(IAMServiceAccount),
	CloudKMSKeyRing:      RegionBaseLinkedItemQueryByName(CloudKMSKeyRing),
	stdlib.NetworkIP: func(_, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		if query != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  query,
					Scope:  "global",
				},
				BlastPropagation: blastPropagation,
			}
		}
		return nil
	},
	stdlib.NetworkDNS: func(_, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		if query != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  query,
					Scope:  "global",
				},
				BlastPropagation: blastPropagation,
			}
		}
		return nil
	},
}
