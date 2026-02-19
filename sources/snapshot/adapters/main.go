package adapters

import (
	"context"
	"fmt"

	"github.com/overmindtech/cli/go/discovery"
	log "github.com/sirupsen/logrus"
)

// InitializeAdapters loads a snapshot and registers one adapter per type found
// in the snapshot data. Each adapter carries the correct category and metadata
// from the embedded adapter catalog so that the discovery engine can route
// specific-type GET/SEARCH queries to it.
func InitializeAdapters(ctx context.Context, e *discovery.Engine, snapshotSource string) error {
	snapshot, err := LoadSnapshot(ctx, snapshotSource)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	index, err := NewSnapshotIndex(snapshot)
	if err != nil {
		return fmt.Errorf("failed to build snapshot index: %w", err)
	}

	types := index.GetAllTypes()
	adapters := make([]discovery.Adapter, 0, len(types))
	for _, typ := range types {
		scopes := index.GetScopesForType(typ)
		adapters = append(adapters, NewSnapshotAdapter(index, typ, scopes))
	}

	if err := e.AddAdapters(adapters...); err != nil {
		return fmt.Errorf("failed to add snapshot adapters: %w", err)
	}

	log.WithFields(log.Fields{
		"items":    len(snapshot.GetProperties().GetItems()),
		"edges":    len(snapshot.GetProperties().GetEdges()),
		"types":    len(types),
		"adapters": len(adapters),
	}).Info("Snapshot adapters initialized successfully")

	return nil
}
