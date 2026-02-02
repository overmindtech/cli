package shared

import (
	"context"
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
		// Extract project ID from URI if present (for cross-project references)
		extractedProjectID := ExtractPathParam("projects", query)
		if extractedProjectID != "" {
			projectID = extractedProjectID
		}
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
		// Extract project ID from URI if present (for cross-project references)
		extractedProjectID := ExtractPathParam("projects", query)
		if extractedProjectID != "" {
			projectID = extractedProjectID
		}
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
		// Extract project ID from URI if present (for cross-project references)
		extractedProjectID := ExtractPathParam("projects", query)
		scope := projectID
		if extractedProjectID != "" {
			scope = extractedProjectID
		}
		if scope != "" && name != "" {
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

// ComputeImageLinker handles linking to compute images using SEARCH method.
// SEARCH supports any format: full URIs, family names, or specific image names.
// The adapter's Search method will intelligently detect the format and use the appropriate API.
func ComputeImageLinker(projectID, _, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	// Extract project ID from the URI if present, otherwise use the provided projectID
	imageProjectID := ExtractPathParam("projects", query)
	if imageProjectID == "" {
		imageProjectID = projectID
	}

	// Extract the name/family (last component)
	name := LastPathComponent(query)
	if imageProjectID != "" && name != "" {
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   ComputeImage.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  query, // Pass the full query string so Search can detect the format
				Scope:  imageProjectID,
			},
			BlastPropagation: blastPropagation,
		}
	}

	return nil
}

// ForwardingRuleTargetLinker handles polymorphic target field in forwarding rules.
// The target field can reference multiple resource types (TargetHttpProxy, TargetHttpsProxy,
// TargetTcpProxy, TargetSslProxy, TargetPool, TargetVpnGateway, TargetInstance, ServiceAttachment).
// This function parses the URI to determine the target type and creates the appropriate link.
// Supports both full HTTPS URLs and resource name formats.
func ForwardingRuleTargetLinker(projectID, fromItemScope, targetURI string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	if targetURI == "" {
		return nil
	}

	// Determine target type from URI path
	var targetType shared.ItemType
	var scope string
	var query string

	// Extract the resource name (last component)
	name := LastPathComponent(targetURI)

	// Normalize URI - remove protocol and domain if present
	normalizedURI := targetURI
	if strings.HasPrefix(normalizedURI, "https://") {
		// Extract path from full URL: https://compute.googleapis.com/compute/v1/projects/{project}/global/targetHttpProxies/{proxy}
		if idx := strings.Index(normalizedURI, "/projects/"); idx != -1 {
			normalizedURI = normalizedURI[idx+1:]
		}
	}

	// Check URI path to determine target type (case-insensitive check for robustness)
	normalizedURI = strings.ToLower(normalizedURI)
	if strings.Contains(normalizedURI, "/targethttpproxies/") {
		targetType = ComputeTargetHttpProxy
		scope = projectID // Global resource
		query = name
	} else if strings.Contains(normalizedURI, "/targethttpsproxies/") {
		targetType = ComputeTargetHttpsProxy
		scope = projectID // Global resource
		query = name
	} else if strings.Contains(normalizedURI, "/targettcpproxies/") {
		targetType = ComputeTargetTcpProxy
		scope = projectID // Global resource
		query = name
	} else if strings.Contains(normalizedURI, "/targetsslproxies/") {
		targetType = ComputeTargetSslProxy
		scope = projectID // Global resource
		query = name
	} else if strings.Contains(normalizedURI, "/targetpools/") {
		targetType = ComputeTargetPool
		// Use original targetURI for path parameter extraction (case-sensitive)
		region := ExtractPathParam("regions", targetURI)
		if region != "" {
			scope = fmt.Sprintf("%s.%s", projectID, region)
		} else {
			scope = projectID
		}
		query = name
	} else if strings.Contains(normalizedURI, "/targetvpngateways/") {
		targetType = ComputeTargetVpnGateway
		// Use original targetURI for path parameter extraction (case-sensitive)
		region := ExtractPathParam("regions", targetURI)
		if region != "" {
			scope = fmt.Sprintf("%s.%s", projectID, region)
		} else {
			scope = projectID
		}
		query = name
	} else if strings.Contains(normalizedURI, "/targetinstances/") {
		targetType = ComputeTargetInstance
		// Use original targetURI for path parameter extraction (case-sensitive)
		zone := ExtractPathParam("zones", targetURI)
		if zone != "" {
			scope = fmt.Sprintf("%s.%s", projectID, zone)
		} else {
			scope = projectID
		}
		query = name
	} else if strings.Contains(normalizedURI, "/serviceattachments/") {
		targetType = ComputeServiceAttachment
		// Use original targetURI for path parameter extraction (case-sensitive)
		region := ExtractPathParam("regions", targetURI)
		if region != "" {
			scope = fmt.Sprintf("%s.%s", projectID, region)
		} else {
			scope = projectID
		}
		query = name
	} else {
		// Unknown target type
		return nil
	}

	if projectID != "" && scope != "" && query != "" {
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   targetType.String(),
				Method: sdp.QueryMethod_GET,
				Query:  query,
				Scope:  scope,
			},
			BlastPropagation: blastPropagation,
		}
	}

	return nil
}

