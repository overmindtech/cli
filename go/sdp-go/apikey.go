package sdp

import "github.com/google/uuid"

func (a *APIKeyMetadata) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUuid())
	if err != nil {
		return nil
	}
	return &u
}
