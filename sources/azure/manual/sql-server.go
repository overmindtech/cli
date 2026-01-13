package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var SQLServerLookupByName = shared.NewItemTypeLookup("name", azureshared.SQLServer)

func NewSqlServer(client clients.SqlServersClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &sqlServerWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.SQLServer,
		),
	}
}

type sqlServerWrapper struct {
	client clients.SqlServersClient

	*azureshared.ResourceGroupBase
}

func (s sqlServerWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := s.client.ListByResourceGroup(ctx, s.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
		}
		for _, server := range page.Value {
			if server.Name == nil {
				continue
			}
			item, sdpErr := s.azureSqlServerToSDPItem(server)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s sqlServerWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	pager := s.client.ListByResourceGroup(ctx, s.ResourceGroup(), nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, s.DefaultScope(), s.Type()))
			return
		}
		for _, server := range page.Value {
			if server.Name == nil {
				continue
			}
			item, sdpErr := s.azureSqlServerToSDPItem(server)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s sqlServerWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Get requires 1 query part: serverName"), s.DefaultScope(), s.Type())
	}
	serverName := queryParts[0]
	if serverName == "" {
		return nil, azureshared.QueryError(errors.New("serverName is empty"), s.DefaultScope(), s.Type())
	}

	resp, err := s.client.Get(ctx, s.ResourceGroup(), serverName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	return s.azureSqlServerToSDPItem(&resp.Server)
}