// BackendServiceOrBucketLinker handles polymorphic backend service/bucket fields in URL maps.
// The service field can reference either a BackendService (global or regional) or a BackendBucket (global).
// This function parses the URI to determine the target type and creates the appropriate link.
// Supports both full HTTPS URLs and resource name formats.
func BackendServiceOrBucketLinker(projectID, fromItemScope, backendURI string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	if backendURI == "" {
		return nil
	}

	// Determine target type from URI path
	var targetType shared.ItemType
	var scope string
	var query string

	// Extract the resource name (last component)
	name := LastPathComponent(backendURI)

	// Normalize URI - remove protocol and domain if present
	normalizedURI := backendURI
	if strings.HasPrefix(normalizedURI, "https://") {
		// Extract path from full URL: https://compute.googleapis.com/compute/v1/projects/{project}/global/backendServices/{service}
		if idx := strings.Index(normalizedURI, "/projects/"); idx != -1 {
			normalizedURI = normalizedURI[idx+1:]
		}
	}

	// Check URI path to determine target type (case-insensitive check for robustness)
	normalizedURILower := strings.ToLower(normalizedURI)
	if strings.Contains(normalizedURILower, "/backendbuckets/") {
		// Backend Bucket (global, project-scoped)
		targetType = ComputeBackendBucket
		scope = projectID
		query = name
	} else if strings.Contains(normalizedURILower, "/backendservices/") {
		// Backend Service - always use same type, scope differentiates global vs regional
		targetType = ComputeBackendService
		// Use original backendURI for path parameter extraction (case-sensitive)
		region := ExtractPathParam("regions", backendURI)
		if region != "" {
			// Regional backend service - scope includes region
			scope = fmt.Sprintf("%s.%s", projectID, region)
		} else {
			// Global backend service - scope is project only
			scope = projectID
		}
		query = name
	} else {
		// Unknown backend type
		return nil
	}

	if projectID != "" && scope != "" && query != "" {
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   targetType.String(),
				Method: sdp.QueryMethod_GET,
				Query:  query,
				Scope:  scope,
			},
			BlastPropagation: blastPropagation,
		}
	}

	return nil
}

