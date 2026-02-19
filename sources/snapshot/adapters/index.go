package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/go/sdp-go"
	log "github.com/sirupsen/logrus"
)

// SnapshotIndex maintains in-memory indices for efficient snapshot querying
type SnapshotIndex struct {
	// All items in the snapshot
	allItems []*sdp.Item

	// Index by GloballyUniqueName for fast GET lookups
	byGUN map[string]*sdp.Item

	// Index by type and scope for filtering
	byTypeScope map[string]map[string][]*sdp.Item

	// Edges from the snapshot (for future use)
	edges []*sdp.Edge
}

// NewSnapshotIndex builds indices from a snapshot
func NewSnapshotIndex(snapshot *sdp.Snapshot) (*SnapshotIndex, error) {
	if snapshot == nil || snapshot.GetProperties() == nil {
		return nil, fmt.Errorf("snapshot or properties is nil")
	}

	items := snapshot.GetProperties().GetItems()
	edges := snapshot.GetProperties().GetEdges()

	index := &SnapshotIndex{
		allItems:    items,
		byGUN:       make(map[string]*sdp.Item),
		byTypeScope: make(map[string]map[string][]*sdp.Item),
		edges:       edges,
	}

	// Build indices
	for _, item := range items {
		gun := item.GloballyUniqueName()
		index.byGUN[gun] = item

		itemType := item.GetType()
		scope := item.GetScope()

		if index.byTypeScope[itemType] == nil {
			index.byTypeScope[itemType] = make(map[string][]*sdp.Item)
		}
		index.byTypeScope[itemType][scope] = append(index.byTypeScope[itemType][scope], item)
	}

	// Hydrate each item's LinkedItems from the snapshot edges so that
	// callers (explore view, etc.) see the graph relationships directly on
	// the returned items instead of having to cross-reference the separate
	// edge list.
	index.hydrateLinkedItems()

	log.WithFields(log.Fields{
		"total_items": len(items),
		"total_edges": len(edges),
		"types":       len(index.byTypeScope),
	}).Info("Snapshot index built")

	return index, nil
}

// hydrateLinkedItems populates each item's LinkedItems field from the snapshot
// edges. For each edge, the item matching edge.From gets a LinkedItem pointing
// to edge.To (with blast propagation). Edges whose From item is not in the
// snapshot are skipped.
func (idx *SnapshotIndex) hydrateLinkedItems() {
	// Build a map from item reference key → existing LinkedItem targets so
	// we don't add duplicates when the item already carries some LinkedItems.
	type refKey struct {
		scope, typ, uav string
	}
	existingLinks := make(map[refKey]map[refKey]bool)

	for _, item := range idx.allItems {
		key := refKey{item.GetScope(), item.GetType(), item.UniqueAttributeValue()}
		set := make(map[refKey]bool)
		for _, li := range item.GetLinkedItems() {
			r := li.GetItem()
			if r != nil {
				set[refKey{r.GetScope(), r.GetType(), r.GetUniqueAttributeValue()}] = true
			}
		}
		existingLinks[key] = set
	}

	for _, edge := range idx.edges {
		from := edge.GetFrom()
		to := edge.GetTo()
		if from == nil || to == nil {
			continue
		}

		item := idx.GetByReference(from)
		if item == nil {
			continue
		}

		fromKey := refKey{item.GetScope(), item.GetType(), item.UniqueAttributeValue()}
		toKey := refKey{to.GetScope(), to.GetType(), to.GetUniqueAttributeValue()}

		if existingLinks[fromKey][toKey] {
			continue
		}

		item.LinkedItems = append(item.LinkedItems, &sdp.LinkedItem{
			Item:             to,
			BlastPropagation: edge.GetBlastPropagation(),
		})
		existingLinks[fromKey][toKey] = true
	}
}

// GetAllItems returns all items in the snapshot
func (idx *SnapshotIndex) GetAllItems() []*sdp.Item {
	return idx.allItems
}

// GetByGUN retrieves an item by its GloballyUniqueName
func (idx *SnapshotIndex) GetByGUN(gun string) *sdp.Item {
	return idx.byGUN[gun]
}

// GetByReference retrieves an item by its Reference using the GUN index.
func (idx *SnapshotIndex) GetByReference(ref *sdp.Reference) *sdp.Item {
	if ref == nil {
		return nil
	}
	return idx.byGUN[ref.GloballyUniqueName()]
}

// GetAllTypes returns all unique types in the snapshot
func (idx *SnapshotIndex) GetAllTypes() []string {
	types := make([]string, 0, len(idx.byTypeScope))
	for itemType := range idx.byTypeScope {
		types = append(types, itemType)
	}
	return types
}

// GetScopesForType returns all unique scopes that contain items of the given type.
func (idx *SnapshotIndex) GetScopesForType(itemType string) []string {
	scopeMap, ok := idx.byTypeScope[itemType]
	if !ok {
		return nil
	}
	scopes := make([]string, 0, len(scopeMap))
	for s := range scopeMap {
		scopes = append(scopes, s)
	}
	return scopes
}

// GetItemsByTypeAndScope returns items matching the given type and scope.
// A wildcard ("*") scope returns all items of that type.
func (idx *SnapshotIndex) GetItemsByTypeAndScope(itemType, scope string) []*sdp.Item {
	scopeMap, ok := idx.byTypeScope[itemType]
	if !ok {
		return nil
	}
	if scope == "*" {
		var all []*sdp.Item
		for _, items := range scopeMap {
			all = append(all, items...)
		}
		return all
	}
	return scopeMap[scope]
}

// EdgesFrom returns all edges whose From reference equals ref.
func (idx *SnapshotIndex) EdgesFrom(ref *sdp.Reference) []*sdp.Edge {
	if ref == nil {
		return nil
	}
	var out []*sdp.Edge
	for _, e := range idx.edges {
		if e.GetFrom() != nil && e.GetFrom().IsEqual(ref) {
			out = append(out, e)
		}
	}
	return out
}

// EdgesTo returns all edges whose To reference equals ref.
func (idx *SnapshotIndex) EdgesTo(ref *sdp.Reference) []*sdp.Edge {
	if ref == nil {
		return nil
	}
	var out []*sdp.Edge
	for _, e := range idx.edges {
		if e.GetTo() != nil && e.GetTo().IsEqual(ref) {
			out = append(out, e)
		}
	}
	return out
}

// NeighborItems returns items that are connected to the given item by any edge
// (as From or To). Each item is returned at most once. Items not present in
// the snapshot are skipped.
func (idx *SnapshotIndex) NeighborItems(item *sdp.Item) []*sdp.Item {
	if item == nil {
		return nil
	}
	ref := item.Reference()
	seen := make(map[string]bool)
	var out []*sdp.Item
	for _, e := range idx.EdgesFrom(ref) {
		if e.GetTo() != nil {
			other := idx.GetByReference(e.GetTo())
			if other != nil {
				gun := other.GloballyUniqueName()
				if !seen[gun] {
					seen[gun] = true
					out = append(out, other)
				}
			}
		}
	}
	for _, e := range idx.EdgesTo(ref) {
		if e.GetFrom() != nil {
			other := idx.GetByReference(e.GetFrom())
			if other != nil {
				gun := other.GloballyUniqueName()
				if !seen[gun] {
					seen[gun] = true
					out = append(out, other)
				}
			}
		}
	}
	return out
}

