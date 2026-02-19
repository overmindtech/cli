package adapters

import (
	"context"
	"fmt"
	"regexp"

	"github.com/overmindtech/cli/go/sdp-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// SnapshotAdapter is a discovery adapter that serves items of a single type
// from a snapshot. One adapter is created per type found in the snapshot so
// that the discovery engine can route specific-type GET/SEARCH queries
// correctly.
type SnapshotAdapter struct {
	index    *SnapshotIndex
	itemType string
	scopes   []string
	metadata *sdp.AdapterMetadata
}

// NewSnapshotAdapter creates a new per-type adapter backed by the shared index.
func NewSnapshotAdapter(index *SnapshotIndex, itemType string, scopes []string) *SnapshotAdapter {
	return &SnapshotAdapter{
		index:    index,
		itemType: itemType,
		scopes:   scopes,
		metadata: lookupAdapterMetadata(itemType, scopes),
	}
}

func cloneItems(items []*sdp.Item) []*sdp.Item {
	out := make([]*sdp.Item, len(items))
	for i, item := range items {
		out[i] = proto.Clone(item).(*sdp.Item)
	}
	return out
}

func (a *SnapshotAdapter) Type() string {
	return a.itemType
}

func (a *SnapshotAdapter) Name() string {
	return fmt.Sprintf("snapshot-%s", a.itemType)
}

func (a *SnapshotAdapter) Scopes() []string {
	return a.scopes
}

func (a *SnapshotAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	log.WithFields(log.Fields{
		"scope": scope,
		"type":  a.itemType,
		"query": query,
	}).Debug("SnapshotAdapter.Get called")

	// Try GUN lookup first (includes type in the GUN so it's already scoped)
	item := a.index.GetByGUN(query)
	if item != nil && item.GetType() == a.itemType {
		if scope == "*" || item.GetScope() == scope {
			return cloneItems([]*sdp.Item{item})[0], nil
		}
	}

	// Fall back to unique attribute value match within this type
	for _, candidateItem := range a.index.GetItemsByTypeAndScope(a.itemType, scope) {
		if candidateItem.UniqueAttributeValue() == query {
			return cloneItems([]*sdp.Item{candidateItem})[0], nil
		}
	}

	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: fmt.Sprintf("item not found: scope=%s, type=%s, query=%s", scope, a.itemType, query),
		Scope:       scope,
	}
}

func (a *SnapshotAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	log.WithFields(log.Fields{
		"scope": scope,
		"type":  a.itemType,
	}).Debug("SnapshotAdapter.List called")

	return cloneItems(a.index.GetItemsByTypeAndScope(a.itemType, scope)), nil
}

// Search searches for items of this type by regex on GUN and includes 1-hop
// neighbors that also match this type and scope.
func (a *SnapshotAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	log.WithFields(log.Fields{
		"scope": scope,
		"type":  a.itemType,
		"query": query,
	}).Debug("SnapshotAdapter.Search called")

	regex, err := regexp.Compile(query)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("invalid regex pattern: %v", err),
			Scope:       scope,
		}
	}

	candidates := a.index.GetItemsByTypeAndScope(a.itemType, scope)

	var primaryMatches []*sdp.Item
	for _, item := range candidates {
		if regex.MatchString(item.GloballyUniqueName()) {
			primaryMatches = append(primaryMatches, item)
		}
	}

	seen := make(map[string]bool, len(primaryMatches))
	for _, item := range primaryMatches {
		seen[item.GloballyUniqueName()] = true
	}

	var neighborMatches []*sdp.Item
	for _, item := range primaryMatches {
		for _, neighbor := range a.index.NeighborItems(item) {
			if neighbor.GetType() != a.itemType {
				continue
			}
			if scope != "*" && neighbor.GetScope() != scope {
				continue
			}
			gun := neighbor.GloballyUniqueName()
			if !seen[gun] {
				seen[gun] = true
				neighborMatches = append(neighborMatches, neighbor)
			}
		}
	}

	result := make([]*sdp.Item, 0, len(primaryMatches)+len(neighborMatches))
	result = append(result, primaryMatches...)
	result = append(result, neighborMatches...)
	return cloneItems(result), nil
}

func (a *SnapshotAdapter) Metadata() *sdp.AdapterMetadata {
	return a.metadata
}