// HealthCheckLinker handles polymorphic health check fields in compute resources.
// Health checks can be either global (project-scoped) or regional (project.region-scoped).
// This function parses the URI to determine the scope and creates the appropriate link.
// Supports both full HTTPS URLs and resource name formats.
func HealthCheckLinker(projectID, fromItemScope, healthCheckURI string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	if healthCheckURI == "" {
		return nil
	}

	// Extract the resource name (last component)
	name := LastPathComponent(healthCheckURI)

	// Normalize URI - remove protocol and domain if present
	normalizedURI := healthCheckURI
	if strings.HasPrefix(normalizedURI, "https://") {
		// Extract path from full URL: https://compute.googleapis.com/compute/v1/projects/{project}/global/healthChecks/{name}
		if idx := strings.Index(normalizedURI, "/projects/"); idx != -1 {
			normalizedURI = normalizedURI[idx+1:]
		}
	}

	// Check URI path to determine scope (case-insensitive check for robustness)
	normalizedURILower := strings.ToLower(normalizedURI)
	if !strings.Contains(normalizedURILower, "/healthchecks/") {
		// Not a health check URL
		return nil
	}

	// Determine if it's regional or global
	var scope string
	region := ExtractPathParam("regions", healthCheckURI)
	if region != "" {
		// Regional health check - scope includes region
		scope = fmt.Sprintf("%s.%s", projectID, region)
	} else {
		// Global health check - scope is project only
		scope = projectID
	}

	if projectID != "" && scope != "" && name != "" {
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   ComputeHealthCheck.String(),
				Method: sdp.QueryMethod_GET,
				Query:  name,
				Scope:  scope,
			},
			BlastPropagation: blastPropagation,
		}
	}

	return nil
}

