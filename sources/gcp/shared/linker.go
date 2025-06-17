package shared

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// ItemTypeMeta holds metadata about an item type.
type ItemTypeMeta struct {
	GCPAssetType      string
	SDPAssetType      shared.ItemType
	SDPCategory       sdp.AdapterCategory
	SelfLink          string
	TerraformMappings []*sdp.TerraformMapping
}

// ItemLookup is a map that associates item type keys (strings) with their metadata.
type ItemLookup map[string]ItemTypeMeta

// Linker is responsible for linking items based on their types and relationships.
type Linker struct {
	sdpAssetTypeToAdapterMeta          map[shared.ItemType]AdapterMeta
	gcpItemTypeToSDPAssetType          map[string]shared.ItemType
	explicitBlastPropagations          map[shared.ItemType]map[string]*Impact
	manualAdapterLinker                map[shared.ItemType]func(scope, fromItemScope, query string, bp *sdp.BlastPropagation) *sdp.LinkedItemQuery
	gcpResourceTypeInURLToSDPAssetType map[string]shared.ItemType
}

// NewLinker creates a new Linker instance with the provided item lookup and predefined mappings.
func NewLinker() *Linker {
	return &Linker{
		sdpAssetTypeToAdapterMeta:          SDPAssetTypeToAdapterMeta,
		gcpItemTypeToSDPAssetType:          GCPResourceTypeInURLToSDPAssetType,
		explicitBlastPropagations:          BlastPropagations,
		manualAdapterLinker:                ManualAdapterLinksByAssetType,
		gcpResourceTypeInURLToSDPAssetType: GCPResourceTypeInURLToSDPAssetType,
	}
}

// AutoLink tries to find the item type of the TO item based on its GCP resource name.
// If the item type is identified, it links the FROM item to the TO item.
func (l *Linker) AutoLink(ctx context.Context, projectID string, fromSDPItem *sdp.Item, fromSDPItemType shared.ItemType, toItemGCPResourceName string, keys []string) {
	key := strings.Join(keys, ".")

	if key == "selfLink" || key == "name" {
		return
	}

	lf := log.Fields{
		"ovm.gcp.projectId":          projectID,
		"ovm.gcp.fromItemType":       fromSDPItemType.String(),
		"ovm.gcp.toItemResourceName": toItemGCPResourceName,
		"ovm.gcp.key":                key,
	}

	impacts, ok := l.explicitBlastPropagations[fromSDPItemType]
	if !ok {
		log.WithContext(ctx).WithFields(lf).Warnf("there are no blast propagations for the FROM item type")
		return
	}

	impact, ok := impacts[key]
	if !ok {
		if strings.Contains(toItemGCPResourceName, "/") {
			// There is a high chance that the item type is not recognized, so we log a warning.
			log.WithContext(ctx).WithFields(lf).Warnf("missing blast propagation between two item types")
		}
		return
	}

	if linkFunc, ok := l.manualAdapterLinker[impact.ToSDPITemType]; ok {
		linkedItemQuery := linkFunc(projectID, fromSDPItem.GetScope(), toItemGCPResourceName, impact.BlastPropagation)
		if linkedItemQuery == nil {
			log.WithContext(ctx).WithFields(lf).Warn(
				"manual adapter linker failed to create a linked item query",
			)
			return
		}

		fromSDPItem.LinkedItemQueries = append(
			fromSDPItem.LinkedItemQueries,
			linkedItemQuery,
		)
		return
	}

	toSDPItemMeta, ok := l.sdpAssetTypeToAdapterMeta[impact.ToSDPITemType]
	if !ok {
		// This should never happen at runtime!
		log.WithContext(ctx).WithFields(lf).Warnf(
			"could not find adapter meta for %s",
			impact.ToSDPITemType.String(),
		)
		return
	}

	var scope string
	var query string
	switch toSDPItemMeta.Scope {
	case ScopeProject:
		scope = projectID
		values := ExtractPathParams(toItemGCPResourceName, toSDPItemMeta.UniqueAttributeKeys...)
		if len(values) != len(toSDPItemMeta.UniqueAttributeKeys) {
			log.WithContext(ctx).WithFields(lf).Warnf(
				"resource name is in unexpected format for project item",
			)
			return
		}
		query = strings.Join(values, shared.QuerySeparator)
	case ScopeRegional:
		keysToExtract := append(toSDPItemMeta.UniqueAttributeKeys, "regions")
		values := ExtractPathParams(toItemGCPResourceName, keysToExtract...)
		if len(values) != len(keysToExtract) {
			log.WithContext(ctx).WithFields(lf).Warnf(
				"resource name is in unexpected format for regional item",
			)
			return
		}
		scope = fmt.Sprintf("%s.%s", projectID, values[len(values)-1])      // e.g., "my-project.my-region"
		query = strings.Join(values[:len(values)-1], shared.QuerySeparator) // e.g., "my-instance" or "my-network"
	case ScopeZonal:
		keysToExtract := append(toSDPItemMeta.UniqueAttributeKeys, "zones")
		values := ExtractPathParams(toItemGCPResourceName, keysToExtract...)
		if len(values) != len(keysToExtract) {
			log.WithContext(ctx).WithFields(lf).Warnf(
				"resource name is in unexpected format for zonal item",
			)
			return
		}
		scope = fmt.Sprintf("%s.%s", projectID, values[len(values)-1])      // e.g., "my-project.my-zone"
		query = strings.Join(values[:len(values)-1], shared.QuerySeparator) // e.g., "my-instance" or "my-network"

	default:
		log.WithContext(ctx).WithFields(lf).Errorf("unsupported scope %s", toSDPItemMeta.Scope)
		return
	}

	fromSDPItem.LinkedItemQueries = append(fromSDPItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   impact.ToSDPITemType.String(),
			Method: sdp.QueryMethod_GET,
			Query:  query,
			Scope:  scope,
		},
		BlastPropagation: impact.BlastPropagation,
	})
}

