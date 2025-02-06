package discovery

import (
	"context"
	"sync"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

// Adapter is capable of finding information about items
//
// Adapters must implement all of the methods to satisfy this interface in order
// to be able to used as an SDP adapter. Note that the `context.Context` value
// that is passed to the Get(), List() and Search() (optional) methods needs to
// handled by each adapter individually. Adapter authors should make an effort
// ensure that expensive operations that the adapter undertakes can be cancelled
// if the context `ctx` is cancelled
type Adapter interface {
	// Type The type of items that this adapter is capable of finding
	Type() string

	// Descriptive name for the adapter, used in logging and metadata
	Name() string

	// List of scopes that this adapter is capable of find items for. If the
	// adapter supports all scopes the special value "*"
	// should be used
	Scopes() []string

	// Get Get a single item with a given scope and query. The item returned
	// should have a UniqueAttributeValue that matches the `query` parameter.
	Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error)

	// A struct that contains information about the adapter, it is used by the api-server to determine the capabilities of the adapter
	// It is mandatory for all adapters to implement this method
	Metadata() *sdp.AdapterMetadata
}

// An adapter that support the List method. This was previously part of the
// Adapter interface however it was split out to allow for the transition to
// streaming responses
type ListableAdapter interface {
	Adapter

	// List Lists all items in a given scope
	List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error)
}

// CachingAdapter Is an adapter of items that supports caching
type CachingAdapter interface {
	Adapter
	Cache() *sdpcache.Cache
}

// SearchableAdapter Is an adapter of items that supports searching
type SearchableAdapter interface {
	Adapter
	// Search executes a specific search and returns zero or many items as a
	// result (and optionally an error). The specific format of the query that
	// needs to be provided to Search is dependant on the adapter itself as each
	// adapter will respond to searches differently
	Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error)
}

// HiddenAdapter adapters that define a `Hidden()` method are able to tell whether
// or not the items they produce should be marked as hidden within the metadata.
// Hidden items will not be shown in GUIs or stored in databases and are used
// for gathering data as part of other processes such as remotely executed
// secondary adapters
type HiddenAdapter interface {
	Hidden() bool
}

// QueryResultStream is a stream of items and errors that are returned from a
// query. Adapters should send items to the stream as soon as they are
// discovered using the `SendItem` method and should send any errors that occur
// using the `SendError` method. These errors will be considered non-fatal. If
// the process encounters a fatal error it should return an error to the caller
// rather then sending one on the stream
type QueryResultStream struct {
	items       chan *sdp.Item
	errs        chan error
	itemHandler ItemHandler
	errHandler  ErrHandler
	open        bool
	wg          sync.WaitGroup
	mutex       sync.RWMutex
}

// ItemHandler is a function that can be used to handle items as they are
// received from a QueryResultStream
type ItemHandler func(item *sdp.Item)

// ErrHandler is a function that can be used to handle errors as they are
// received from a QueryResultStream
type ErrHandler func(err error)

// NewQueryResultStream creates a new QueryResultStream
func NewQueryResultStream(itemHandler ItemHandler, errHandler ErrHandler) *QueryResultStream {
	stream := &QueryResultStream{
		items:       make(chan *sdp.Item),
		errs:        make(chan error),
		itemHandler: itemHandler,
		errHandler:  errHandler,
		open:        true,
	}

	stream.wg.Add(2)
	go stream.processItems()
	go stream.processErrors()

	return stream
}

// SendItem sends an item to the stream
func (qrs *QueryResultStream) SendItem(item *sdp.Item) {
	qrs.mutex.RLock()
	defer qrs.mutex.RUnlock()
	if qrs.open {
		qrs.items <- item
	}
}

// SendError sends an error to the stream
func (qrs *QueryResultStream) SendError(err error) {
	qrs.mutex.RLock()
	defer qrs.mutex.RUnlock()
	if qrs.open {
		qrs.errs <- err
	}
}

// Close closes the stream and waits for all handlers to finish. This should be
// called by the caller, and not by adapters themselves
func (qrs *QueryResultStream) Close() {
	qrs.mutex.Lock()
	defer qrs.mutex.Unlock()
	qrs.open = false
	close(qrs.items)
	close(qrs.errs)
	qrs.wg.Wait()
}

// processItems processes items using the itemHandler
func (qrs *QueryResultStream) processItems() {
	defer qrs.wg.Done()
	for item := range qrs.items {
		qrs.itemHandler(item)
	}
}

// processErrors processes errors using the errHandler
func (qrs *QueryResultStream) processErrors() {
	defer qrs.wg.Done()
	for err := range qrs.errs {
		qrs.errHandler(err)
	}
}

// An adapter that supports streaming responses for List and Search queries
type StreamingAdapter interface {
	Adapter

	// List Lists all items in a given scope
	ListStream(ctx context.Context, scope string, ignoreCache bool, stream *QueryResultStream)

	// Search executes a specific search and returns zero or many items as a
	// result (and optionally an error). The specific format of the query that
	// needs to be provided to Search is dependant on the adapter itself as each
	// adapter will respond to searches differently
	SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream *QueryResultStream)
}
