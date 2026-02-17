package sdp

import "github.com/google/uuid"

func (a *SourceMetadata) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *SourceHealth) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}
