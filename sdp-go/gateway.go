package sdp

import (
	"encoding/hex"

	"github.com/google/uuid"
)

// Equal Returns whether two statuses are functionally equal
func (x *GatewayRequestStatus) Equal(y *GatewayRequestStatus) bool {
	if x == nil {
		if y == nil {
			return true
		} else {
			return false
		}
	}

	if (x.GetSummary() == nil || y.GetSummary() == nil) && x.GetSummary() != y.GetSummary() {
		// If one of them is nil, and they aren't both nil
		return false
	}

	if x.GetSummary() != nil && y.GetSummary() != nil {
		if x.GetSummary().GetWorking() != y.GetSummary().GetWorking() {
			return false
		}
		if x.GetSummary().GetStalled() != y.GetSummary().GetStalled() {
			return false
		}
		if x.GetSummary().GetComplete() != y.GetSummary().GetComplete() {
			return false
		}
		if x.GetSummary().GetError() != y.GetSummary().GetError() {
			return false
		}
		if x.GetSummary().GetCancelled() != y.GetSummary().GetCancelled() {
			return false
		}
		if x.GetSummary().GetResponders() != y.GetSummary().GetResponders() {
			return false
		}
	}

	if x.GetPostProcessingComplete() != y.GetPostProcessingComplete() {
		return false
	}

	return true
}

// Whether the gateway request is complete
func (x *GatewayRequestStatus) Done() bool {
	return x.GetPostProcessingComplete() && x.GetSummary().GetWorking() == 0
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *StoreBookmark) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err != nil {
			return ""
		}
		return u.String()
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *BookmarkStoreResult) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *LoadBookmark) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *BookmarkLoadResult) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *StoreSnapshot) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *SnapshotStoreResult) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *LoadSnapshot) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *SnapshotLoadResult) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *QueryStatus) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *LoadSnapshot) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}
