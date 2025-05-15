package shared

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/overmindtech/cli/sdp-go"
)

// QueryError is a helper function to convert errors into sdp.QueryError
func QueryError(err error) *sdp.QueryError {
	// Check if the error is a gRPC `not_found` error
	if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
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
