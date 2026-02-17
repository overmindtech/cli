package sdp

func (b *Bookmark) ToMap() map[string]any {
	return map[string]any{
		"metadata":   b.GetMetadata().ToMap(),
		"properties": b.GetProperties().ToMap(),
	}
}

func (bm *BookmarkMetadata) ToMap() map[string]any {
	return map[string]any{
		"UUID":    stringFromUuidBytes(bm.GetUUID()),
		"created": bm.GetCreated().AsTime(),
	}
}

func (bp *BookmarkProperties) ToMap() map[string]any {
	return map[string]any{
		"name":        bp.GetName(),
		"description": bp.GetDescription(),
		"queries":     bp.GetQueries(),
	}
}
