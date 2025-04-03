package sdp

import (
	"errors"
	"fmt"
	"strings"
)

// Validate ensures that GetLogRecordsRequest is valid
func (req *GetLogRecordsRequest) Validate() error {
	if req == nil {
		return errors.New("GetLogRecordsRequest is nil")
	}

	// scope has to be non-nil, non-empty string
	if strings.TrimSpace(req.GetScope()) == "" {
		return errors.New("scope has to be non-empty")
	}

	// query has to be non-nil, non-empty string
	if strings.TrimSpace(req.GetQuery()) == "" {
		return errors.New("query has to be non-empty")
	}

	// from and to have to be valid timestamps
	if req.GetFrom() == nil {
		return errors.New("from timestamp is required")
	}

	if req.GetTo() == nil {
		return errors.New("to timestamp is required")
	}

	// from has to be before or equal to to
	fromTime := req.GetFrom().AsTime()
	toTime := req.GetTo().AsTime()
	if fromTime.After(toTime) {
		return fmt.Errorf("from timestamp (%v) must be before or equal to to timestamp (%v)", fromTime, toTime)
	}

	// maxRecords is either missing or greater than zero
	if req.MaxRecords != nil && req.GetMaxRecords() <= 0 {
		return errors.New("maxRecords must be greater than zero")
	}

	return nil
}
