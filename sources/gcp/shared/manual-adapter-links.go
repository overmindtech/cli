package shared

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/sdp-go"
	aws "github.com/overmindtech/cli/sources/aws/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func ZoneBaseLinkedItemQueryByName(sdpItem shared.ItemType) func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	return func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		name := LastPathComponent(query)
		zone := ExtractPathParam("zones", query)
		scope := fromItemScope
		if zone != "" {
			scope = fmt.Sprintf("%s.%s", projectID, zone)
		}
		if projectID != "" && scope != "" && name != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   sdpItem.String(),
					Method: sdp.QueryMethod_GET,
					Query:  name,
					Scope:  scope,
				},
				BlastPropagation: blastPropagation,
			}
		}

		return nil
	}
}

func RegionBaseLinkedItemQueryByName(sdpItem shared.ItemType) func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	return func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		name := LastPathComponent(query)
		scope := fromItemScope
		region := ExtractPathParam("regions", query)
		if region != "" {
			scope = fmt.Sprintf("%s.%s", projectID, region)
		}
		if projectID != "" && region != "" && name != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   sdpItem.String(),
					Method: sdp.QueryMethod_GET,
					Query:  name,
					Scope:  scope,
				},
				BlastPropagation: blastPropagation,
			}
		}

		return nil
	}
}

func ProjectBaseLinkedItemQueryByName(sdpItem shared.ItemType) func(projectID, _, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	return func(projectID, _, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
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

func AWSLinkByARN(awsItem string) func(_, _, arn string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	return func(_, _, arn string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference-arns.html#arns-syntax
		parts := strings.Split(arn, ":")
		if len(parts) < 5 {
			log.Warnf("invalid ARN: %s", arn)
			return nil
		}
		/*
			arn:partition:service:region:account-id:resource-id
			arn:partition:service:region:account-id:resource-type/resource-id
			arn:partition:service:region:account-id:resource-type:resource-id
		*/
		region := parts[3]
		accountID := parts[4]
		scope := accountID
		if region != "" {
			scope = fmt.Sprintf("%s.%s", accountID, region)
		}
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   awsItem,
				Method: sdp.QueryMethod_SEARCH,
				Query:  arn, // By default, we search by the full ARN
				Scope:  scope,
			},
			BlastPropagation: blastPropagation,
		}
	}
}

// ManualAdapterLinksByAssetType defines how to link a specific item type to its linked items.
// This is used when the query that holds the linked item information is not a standard query for the dynamic adapter framework.
// So we need to manually define how to create the linked item query based on the item type and the query string.
//
// Expects that the query will have all the necessary information to create the linked item query.
var ManualAdapterLinksByAssetType = map[shared.ItemType]func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery{
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
	ComputeSecurityPolicy:       ProjectBaseLinkedItemQueryByName(ComputeSecurityPolicy),
	ComputeSnapshot:             ProjectBaseLinkedItemQueryByName(ComputeSnapshot),
	ComputeHealthCheck:          ProjectBaseLinkedItemQueryByName(ComputeHealthCheck),
	ComputeBackendService:       ProjectBaseLinkedItemQueryByName(ComputeBackendService),
	ComputeImage:                ProjectBaseLinkedItemQueryByName(ComputeImage),
	ComputeAddress:              RegionBaseLinkedItemQueryByName(ComputeAddress),
	ComputeForwardingRule:       RegionBaseLinkedItemQueryByName(ComputeForwardingRule),
	ComputeNodeTemplate:         RegionBaseLinkedItemQueryByName(ComputeNodeTemplate),
	CloudKMSCryptoKeyVersion: func(projectID, _, keyName string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
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
	stdlib.NetworkIP: func(_, _, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
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
	stdlib.NetworkDNS: func(_, _, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
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
	CloudKMSCryptoKey: func(projectID, _, keyName string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		//"projects/{kms_project_id}/locations/{region}/keyRings/{key_region}/cryptoKeys/{key}
		values := ExtractPathParams(keyName, "locations", "keyRings", "cryptoKeys")
		if len(values) != 3 {
			return nil
		}

		location := values[0]
		keyRing := values[1]
		cryptoKey := values[2]
		if projectID != "" && location != "" && keyRing != "" && cryptoKey != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   CloudKMSCryptoKey.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey),
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			}
		}
		return nil
	},
	BigQueryTable: func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		// expected query format: {projectId}.{datasetId}.{tableId}
		// See: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.subscriptions#bigqueryconfig
		parts := strings.Split(query, ".")
		if len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   BigQueryTable.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(parts[1], parts[2]),
					Scope:  parts[0],
				},
				BlastPropagation: blastPropagation,
			}
		}

		return nil
	},
	aws.KinesisStream:         AWSLinkByARN("kinesis-stream"),
	aws.KinesisStreamConsumer: AWSLinkByARN("kinesis-stream-consumer"),
	aws.IAMRole:               AWSLinkByARN("iam-role"),
	SQLAdminInstance: func(_, _, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		// expected query format: {project}:{location}:{instance}
		// See: https://cloud.google.com/run/docs/reference/rest/v2/Volume#cloudsqlinstance
		parts := strings.Split(query, ":")
		if len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
			// It will be a project level adapter
			// https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/instances/get
			projectID := parts[0]
			instance := parts[2]
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   SQLAdminInstance.String(),
					Method: sdp.QueryMethod_GET,
					Query:  instance,
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			}
		}
		return nil
	},
}