// AddressUsersLinker handles the polymorphic users field in Compute Address resources.
// The users field contains an array of URLs referencing resources that are using the address.
// This can include: forwarding rules (regional/global), instances, target VPN gateways, routers.
// This function parses the URI to determine the resource type and creates the appropriate link.
// Supports both full HTTPS URLs and resource name formats.
func AddressUsersLinker(ctx context.Context, projectID, userURI string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	if userURI == "" {
		return nil
	}

	// Determine resource type from URI path
	var targetType shared.ItemType
	var scope string
	var query string

	// Extract the resource name (last component)
	name := LastPathComponent(userURI)

	// Normalize URI - remove protocol and domain if present
	normalizedURI := userURI
	if strings.HasPrefix(normalizedURI, "https://") {
		// Extract path from full URL: https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/forwardingRules/{rule}
		if idx := strings.Index(normalizedURI, "/projects/"); idx != -1 {
			normalizedURI = normalizedURI[idx+1:]
		}
	}

	// Check URI path to determine resource type (case-insensitive check for robustness)
	normalizedURILower := strings.ToLower(normalizedURI)
	if strings.Contains(normalizedURILower, "/global/forwardingrules/") {
		// Global forwarding rule (project-scoped)
		targetType = ComputeGlobalForwardingRule
		scope = projectID
		query = name
	} else if strings.Contains(normalizedURILower, "/forwardingrules/") {
		// Regional forwarding rule
		targetType = ComputeForwardingRule
		// Use original userURI for path parameter extraction (case-sensitive)
		region := ExtractPathParam("regions", userURI)
		if region != "" {
			scope = fmt.Sprintf("%s.%s", projectID, region)
		} else {
			// Try to extract scope from URI using utility function
			extractedScope, err := ExtractScopeFromURI(ctx, userURI)
			if err == nil {
				scope = extractedScope
			} else {
				scope = projectID
			}
		}
		query = name
	} else if strings.Contains(normalizedURILower, "/instances/") {
		// VM Instance (zonal)
		targetType = ComputeInstance
		// Use original userURI for path parameter extraction (case-sensitive)
		zone := ExtractPathParam("zones", userURI)
		if zone != "" {
			scope = fmt.Sprintf("%s.%s", projectID, zone)
		} else {
			// Try to extract scope from URI using utility function
			extractedScope, err := ExtractScopeFromURI(ctx, userURI)
			if err == nil {
				scope = extractedScope
			} else {
				scope = projectID
			}
		}
		query = name
	} else if strings.Contains(normalizedURILower, "/targetvpngateways/") {
		// Target VPN Gateway (regional)
		targetType = ComputeTargetVpnGateway
		// Use original userURI for path parameter extraction (case-sensitive)
		region := ExtractPathParam("regions", userURI)
		if region != "" {
			scope = fmt.Sprintf("%s.%s", projectID, region)
		} else {
			// Try to extract scope from URI using utility function
			extractedScope, err := ExtractScopeFromURI(ctx, userURI)
			if err == nil {
				scope = extractedScope
			} else {
				scope = projectID
			}
		}
		query = name
	} else if strings.Contains(normalizedURILower, "/routers/") {
		// Router (regional)
		targetType = ComputeRouter
		// Use original userURI for path parameter extraction (case-sensitive)
		region := ExtractPathParam("regions", userURI)
		if region != "" {
			scope = fmt.Sprintf("%s.%s", projectID, region)
		} else {
			// Try to extract scope from URI using utility function
			extractedScope, err := ExtractScopeFromURI(ctx, userURI)
			if err == nil {
				scope = extractedScope
			} else {
				scope = projectID
			}
		}
		query = name
	} else {
		// Unknown resource type - log but don't fail
		log.Debugf("AddressUsersLinker: unknown resource type in users field: %s", userURI)
		return nil
	}

	if projectID != "" && scope != "" && query != "" {
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   targetType.String(),
				Method: sdp.QueryMethod_GET,
				Query:  query,
				Scope:  scope,
			},
			BlastPropagation: blastPropagation,
		}
	}

	return nil
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
	ComputeInstance:                   ZoneBaseLinkedItemQueryByName(ComputeInstance),
	ComputeInstanceGroup:              ZoneBaseLinkedItemQueryByName(ComputeInstanceGroup),
	ComputeInstanceGroupManager:       ZoneBaseLinkedItemQueryByName(ComputeInstanceGroupManager),
	ComputeRegionInstanceGroupManager: RegionBaseLinkedItemQueryByName(ComputeRegionInstanceGroupManager),
	ComputeAutoscaler:                 ZoneBaseLinkedItemQueryByName(ComputeAutoscaler),
	ComputeDisk:                       ZoneBaseLinkedItemQueryByName(ComputeDisk),
	ComputeReservation:                ZoneBaseLinkedItemQueryByName(ComputeReservation),
	ComputeNodeGroup:                  ZoneBaseLinkedItemQueryByName(ComputeNodeGroup),
	ComputeInstantSnapshot:            ZoneBaseLinkedItemQueryByName(ComputeInstantSnapshot),
	ComputeMachineImage:               ProjectBaseLinkedItemQueryByName(ComputeMachineImage),
	ComputeSecurityPolicy:             ProjectBaseLinkedItemQueryByName(ComputeSecurityPolicy),
	ComputeSnapshot:                   ProjectBaseLinkedItemQueryByName(ComputeSnapshot),
	ComputeHealthCheck:                HealthCheckLinker,            // Handles both global and regional health checks
	ComputeBackendService:             BackendServiceOrBucketLinker, // Handles both global and regional backend services, plus backend buckets
	ComputeImage:                      ComputeImageLinker,           // Custom linker that uses SEARCH for all image references (handles both names and families)
	ComputeAddress:                    RegionBaseLinkedItemQueryByName(ComputeAddress),
	ComputeForwardingRule:             RegionBaseLinkedItemQueryByName(ComputeForwardingRule),
	ComputeInterconnectAttachment:     RegionBaseLinkedItemQueryByName(ComputeInterconnectAttachment),
	ComputeNodeTemplate:               RegionBaseLinkedItemQueryByName(ComputeNodeTemplate),
	// Target proxy types (global, project-scoped) - use polymorphic linker for forwarding rule target field
	ComputeTargetHttpProxy:  ForwardingRuleTargetLinker,
	ComputeTargetHttpsProxy: ForwardingRuleTargetLinker,
	ComputeTargetTcpProxy:   ForwardingRuleTargetLinker,
	ComputeTargetSslProxy:   ForwardingRuleTargetLinker,
	// Target pool (regional) - use polymorphic linker
	ComputeTargetPool: ForwardingRuleTargetLinker,
	// Target VPN Gateway (regional) - use polymorphic linker
	ComputeTargetVpnGateway: ForwardingRuleTargetLinker,
	// Target Instance (zonal) - use polymorphic linker
	ComputeTargetInstance: ForwardingRuleTargetLinker,
	// Service Attachment (regional) - use polymorphic linker
	ComputeServiceAttachment: ForwardingRuleTargetLinker,
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
	// ProjectFolderOrganizationLinker handles polymorphic project/folder/organization fields in resource names.
	// The name field can reference projects, folders, or organizations depending on the resource scope.
	// This function parses the name to determine the target type and creates the appropriate link.
	// This is registered for CloudResourceManagerProject but can detect and link to all three types.
	CloudResourceManagerProject: func(projectID, _, name string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		if name == "" {
			return nil
		}
		// Extract resource ID based on prefix - handle projects, folders, and organizations
		if strings.HasPrefix(name, "projects/") {
			projectIDFromName := ExtractPathParam("projects", name)
			if projectIDFromName != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudResourceManagerProject.String(),
						Method: sdp.QueryMethod_GET,
						Query:  projectIDFromName,
						Scope:  projectIDFromName, // Project scope uses project ID as scope
					},
					BlastPropagation: blastPropagation,
				}
			}
		} else if strings.HasPrefix(name, "folders/") {
			folderID := ExtractPathParam("folders", name)
			if folderID != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudResourceManagerFolder.String(),
						Method: sdp.QueryMethod_GET,
						Query:  folderID,
						Scope:  projectID, // Folder scope uses project ID (may need adjustment when folder adapter is created)
					},
					BlastPropagation: blastPropagation,
				}
			}
		} else if strings.HasPrefix(name, "organizations/") {
			orgID := ExtractPathParam("organizations", name)
			if orgID != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudResourceManagerOrganization.String(),
						Method: sdp.QueryMethod_GET,
						Query:  orgID,
						Scope:  projectID, // Organization scope uses project ID (may need adjustment when org adapter is created)
					},
					BlastPropagation: blastPropagation,
				}
			}
		}
		return nil
	},
	CloudResourceManagerFolder: func(projectID, _, name string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		if name == "" {
			return nil
		}
		// Extract folder ID from name
		if strings.HasPrefix(name, "folders/") {
			folderID := ExtractPathParam("folders", name)
			if folderID != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudResourceManagerFolder.String(),
						Method: sdp.QueryMethod_GET,
						Query:  folderID,
						Scope:  projectID, // Folder scope uses project ID (may need adjustment when folder adapter is created)
					},
					BlastPropagation: blastPropagation,
				}
			}
		}
		return nil
	},
	CloudResourceManagerOrganization: func(projectID, _, name string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		if name == "" {
			return nil
		}
		// Extract organization ID from name
		if strings.HasPrefix(name, "organizations/") {
			orgID := ExtractPathParam("organizations", name)
			if orgID != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudResourceManagerOrganization.String(),
						Method: sdp.QueryMethod_GET,
						Query:  orgID,
						Scope:  projectID, // Organization scope uses project ID (may need adjustment when org adapter is created)
					},
					BlastPropagation: blastPropagation,
				}
			}
		}
		return nil
	},
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
	stdlib.NetworkHTTP: func(_, _, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		if query != "" {
			// Extract the base URL (remove query parameters and fragments)
			httpURL := query
			if idx := strings.Index(httpURL, "?"); idx != -1 {
				httpURL = httpURL[:idx]
			}
			if idx := strings.Index(httpURL, "#"); idx != -1 {
				httpURL = httpURL[:idx]
			}

			if httpURL != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "http",
						Method: sdp.QueryMethod_SEARCH,
						Query:  httpURL,
						Scope:  "global",
					},
					BlastPropagation: blastPropagation,
				}
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
		if query == "" {
			return nil
		}

		// Supported formats:
		// 1) //bigquery.googleapis.com/projects/PROJECT_ID/datasets/DATASET_ID/tables/TABLE_ID
		//    See: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.dataScans#DataSource
		// 2) projects/PROJECT_ID/datasets/DATASET_ID/tables/TABLE_ID
		// 3) {projectId}.{datasetId}.{tableId}
		//    See: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.subscriptions#bigqueryconfig
		// 4) bq://projectId or bq://projectId.bqDatasetId or bq://projectId.bqDatasetId.bqTableId
		//    See: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/BigQueryDestination

		// Try full URI format first: //bigquery.googleapis.com/projects/PROJECT_ID/datasets/DATASET_ID/tables/TABLE_ID
		if strings.HasPrefix(query, "//bigquery.googleapis.com/") || strings.HasPrefix(query, "https://bigquery.googleapis.com/") {
			values := ExtractPathParams(query, "projects", "datasets", "tables")
			if len(values) == 3 && values[0] != "" && values[1] != "" && values[2] != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   BigQueryTable.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(values[1], values[2]),
						Scope:  values[0],
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		// Try path format: projects/PROJECT_ID/datasets/DATASET_ID/tables/TABLE_ID
		if strings.HasPrefix(query, "projects/") {
			values := ExtractPathParams(query, "projects", "datasets", "tables")
			if len(values) == 3 && values[0] != "" && values[1] != "" && values[2] != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   BigQueryTable.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(values[1], values[2]),
						Scope:  values[0],
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		// Try dot-separated format: {projectId}.{datasetId}.{tableId} or bq://projectId.bqDatasetId.bqTableId
		query = strings.TrimPrefix(query, "bq://")
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
	aws.MSKCluster:            AWSLinkByARN("msk-cluster"),
	SQLAdminInstance: func(projectID, _, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		// Supported formats:
		// 1) {project}:{location}:{instance} (Cloud Run format)
		//    See: https://cloud.google.com/run/docs/reference/rest/v2/Volume#cloudsqlinstance
		// 2) projects/{project}/instances/{instance} (full resource name)
		// 3) {instance} (simple instance name, uses projectID from context)

		// Try colon separator first
		parts := strings.Split(query, ":")
		if len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   SQLAdminInstance.String(),
					Method: sdp.QueryMethod_GET,
					Query:  parts[2],
					Scope:  parts[0],
				},
				BlastPropagation: blastPropagation,
			}
		}

		// Try slash separator (full resource name)
		if strings.Contains(query, "/") {
			values := ExtractPathParams(query, "projects", "instances")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   SQLAdminInstance.String(),
						Method: sdp.QueryMethod_GET,
						Query:  values[1],
						Scope:  values[0],
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		// Single word (simple instance name) - use projectID from context
		if !strings.Contains(query, ":") && !strings.Contains(query, "/") && query != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   SQLAdminInstance.String(),
					Method: sdp.QueryMethod_GET,
					Query:  query,
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			}
		}

		return nil
	},
	BigQueryDataset: func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		// Supported formats:
		// 1) datasetId (e.g., "my_dataset")
		// 2) projects/{project}/datasets/{dataset}
		// 3) project:dataset (BigQuery FullID style)
		// 4) bigquery.googleapis.com/projects/{project}/datasets/{dataset}
		if query == "" {
			return nil
		}

		// Normalize URI formats (bigquery.googleapis.com/... or https://bigquery.googleapis.com/...)
		normalizedQuery := query
		if strings.Contains(query, ".googleapis.com/") {
			// Handle service destination formats: bigquery.googleapis.com/path
			parts := strings.SplitN(query, ".googleapis.com/", 2)
			if len(parts) > 1 {
				path := parts[1]
				// Strip version paths like /v1/, /v2/, /bigquery/v2/, etc.
				pathParts := strings.Split(path, "/")
				// Remove version paths (v1, v2, bigquery/v2, etc.) that appear before "projects"
				for i, part := range pathParts {
					if part == "projects" {
						normalizedQuery = strings.Join(pathParts[i:], "/")
						break
					}
				}
			}
		} else if strings.HasPrefix(query, "https://") || strings.HasPrefix(query, "http://") {
			// Handle HTTPS/HTTP URLs: https://bigquery.googleapis.com/bigquery/v2/projects/...
			uri := query[strings.Index(query, "://")+3:]
			parts := strings.SplitN(uri, "/", 2)
			if len(parts) > 1 {
				path := parts[1]
				// Strip version paths
				pathParts := strings.Split(path, "/")
				for i, part := range pathParts {
					if part == "projects" {
						normalizedQuery = strings.Join(pathParts[i:], "/")
						break
					}
				}
			}
		}

		// Try path-style: projects/{project}/datasets/{dataset}
		if strings.Contains(normalizedQuery, "projects/") && strings.Contains(normalizedQuery, "datasets/") {
			values := ExtractPathParams(normalizedQuery, "projects", "datasets")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				parsedProject := values[0]
				dataset := values[1]
				scope := parsedProject
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   BigQueryDataset.String(),
						Method: sdp.QueryMethod_GET,
						Query:  dataset,
						Scope:  scope,
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		// Try fullID style: project:dataset
		if strings.HasPrefix(query, "project:") {
			parts := strings.Split(query, ":")
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   BigQueryDataset.String(),
						Method: sdp.QueryMethod_GET,
						Query:  parts[1], // dataset ID
						Scope:  parts[0], // project ID
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		if strings.Contains(query, ":") || strings.Contains(query, "/") {
			// At this point we don't recognize the pattern.
			return nil
		}

		// Fallback: treat as datasetId in current project
		if projectID != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   BigQueryDataset.String(),
					Method: sdp.QueryMethod_GET,
					Query:  query, // dataset ID
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			}
		}
		return nil
	},
	BigQueryModel: func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		// Supported format:
		// projects/{project}/datasets/{dataset}/models/{model}
		if query == "" {
			return nil
		}

		if strings.HasPrefix(query, "projects/") {
			// Path-style
			values := ExtractPathParams(query, "projects", "datasets", "models")
			if len(values) == 3 && values[0] != "" && values[1] != "" && values[2] != "" {
				parsedProject := values[0]
				dataset := values[1]
				model := values[2]
				scope := parsedProject
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   BigQueryModel.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(dataset, model),
						Scope:  scope,
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		if strings.HasPrefix(query, "datasets/") {
			values := ExtractPathParams(query, "datasets", "models")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				scope := projectID
				dataset := values[0]
				model := values[1]
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   BigQueryModel.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(dataset, model),
						Scope:  scope,
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		return nil
	},
	StorageBucket: func(projectID, fromItemScope, query string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		if query == "" {
			return nil
		}

		// Supported formats:
		// 1) //storage.googleapis.com/projects/PROJECT_ID/buckets/BUCKET_ID
		// 2) gs://bucket-name
		// 3) gs://bucket-name/path/to/file
		// 4) bucket-name (without gs:// prefix)

		// Try full URI format first: //storage.googleapis.com/projects/PROJECT_ID/buckets/BUCKET_ID
		if strings.HasPrefix(query, "//storage.googleapis.com/") || strings.HasPrefix(query, "https://storage.googleapis.com/") {
			values := ExtractPathParams(query, "projects", "buckets")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   StorageBucket.String(),
						Method: sdp.QueryMethod_GET,
						Query:  values[1],
						Scope:  values[0],
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		// Try path format: projects/PROJECT_ID/buckets/BUCKET_ID
		if strings.HasPrefix(query, "projects/") {
			values := ExtractPathParams(query, "projects", "buckets")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				return &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   StorageBucket.String(),
						Method: sdp.QueryMethod_GET,
						Query:  values[1],
						Scope:  values[0],
					},
					BlastPropagation: blastPropagation,
				}
			}
		}

		// Strip gs:// prefix if present
		query = strings.TrimPrefix(query, "gs://")

		// Extract bucket name (everything before the first slash)
		bucketName := query
		if idx := strings.Index(query, "/"); idx != -1 {
			bucketName = query[:idx]
		}

		// Validate bucket name is not empty
		if bucketName == "" {
			return nil
		}

		// Storage buckets are project-scoped
		if projectID != "" {
			return &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   StorageBucket.String(),
					Method: sdp.QueryMethod_GET,
					Query:  bucketName,
					Scope:  projectID,
				},
				BlastPropagation: blastPropagation,
			}
		}

		return nil
	},
	// OrgPolicyPolicy name field can reference parent project, folder, or organization
	// This linker is registered for all three parent types since the name field can reference any of them
	// Format: projects/{project_number}/policies/{constraint} or
	//         folders/{folder_id}/policies/{constraint} or
	//         organizations/{organization_id}/policies/{constraint}
	CloudResourceManagerProject: func(projectID, _, policyName string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		return orgPolicyParentLinker(projectID, policyName, blastPropagation)
	},
	CloudResourceManagerFolder: func(projectID, _, policyName string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		return orgPolicyParentLinker(projectID, policyName, blastPropagation)
	},
	CloudResourceManagerOrganization: func(projectID, _, policyName string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
		return orgPolicyParentLinker(projectID, policyName, blastPropagation)
	},
}

