package clients

import "context"

// Pager is a generic interface for paging through Azure API results.
// T represents the response type returned by NextPage.
// This generic interface eliminates the need to define a separate Pager interface
// for each Azure client type, reducing code duplication.
type Pager[T any] interface {
	More() bool
	NextPage(ctx context.Context) (T, error)
}
