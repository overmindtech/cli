package sdp

import "github.com/google/uuid"

// ToMap converts a Snapshot to a map for serialization.
func (s *Snapshot) ToMap() map[string]any {
	return map[string]any{
		"metadata":   s.GetMetadata().ToMap(),
		"properties": s.GetProperties().ToMap(),
	}
}

// ToMap converts SnapshotMetadata to a map for serialization.
func (sm *SnapshotMetadata) ToMap() map[string]any {
	return map[string]any{
		"UUID":    stringFromUuidBytes(sm.GetUUID()),
		"created": sm.GetCreated().AsTime(),
	}
}

// GetUUIDParsed returns the parsed UUID from the SnapshotMetadata, or nil if invalid.
func (sm *SnapshotMetadata) GetUUIDParsed() *uuid.UUID {
	if sm == nil {
		return nil
	}
	u, err := uuid.FromBytes(sm.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

// ToMap converts SnapshotProperties to a map for serialization.
func (sp *SnapshotProperties) ToMap() map[string]any {
	return map[string]any{
		"name":        sp.GetName(),
		"description": sp.GetDescription(),
		"queries":     sp.GetQueries(),
		"Items":       sp.GetItems(),
	}
}