// orgPolicyParentLinker parses an org policy name to determine the parent resource type
// and creates a linked item query for the appropriate parent (project, folder, or organization).
// The policy name format is: projects/{project_number}/policies/{constraint} or
//
//	folders/{folder_id}/policies/{constraint} or
//	organizations/{organization_id}/policies/{constraint}
//
// It also handles simple project references: projects/{project_id} (without /policies/)
// In that case, the scope should be the current project (projectID), not the referenced project.
func orgPolicyParentLinker(projectID, policyName string, blastPropagation *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	if policyName == "" {
		return nil
	}

	var targetType shared.ItemType
	var parentID string
	var scope string

	// Parse the policy name to determine parent type
	if strings.HasPrefix(policyName, "projects/") {
		// Check if this is a simple project reference (projects/{project_id}) or org policy (projects/{project_id}/policies/...)
		if strings.Contains(policyName, "/policies/") {
			// Org policy format: projects/{project_number}/policies/{constraint}
			values := ExtractPathParams(policyName, "projects")
			if len(values) >= 1 && values[0] != "" {
				targetType = CloudResourceManagerProject
				parentID = values[0]
				scope = parentID // For org policies, use the project ID as scope
			}
		} else {
			// Simple project reference: projects/{project_id}
			// Extract project ID and use current project as scope
			values := ExtractPathParams(policyName, "projects")
			if len(values) >= 1 && values[0] != "" {
				targetType = CloudResourceManagerProject
				parentID = values[0]
				scope = projectID // Use current project as scope when querying for another project
			}
		}
	} else if strings.HasPrefix(policyName, "folders/") {
		// Extract folder ID from: folders/{folder_id}/policies/{constraint}
		values := ExtractPathParams(policyName, "folders")
		if len(values) >= 1 && values[0] != "" {
			targetType = CloudResourceManagerFolder
			parentID = values[0]
			// Folders are organization-scoped, but we don't have org ID here
			// Use projectID as fallback scope (folder adapters will need to handle this)
			scope = projectID
		}
	} else if strings.HasPrefix(policyName, "organizations/") {
		// Extract organization ID from: organizations/{organization_id}/policies/{constraint}
		values := ExtractPathParams(policyName, "organizations")
		if len(values) >= 1 && values[0] != "" {
			targetType = CloudResourceManagerOrganization
			parentID = values[0]
			// Organizations are global-scoped
			scope = "global"
		}
	}

	if parentID != "" && scope != "" {
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   targetType.String(),
				Method: sdp.QueryMethod_GET,
				Query:  parentID,
				Scope:  scope,
			},
			BlastPropagation: blastPropagation,
		}
	}

	return nil
}
