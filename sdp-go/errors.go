package sdp

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const ErrorTemplate string = `%v

ErrorType: %v
Scope: %v
SourceName: %v
ItemType: %v
ResponderName: %v`

// assert interface
var _ error = (*QueryError)(nil)

func (e *QueryError) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(e.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

// Ensure that the QueryError is seen as a valid error in golang
func (e *QueryError) Error() string {
	return fmt.Sprintf(
		ErrorTemplate,
		e.GetErrorString(),
		e.GetErrorType().String(),
		e.GetScope(),
		e.GetSourceName(),
		e.GetItemType(),
		e.GetResponderName(),
	)
}

// NewQueryError converts a regular error to a QueryError of type
// OTHER. If the input error is already a QueryError then it is preserved
func NewQueryError(err error) *QueryError {
	var sdpErr *QueryError
	if errors.As(err, &sdpErr) {
		return sdpErr
	}

	return &QueryError{
		ErrorType:   QueryError_OTHER,
		ErrorString: err.Error(),
	}
}
