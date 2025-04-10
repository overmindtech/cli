package sdp

import (
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
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

	if req.GetMaxRecords() < 0 {
		return errors.New("maxRecords must be greater than or equal to zero")
	}

	return nil
}

// NewUpstreamSourceError creates a new SourceError with the given message and error
func NewUpstreamSourceError(code connect.Code, message string) *SourceError {
	return &SourceError{
		Code:     SourceError_Code(code), //nolint:gosec
		Message:  message,
		Upstream: true,
	}
}

// NewLocalSourceError creates a new SourceError with the given message and error, indicating a local (non-upstream) error
func NewLocalSourceError(code connect.Code, message string) *SourceError {
	return &SourceError{
		Code:     SourceError_Code(code), //nolint:gosec
		Message:  message,
		Upstream: false,
	}
}

// assert interface implementation
var _ error = (*SourceError)(nil)

// Error implements the error interface for SourceError
func (e *SourceError) Error() string {
	if e.GetUpstream() {
		return fmt.Sprintf("Upstream Error: %s", e.GetMessage())
	}
	return fmt.Sprintf("Source Error: %s", e.GetMessage())
}
