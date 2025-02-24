package sdp

func (s *Snapshot) ToMap() map[string]any {
	return map[string]any{
		"metadata":   s.GetMetadata().ToMap(),
		"properties": s.GetProperties().ToMap(),
	}
}

func (sm *SnapshotMetadata) ToMap() map[string]any {
	return map[string]any{
		"UUID":    stringFromUuidBytes(sm.GetUUID()),
		"created": sm.GetCreated().AsTime(),
	}
}

func (sp *SnapshotProperties) ToMap() map[string]any {
	return map[string]any{
		"name":        sp.GetName(),
		"description": sp.GetDescription(),
		"queries":     sp.GetQueries(),
		"Items":       sp.GetItems(),
	}
}