func (s sqlServerWrapper) azureSqlServerToSDPItem(server *armsql.Server) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(server, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	serverName := ""
	if server.Name != nil {
		serverName = *server.Name
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.SQLServer.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           s.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(server.Tags),
	}

	// Child resources - can be discovered via SEARCH using server name
	// These child resources have their own REST API endpoints under the SQL Server
	if serverName != "" {
		// Link to Databases (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/databases/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/databases
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLDatabase.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // SQL Server changes (especially deletion) affect databases
				Out: false, // Database changes don't affect the SQL Server itself
			}, // SQL Databases are child resources that depend on their parent SQL Server
		})

		// Link to Elastic Pools (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/elastic-pools/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/elasticPools
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLElasticPool.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // SQL Server changes affect elastic pools
				Out: false, // Elastic pool changes don't affect the SQL Server itself
			}, // SQL Elastic Pools are child resources that depend on their parent SQL Server
		})

		// Link to Firewall Rules (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/firewall-rules/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/firewallRules
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerFirewallRule.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect firewall rules
				Out: true, // Firewall rule changes affect server connectivity
			}, // SQL Server Firewall Rules are child resources that control server access
		})

		// Link to Virtual Network Rules (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/virtual-network-rules/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/virtualNetworkRules
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerVirtualNetworkRule.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect virtual network rules
				Out: true, // Virtual network rule changes affect server connectivity
			}, // SQL Server Virtual Network Rules are child resources that control server access
		})

		// Link to Server Keys (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-keys/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/keys
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerKey.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect server keys
				Out: true, // Server key changes (especially deletion) affect server encryption and availability
			}, // SQL Server Keys are child resources used for encryption
		})

		// Link to Failover Groups (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/failover-groups/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/failoverGroups
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerFailoverGroup.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect failover groups
				Out: true, // Failover group changes affect server availability and failover behavior
			}, // SQL Server Failover Groups are child resources that manage high availability
		})

		// Link to Administrators (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-azure-ad-administrators/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/administrators/ActiveDirectory
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerAdministrator.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect administrators
				Out: true, // Administrator changes affect server authentication and access control
			}, // SQL Server Administrators are child resources that control authentication
		})

		// Link to Sync Groups (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/sync-groups/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/syncGroups
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerSyncGroup.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // SQL Server changes affect sync groups
				Out: false, // Sync group changes don't affect the SQL Server itself
			}, // SQL Server Sync Groups are child resources for data synchronization
		})

		// Link to Sync Agents (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/sync-agents/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/syncAgents
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerSyncAgent.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // SQL Server changes affect sync agents
				Out: false, // Sync agent changes don't affect the SQL Server itself
			}, // SQL Server Sync Agents are child resources for data synchronization
		})

		// Link to Private Endpoint Connections (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/private-endpoint-connections/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/privateEndpointConnections
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerPrivateEndpointConnection.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect private endpoint connections
				Out: true, // Private endpoint connection changes affect server network connectivity
			}, // SQL Server Private Endpoint Connections are child resources that manage private network access
		})

		// Link to Network Private Endpoints (external resources) from PrivateEndpointConnections
		// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateEndpoints/{privateEndpointName}
		if server.Properties != nil && server.Properties.PrivateEndpointConnections != nil {
			for _, peConnection := range server.Properties.PrivateEndpointConnections {
				if peConnection.Properties != nil && peConnection.Properties.PrivateEndpoint != nil && peConnection.Properties.PrivateEndpoint.ID != nil {
					privateEndpointID := *peConnection.Properties.PrivateEndpoint.ID
					privateEndpointName := azureshared.ExtractResourceName(privateEndpointID)
					if privateEndpointName != "" {
						// Extract scope from resource ID if it's in a different resource group
						scope := s.DefaultScope()
						if extractedScope := azureshared.ExtractScopeFromResourceID(privateEndpointID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkPrivateEndpoint.String(),
								Method: sdp.QueryMethod_GET,
								Query:  privateEndpointName,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // Private endpoint changes (deletion, network configuration) affect the SQL Server's private connectivity
								Out: true, // SQL Server deletion or configuration changes may affect the private endpoint's connection state
							}, // Private endpoints are tightly coupled to the SQL Server - changes affect connectivity
						})
					}
				}
			}
		}

		// Link to Auditing Settings (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-auditing-settings/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/auditingSettings/default
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerAuditingSetting.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect auditing settings
				Out: true, // Auditing setting changes affect server security and compliance
			}, // SQL Server Auditing Settings are child resources that control audit logging
		})

		// Link to Security Alert Policies (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-security-alert-policies/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/securityAlertPolicies/default
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerSecurityAlertPolicy.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect security alert policies
				Out: true, // Security alert policy changes affect server security monitoring
			}, // SQL Server Security Alert Policies are child resources that control threat detection
		})

		// Link to Vulnerability Assessments (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-vulnerability-assessments/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/vulnerabilityAssessments/default
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerVulnerabilityAssessment.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect vulnerability assessments
				Out: true, // Vulnerability assessment changes affect server security scanning
			}, // SQL Server Vulnerability Assessments are child resources that control security scanning
		})

		// Link to Encryption Protector (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/encryption-protectors/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/encryptionProtector
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerEncryptionProtector.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect encryption protector
				Out: true, // Encryption protector changes affect server encryption and data access
			}, // SQL Server Encryption Protector is a child resource that controls encryption
		})

		// Link to Blob Auditing Policies (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-blob-auditing-policies/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/auditingSettings/default
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerBlobAuditingPolicy.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect blob auditing policies
				Out: true, // Blob auditing policy changes affect server audit logging
			}, // SQL Server Blob Auditing Policies are child resources that control blob audit logging
		})

		// Link to Automatic Tuning (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-automatic-tuning/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/automaticTuning
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerAutomaticTuning.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect automatic tuning
				Out: true, // Automatic tuning changes affect server performance optimization
			}, // SQL Server Automatic Tuning is a child resource that controls performance optimization
		})

		// Link to Advanced Threat Protection Settings (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-advanced-threat-protection-settings/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/advancedThreatProtectionSettings/default
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerAdvancedThreatProtectionSetting.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect advanced threat protection settings
				Out: true, // Advanced threat protection setting changes affect server security
			}, // SQL Server Advanced Threat Protection Settings are child resources that control threat protection
		})

		// Link to DNS Aliases (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-dns-aliases/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/dnsAliases
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerDnsAlias.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect DNS aliases
				Out: true, // DNS alias changes affect server connectivity and routing
			}, // SQL Server DNS Aliases are child resources that provide alternate DNS names
		})

		// Link to Server Usages (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-usages/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/usages
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerUsage.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // SQL Server changes affect usage metrics
				Out: false, // Usage metrics are read-only and don't affect the server
			}, // SQL Server Usages are child resources that provide usage metrics
		})

		// Link to Server Operations (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-operations/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/operations
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerOperation.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // SQL Server changes affect operations history
				Out: false, // Operations history is read-only and doesn't affect the server
			}, // SQL Server Operations are child resources that provide operation history
		})

		// Link to Server Advisors (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-advisors/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/advisors
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerAdvisor.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // SQL Server changes affect advisors
				Out: false, // Advisor recommendations don't affect the server until applied
			}, // SQL Server Advisors are child resources that provide optimization recommendations
		})

		// Link to Backup Long-Term Retention Policies (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/long-term-retention-backups/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/backupLongTermRetentionPolicies
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerBackupLongTermRetentionPolicy.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect backup retention policies
				Out: true, // Backup retention policy changes affect backup management and storage
			}, // SQL Server Backup Long-Term Retention Policies are child resources that control backup retention
		})

		// Link to DevOps Audit Settings (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-devops-auditing-settings/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/devOpsAuditSettings/default
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerDevOpsAuditSetting.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect DevOps audit settings
				Out: true, // DevOps audit setting changes affect server audit logging
			}, // SQL Server DevOps Audit Settings are child resources that control DevOps audit logging
		})

		// Link to Server Trust Groups (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/server-trust-groups/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/serverTrustGroups
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerTrustGroup.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect trust groups
				Out: true, // Trust group changes affect server trust relationships
			}, // SQL Server Trust Groups are child resources that manage trust relationships
		})

		// Link to Outbound Firewall Rules (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/outbound-firewall-rules/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/outboundFirewallRules
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerOutboundFirewallRule.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // SQL Server changes affect outbound firewall rules
				Out: true, // Outbound firewall rule changes affect server outbound connectivity
			}, // SQL Server Outbound Firewall Rules are child resources that control outbound access
		})

		// Link to Private Link Resources (child resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/sql/2021-11-01/private-link-resources/list-by-server
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Sql/servers/{serverName}/privateLinkResources
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.SQLServerPrivateLinkResource.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  serverName,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // SQL Server changes affect private link resources
				Out: false, // Private link resources are metadata about available private endpoints
			}, // SQL Server Private Link Resources are child resources that provide private link metadata
		})
	}

	// External resources - extracted from IDs in the server response
	if server.Properties != nil {
		// Track processed identity resource IDs to avoid duplicates
		processedIdentityIDs := make(map[string]bool)

		// Link to Primary Managed Identity (external resource)
		// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{identityName}
		if server.Properties.PrimaryUserAssignedIdentityID != nil && *server.Properties.PrimaryUserAssignedIdentityID != "" {
			identityName := azureshared.ExtractResourceName(*server.Properties.PrimaryUserAssignedIdentityID)
			if identityName != "" {
				scope := s.DefaultScope()
				if extractedScope := azureshared.ExtractScopeFromResourceID(*server.Properties.PrimaryUserAssignedIdentityID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Managed identity deletion/modification affects server authentication and operations
						Out: false, // Server changes don't affect the managed identity itself
					}, // SQL Server depends on managed identity for authentication
				})
				processedIdentityIDs[*server.Properties.PrimaryUserAssignedIdentityID] = true
			}
		}

		// Link to all User Assigned Managed Identities (external resources)
		// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{identityName}
		if server.Identity != nil && server.Identity.UserAssignedIdentities != nil {
			for identityResourceID := range server.Identity.UserAssignedIdentities {
				// Skip if we already processed this identity (e.g., as the primary identity)
				if processedIdentityIDs[identityResourceID] {
					continue
				}
				identityName := azureshared.ExtractResourceName(identityResourceID)
				if identityName != "" {
					scope := s.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
							Method: sdp.QueryMethod_GET,
							Query:  identityName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Managed identity deletion/modification affects server authentication and operations
							Out: false, // Server changes don't affect the managed identity itself
						}, // SQL Server depends on managed identity for authentication
					})
					processedIdentityIDs[identityResourceID] = true
				}
			}
		}

		// Link to Key Vault (external resource) from KeyId encryption property
		// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get?view=rest-keyvault-keyvault-2024-11-01&tabs=HTTP
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}
		//
		// NOTE: Key Vaults can be in a different resource group than the SQL Server. However, the Key Vault URI
		// format (https://{vaultName}.vault.azure.net/keys/{keyName}/{version}) does not contain resource group information.
		// Key Vault names are globally unique within a subscription, so we use the SQL Server's scope as a best-effort
		// approach. If the Key Vault is in a different resource group, the query may fail and would need to be manually corrected
		// or the Key Vault adapter would need to support subscription-level search.
		if server.Properties != nil && server.Properties.KeyID != nil && *server.Properties.KeyID != "" {
			keyID := *server.Properties.KeyID
			// Key Vault URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
			vaultName := azureshared.ExtractVaultNameFromURI(keyID)
			if vaultName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultVault.String(),
						Method: sdp.QueryMethod_GET,
						Query:  vaultName,
						Scope:  s.DefaultScope(), // Limitation: Key Vault URI doesn't contain resource group info
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Key Vault changes (key deletion, rotation, access policy) affect the SQL Server's encryption
						Out: false, // SQL Server changes don't directly affect the Key Vault
					}, // SQL Server depends on Key Vault for customer-managed encryption keys - key changes impact encryption/decryption
				})
			}
		}

		// Link to DNS name (standard library) if FQDN is configured
		// SQL Server's fullyQualifiedDomainName represents the DNS name for the server
		if server.Properties.FullyQualifiedDomainName != nil && *server.Properties.FullyQualifiedDomainName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  *server.Properties.FullyQualifiedDomainName,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true, // DNS name changes affect server connectivity
					Out: true, // DNS names are always linked bidirectionally
				}, // DNS names are shared resources that affect multiple entities
			})
		}
	}

	return sdpItem, nil
}

