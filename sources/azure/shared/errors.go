package shared

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/overmindtech/cli/sdp-go"
)

// QueryError is a helper function to convert errors into sdp.QueryError
// TODO: fix error handling to use Azure SDK error types instead of gRPC status codes in https://linear.app/overmind/issue/ENG-1830/authenticate-to-azure-using-federated-credentials
func QueryError(err error, scope string, itemType string) *sdp.QueryError {
	// Check if the error is an Azure `not_found` error
	// TODO: Replace gRPC status check with Azure SDK error type check (e.g., *azcore.ResponseError with StatusCode 404)
	if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
		return &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: err.Error(),
		}
	}

	return &sdp.QueryError{
		ErrorType:   sdp.QueryError_OTHER,
		ErrorString: err.Error(),
		SourceName:  "azure-source",
		Scope:       scope,
		ItemType:    itemType,
	}
}
