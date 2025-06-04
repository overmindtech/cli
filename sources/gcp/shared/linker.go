package shared

import (
	"context"
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
	AllKnownItems                      ItemLookup
	blastPropagations                  map[shared.ItemType]map[shared.ItemType]Impact
	manualAdapterLinker                map[shared.ItemType]func(scope, selfLink string, bp *sdp.BlastPropagation) *sdp.LinkedItemQuery
	gcpResourceTypeInURLToSDPAssetType map[string]shared.ItemType
}

// NewLinker creates a new Linker instance with the provided item lookup and predefined mappings.
func NewLinker(allKnownItems ItemLookup) *Linker {
	return &Linker{
		AllKnownItems:                      allKnownItems,
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

	fromSDPItem.LinkedItemQueries = append(fromSDPItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   toSDPItemType.String(),
			Method: sdp.QueryMethod_GET, // Dynamic adapters use GET method.
			Query:  toItemGCPResourceName,
			Scope:  projectID, // This is a dynamic adapter, so we use the project ID as the scope.
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
	toSDPItemType, ok := l.identifyItemType(ctx, toItemGCPResourceName)
	if ok {
		l.Link(ctx, projectID, fromSDPItem, fromSDPItemType, toItemGCPResourceName, *toSDPItemType)
		return
	}

	if isIPAddress(toItemGCPResourceName) {
		fromSDPItem.LinkedItemQueries = append(fromSDPItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ip",
				Method: sdp.QueryMethod_GET,
				Query:  toItemGCPResourceName,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	if isDNSName(toItemGCPResourceName) {
		fromSDPItem.LinkedItemQueries = append(fromSDPItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  toItemGCPResourceName,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}
}

// identifyItemType identifies the item type based on the GCP resource name.
func (l *Linker) identifyItemType(ctx context.Context, gcpResourceName string) (*shared.ItemType, bool) {
	lf := log.Fields{
		"ovm.GCPResourceName": gcpResourceName,
	}

	if itemType, ok := l.AllKnownItems[gcpResourceName]; ok {
		return &itemType.SDPAssetType, true
	}

	if strings.HasPrefix(gcpResourceName, "http://") || strings.HasPrefix(gcpResourceName, "https://") ||
		strings.Contains(gcpResourceName, "projects/") ||
		strings.Contains(gcpResourceName, "zones/") ||
		strings.Contains(gcpResourceName, "regions/") {
		parts := strings.Split(gcpResourceName, "/")
		if len(parts) > 1 {
			// We are extracting the GCP resource type from the URL.
			// Example: https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/instances/my-instance
			// `instances` is the item type identifier.
			itemTypeIdentifier := parts[len(parts)-2]

			// ignore region and zone links, because we don't have adapters and we are not interested in them (yet).
			if itemTypeIdentifier == "zones" || itemTypeIdentifier == "regions" {
				return nil, false
			}

			if toSDPAssetType, ok := l.gcpResourceTypeInURLToSDPAssetType[itemTypeIdentifier]; ok {
				return &toSDPAssetType, true
			}

			log.WithContext(ctx).WithFields(lf).Warnf("failed to identify item type for a potentially linked item")
		}
	}

	return nil, false
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