func (s sqlServerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		SQLServerLookupByName,
	}
}

func (s sqlServerWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		// Child resources
		azureshared.SQLDatabase,
		azureshared.SQLElasticPool,
		azureshared.SQLServerFirewallRule,
		azureshared.SQLServerVirtualNetworkRule,
		azureshared.SQLServerKey,
		azureshared.SQLServerFailoverGroup,
		azureshared.SQLServerAdministrator,
		azureshared.SQLServerSyncGroup,
		azureshared.SQLServerSyncAgent,
		azureshared.SQLServerPrivateEndpointConnection,
		azureshared.SQLServerAuditingSetting,
		azureshared.SQLServerSecurityAlertPolicy,
		azureshared.SQLServerVulnerabilityAssessment,
		azureshared.SQLServerEncryptionProtector,
		azureshared.SQLServerBlobAuditingPolicy,
		azureshared.SQLServerAutomaticTuning,
		azureshared.SQLServerAdvancedThreatProtectionSetting,
		azureshared.SQLServerDnsAlias,
		azureshared.SQLServerUsage,
		azureshared.SQLServerOperation,
		azureshared.SQLServerAdvisor,
		azureshared.SQLServerBackupLongTermRetentionPolicy,
		azureshared.SQLServerDevOpsAuditSetting,
		azureshared.SQLServerTrustGroup,
		azureshared.SQLServerOutboundFirewallRule,
		azureshared.SQLServerPrivateLinkResource,
		// External resources
		azureshared.ManagedIdentityUserAssignedIdentity,
		azureshared.NetworkPrivateEndpoint,
		azureshared.KeyVaultVault,
		// Standard library types
		stdlib.NetworkDNS,
	)
}

func (s sqlServerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_mssql_server.name",
		},
	}
}

func (s sqlServerWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Sql/servers/read",
	}
}

func (s sqlServerWrapper) PredefinedRole() string {
	return "Reader"
}
