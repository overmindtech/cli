package aws

import (
	"errors"
	"slices"

	awsHttp "github.com/aws/smithy-go/transport/http"

	"github.com/overmindtech/cli/sdp-go"
)

// queryError takes an error and returns a sdp.QueryError.
func queryError(err error) *sdp.QueryError {
	var responseErr *awsHttp.ResponseError
	if errors.As(err, &responseErr) {
		// If the input is bad, access is denied, or the thing wasn't found then
		// we should assume that it is not exist for this adapter
		if slices.Contains([]int{400, 403, 404}, responseErr.HTTPStatusCode()) {
			return &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: err.Error(),
			}
		}
	}

	return &sdp.QueryError{
		ErrorType:   sdp.QueryError_OTHER,
		ErrorString: err.Error(),
	}
}
