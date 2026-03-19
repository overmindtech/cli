package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var (
	NetworkDNSRecordSetLookupByRecordType = shared.NewItemTypeLookup("recordType", azureshared.NetworkDNSRecordSet)
	NetworkDNSRecordSetLookupByName       = shared.NewItemTypeLookup("name", azureshared.NetworkDNSRecordSet)
)

type networkDNSRecordSetWrapper struct {
	client clients.RecordSetsClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkDNSRecordSet(client clients.RecordSetsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkDNSRecordSetWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkDNSRecordSet,
		),
	}
}

// recordTypeFromResourceType extracts the DNS record type (e.g. "A", "AAAA") from the ARM resource type (e.g. "Microsoft.Network/dnszones/A").
func recordTypeFromResourceType(resourceType string) string {
	if resourceType == "" {
		return ""
	}
	parts := strings.Split(resourceType, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// ref: https://learn.microsoft.com/en-us/rest/api/dns/record-sets/get?view=rest-dns-2018-05-01&tabs=HTTP
func (n networkDNSRecordSetWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 3 {
		return nil, azureshared.QueryError(errors.New("Get requires 3 query parts: zoneName, recordType, and relativeRecordSetName"), scope, n.Type())
	}
	zoneName := queryParts[0]
	recordTypeStr := queryParts[1]
	relativeRecordSetName := queryParts[2]
	if zoneName == "" || recordTypeStr == "" || relativeRecordSetName == "" {
		return nil, azureshared.QueryError(errors.New("zoneName, recordType and relativeRecordSetName cannot be empty"), scope, n.Type())
	}
	recordType := armdns.RecordType(recordTypeStr)

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, zoneName, relativeRecordSetName, recordType, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	return n.azureRecordSetToSDPItem(&resp.RecordSet, zoneName, scope)
}

func (n networkDNSRecordSetWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkZoneLookupByName,
		NetworkDNSRecordSetLookupByRecordType,
		NetworkDNSRecordSetLookupByName,
	}
}

func (n networkDNSRecordSetWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Search requires 1 query part: zoneName"), scope, n.Type())
	}
	zoneName := queryParts[0]
	if zoneName == "" {
		return nil, azureshared.QueryError(errors.New("zoneName cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListAllByDNSZonePager(rgScope.ResourceGroup, zoneName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}
		for _, rs := range page.Value {
			if rs == nil || rs.Name == nil {
				continue
			}
			item, sdpErr := n.azureRecordSetToSDPItem(rs, zoneName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkDNSRecordSetWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: zoneName"), scope, n.Type()))
		return
	}
	zoneName := queryParts[0]
	if zoneName == "" {
		stream.SendError(azureshared.QueryError(errors.New("zoneName cannot be empty"), scope, n.Type()))
		return
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListAllByDNSZonePager(rgScope.ResourceGroup, zoneName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, rs := range page.Value {
			if rs == nil || rs.Name == nil {
				continue
			}
			item, sdpErr := n.azureRecordSetToSDPItem(rs, zoneName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkDNSRecordSetWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{NetworkZoneLookupByName},
	}
}

func (n networkDNSRecordSetWrapper) azureRecordSetToSDPItem(rs *armdns.RecordSet, zoneName, scope string) (*sdp.Item, *sdp.QueryError) {
	if rs.Name == nil {
		return nil, azureshared.QueryError(errors.New("record set name is nil"), scope, n.Type())
	}
	relativeName := *rs.Name
	recordTypeStr := ""
	if rs.Type != nil {
		recordTypeStr = recordTypeFromResourceType(*rs.Type)
	}
	if recordTypeStr == "" {
		return nil, azureshared.QueryError(errors.New("record set type is nil or invalid"), scope, n.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(rs, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	uniqueAttr := shared.CompositeLookupKey(zoneName, recordTypeStr, relativeName)
	if err := attributes.Set("uniqueAttr", uniqueAttr); err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkDNSRecordSet.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link to parent DNS zone
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkZone.String(),
			Method: sdp.QueryMethod_GET,
			Query:  zoneName,
			Scope:  scope,
		},
	})

	// Link to DNS name (standard library) from FQDN if present
	if rs.Properties != nil && rs.Properties.Fqdn != nil && *rs.Properties.Fqdn != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  *rs.Properties.Fqdn,
				Scope:  "global",
			},
		})
	}

	// LinkedItemQueries for IP addresses and DNS names in record data
	if rs.Properties != nil {
		seenIPs := make(map[string]struct{})
		seenDNS := make(map[string]struct{})

		// A records (IPv4) -> stdlib.NetworkIP, GET, global
		for _, a := range rs.Properties.ARecords {
			if a != nil && a.IPv4Address != nil && *a.IPv4Address != "" {
				ip := *a.IPv4Address
				if _, seen := seenIPs[ip]; !seen {
					seenIPs[ip] = struct{}{}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  ip,
							Scope:  "global",
						},
					})
				}
			}
		}
		// AAAA records (IPv6) -> stdlib.NetworkIP, GET, global
		for _, aaaa := range rs.Properties.AaaaRecords {
			if aaaa != nil && aaaa.IPv6Address != nil && *aaaa.IPv6Address != "" {
				ip := *aaaa.IPv6Address
				if _, seen := seenIPs[ip]; !seen {
					seenIPs[ip] = struct{}{}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  ip,
							Scope:  "global",
						},
					})
				}
			}
		}

		// DNS names in record data -> stdlib.NetworkDNS, SEARCH, global
		appendDNSLink := func(name string) {
			if name == "" {
				return
			}
			if _, seen := seenDNS[name]; !seen {
				seenDNS[name] = struct{}{}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  name,
						Scope:  "global",
					},
				})
			}
		}
		if rs.Properties.CnameRecord != nil && rs.Properties.CnameRecord.Cname != nil && *rs.Properties.CnameRecord.Cname != "" {
			appendDNSLink(*rs.Properties.CnameRecord.Cname)
		}
		for _, mx := range rs.Properties.MxRecords {
			if mx != nil && mx.Exchange != nil && *mx.Exchange != "" {
				appendDNSLink(*mx.Exchange)
			}
		}
		for _, ns := range rs.Properties.NsRecords {
			if ns != nil && ns.Nsdname != nil && *ns.Nsdname != "" {
				appendDNSLink(*ns.Nsdname)
			}
		}
		for _, ptr := range rs.Properties.PtrRecords {
			if ptr != nil && ptr.Ptrdname != nil && *ptr.Ptrdname != "" {
				appendDNSLink(*ptr.Ptrdname)
			}
		}
		// SOA Host is the authoritative name server (DNS name). SOA Email is an email in DNS
		// notation (e.g. admin.example.com = admin@example.com), not a resolvable hostname.
		if rs.Properties.SoaRecord != nil && rs.Properties.SoaRecord.Host != nil && *rs.Properties.SoaRecord.Host != "" {
			appendDNSLink(*rs.Properties.SoaRecord.Host)
		}
		// Only "issue" and "issuewild" CAA values are DNS names (CA domain). "iodef" values
		// are URLs (e.g. mailto: or https:) and must not be passed to appendDNSLink.
		for _, caa := range rs.Properties.CaaRecords {
			if caa == nil || caa.Tag == nil || caa.Value == nil || *caa.Value == "" {
				continue
			}
			tag := *caa.Tag
			if tag != "issue" && tag != "issuewild" {
				continue
			}
			appendDNSLink(*caa.Value)
		}
		for _, srv := range rs.Properties.SrvRecords {
			if srv != nil && srv.Target != nil && *srv.Target != "" {
				appendDNSLink(*srv.Target)
			}
		}

		// TargetResource (Azure resource ID) -> link to referenced resource.
		// Pass the composite lookup key (extracted query parts) so the target adapter's Get
		// receives the expected parts when the transformer splits by QuerySeparator; it does
		// not parse full resource IDs for linked GET queries.
		// For types in pathKeysMap we use ExtractPathParamsFromResourceIDByType; for simple
		// single-name resources (e.g. public IP, Traffic Manager) we fall back to ExtractResourceName.
		if rs.Properties.TargetResource != nil && rs.Properties.TargetResource.ID != nil && *rs.Properties.TargetResource.ID != "" {
			targetID := *rs.Properties.TargetResource.ID
			linkScope := azureshared.ExtractScopeFromResourceID(targetID)
			if linkScope == "" {
				linkScope = scope
			}
			itemType := azureshared.ItemTypeFromLinkedResourceID(targetID)
			if itemType != "" {
				queryParts := azureshared.ExtractPathParamsFromResourceIDByType(itemType, targetID)
				var query string
				if queryParts != nil {
					query = shared.CompositeLookupKey(queryParts...)
				} else {
					// Simple resource type (no pathKeysMap): use resource name as single query part
					query = azureshared.ExtractResourceName(targetID)
				}
				if query != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   itemType,
							Method: sdp.QueryMethod_GET,
							Query:  query,
							Scope:  linkScope,
						},
					})
				}
			}
		}
	}

	// Health from provisioning state
	if rs.Properties != nil && rs.Properties.ProvisioningState != nil {
		switch *rs.Properties.ProvisioningState {
		case "Succeeded":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case "Creating", "Updating", "Deleting":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Failed", "Canceled":
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return sdpItem, nil
}

func (n networkDNSRecordSetWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkZone,
		stdlib.NetworkDNS,
		stdlib.NetworkIP,
	)
}

func (n networkDNSRecordSetWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return nil
}

func (n networkDNSRecordSetWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/dnszones/*/read",
	}
}

func (n networkDNSRecordSetWrapper) PredefinedRole() string {
	return "Reader"
}
