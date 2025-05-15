package example

import (
	"errors"

	"github.com/overmindtech/cli/sdp-go"
)

// queryError is a helper function to convert errors into sdp.QueryError
func queryError(err error) *sdp.QueryError {
	if errors.As(err, new(NotFoundError)) {
		return &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: err.Error(),
		}
	}

	return &sdp.QueryError{
		ErrorType:   sdp.QueryError_OTHER,
		ErrorString: err.Error(),
	}
}
