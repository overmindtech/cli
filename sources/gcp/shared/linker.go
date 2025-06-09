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
	blastPropagations                  map[shared.ItemType]map[shared.ItemType]Impact
	manualAdapterLinker                map[shared.ItemType]func(scope, selfLink string, bp *sdp.BlastPropagation) *sdp.LinkedItemQuery
	gcpResourceTypeInURLToSDPAssetType map[string]shared.ItemType
}

// NewLinker creates a new Linker instance with the provided item lookup and predefined mappings.
func NewLinker() *Linker {
	return &Linker{
		sdpAssetTypeToAdapterMeta:          SDPAssetTypeToAdapterMeta,
		gcpItemTypeToSDPAssetType:          GCPResourceTypeInURLToSDPAssetType,
		blastPropagations:                  BlastPropagations,
		manualAdapterLinker:                ManualAdapterGetLinksByAssetType,
		gcpResourceTypeInURLToSDPAssetType: GCPResourceTypeInURLToSDPAssetType,
	}
}

// Link links the FROM item TO another item based on the provided parameters.
func (l *Linker) Link(
	ctx context.Context,
	projectID string,
	fromSDPItem *sdp.Item,
	fromSDPItemType shared.ItemType,
	toItemGCPResourceName string,
	toSDPItemType shared.ItemType,
) {
	if fromSDPItemType == toSDPItemType {
		return
	}

	lf := log.Fields{
		"ovm.gcp.projectId":    projectID,
		"ovm.gcp.fromItemType": fromSDPItemType.String(),
		"ovm.gcp.toItemType":   toSDPItemType.String(),
	}

	impacts, ok := l.blastPropagations[fromSDPItemType]
	if !ok {
		log.WithContext(ctx).WithFields(lf).Warnf("there are no blast propagations for the FROM item type")
		return
	}

	impact, ok := impacts[toSDPItemType]
	if !ok {
		log.WithContext(ctx).WithFields(lf).Warnf("missing blast propagation between two item types")
		return
	}

	if linkFunc, ok := l.manualAdapterLinker[toSDPItemType]; ok {
		fromSDPItem.LinkedItemQueries = append(
			fromSDPItem.LinkedItemQueries,
			linkFunc(projectID, toItemGCPResourceName, impact.BlastPropagation),
		)
		return
	}

	sdpItemTypeMeta, ok := l.sdpAssetTypeToAdapterMeta[toSDPItemType]
	if !ok {
		// This should never happen at runtime!
		log.WithContext(ctx).WithFields(lf).Warnf(
			"could not find adapter meta for %s",
			toSDPItemType.String(),
		)
		return
	}

	parts := strings.Split(toItemGCPResourceName, "/")
	if len(parts) < 2 {
		log.WithContext(ctx).WithFields(lf).Warnf(
			"resource name is in unexpected format: %s",
			toItemGCPResourceName,
		)
		return
	}

	scope := determineScope(ctx, projectID, sdpItemTypeMeta.Scope, lf, toItemGCPResourceName, parts)
	if scope == "" {
		log.WithContext(ctx).WithFields(lf).Warnf(
			"failed to determine scope for item type %s",
			toSDPItemType.String(),
		)
		return
	}

	fromSDPItem.LinkedItemQueries = append(fromSDPItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   toSDPItemType.String(),
			Method: sdp.QueryMethod_GET,
			Query:  parts[len(parts)-1], // e.g., "my-instance", "my-network", etc.
			Scope:  scope,
		},
		BlastPropagation: impact.BlastPropagation,
	})
}

// AutoLink tries to find the item type of the TO item based on its GCP resource name.
// If the item type is identified, it links the FROM item to the TO item.
func (l *Linker) AutoLink(
	ctx context.Context,
	projectID string,
	fromSDPItem *sdp.Item,
	fromSDPItemType shared.ItemType,
	toItemGCPResourceName string,
) {
	lf := log.Fields{
		"ovm.gcp.projectId":          projectID,
		"ovm.gcp.fromItemType":       fromSDPItemType.String(),
		"ovm.gcp.toItemResourceName": toItemGCPResourceName,
	}

	parts := strings.Split(toItemGCPResourceName, "/")
	if len(parts) < 2 {
		l.tryGlobalResources(fromSDPItem, toItemGCPResourceName)
		return
	}

	toItemType, ok := l.gcpItemTypeToSDPAssetType[parts[len(parts)-2]] // e.g., "instances", "networks", etc.
	if !ok {
		log.WithContext(ctx).WithFields(lf).Warnf(
			"could not identify item type for GCP resource name %s",
			toItemGCPResourceName,
		)
		return
	}

	l.Link(ctx, projectID, fromSDPItem, fromSDPItemType, toItemGCPResourceName, toItemType)
}

func (l *Linker) tryGlobalResources(fromSDPItem *sdp.Item, toItemValue string) {
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