func (l *Linker) tryGlobalResources(fromSDPItem *sdp.Item, toItemValue string) { //nolint: unused
	if isIPAddress(toItemValue) {
		fromSDPItem.LinkedItemQueries = append(fromSDPItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ip",
				Method: sdp.QueryMethod_GET,
				Query:  toItemValue,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	if isDNSName(toItemValue) {
		fromSDPItem.LinkedItemQueries = append(fromSDPItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  toItemValue,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}
}

func isIPAddress(s string) bool {
	return net.ParseIP(s) != nil
}

func isDNSName(s string) bool {
	if isIPAddress(s) {
		return false
	}

	// Normalize to lowercase to ensure case-insensitivity and trim trailing dot
	s = strings.TrimSuffix(strings.ToLower(s), ".")
	// Must contain at least one dot and only valid DNS characters
	if strings.Contains(s, ".") && dnsNameRegexp.MatchString(s) {
		return true
	}
	return false
}

// Source:
// https://stackoverflow.com/questions/10306690/what-is-a-regular-expression-which-will-match-a-valid-domain-name-without-a-subd/30007882#30007882
var dnsNameRegexp = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`)

// determineScope determines the scope of the GCP resource based on its type and parts.
// If it fails to determine the scope.
func determineScope(ctx context.Context, projectID string, scope Scope, lf log.Fields, toItemGCPResourceName string, parts []string) string {
	switch scope {
	case ScopeProject:
		return projectID
	case ScopeRegional:
		if len(parts) < 4 {
			log.WithContext(ctx).WithFields(lf).Warnf(
				"resource name is in unexpected format for regional item %s",
				toItemGCPResourceName,
			)
			return ""
		}
		return fmt.Sprintf("%s.%s", projectID, parts[len(parts)-3])
	case ScopeZonal:
		if len(parts) < 4 {
			log.WithContext(ctx).WithFields(lf).Warnf(
				"resource name is in unexpected format for zonal item %s",
				toItemGCPResourceName,
			)
			return ""
		}
		return fmt.Sprintf("%s.%s", projectID, parts[len(parts)-3])
	default:
		return ""
	}
}
