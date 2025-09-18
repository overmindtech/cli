package discovery

import (
	"context"
	"slices"
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

// ListStreamableAdapter supports streaming for the List queries.
type ListStreamableAdapter interface {
	Adapter
	ListStream(ctx context.Context, scope string, ignoreCache bool, stream QueryResultStream)
}

// SearchStreamableAdapter supports streaming for the Search queries.
type SearchStreamableAdapter interface {
	Adapter
	SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream QueryResultStream)
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
// rather then sending one on the stream.
//
// Note that this interface does not have a `Close()` method. Clients of this
// interface are specific functions that get passed in an instance implementing
// this interface. The expectation is that those clients do not return until all
// calls into the stream have finished.
type QueryResultStream interface {
	// SendItem sends an item to the stream. This method is thread-safe, but the
	// ordering vs SendError is only guaranteed for non-overlapping calls.
	SendItem(item *sdp.Item)
	// SendError sends an Error to the stream. This method is thread-safe, but
	// the ordering vs SendItem is only guaranteed for non-overlapping calls.
	SendError(err error)
}

// QueryResultStream is a stream of items and errors that are returned from a
// query. Adapters should send items to the stream as soon as they are
// discovered using the `SendItem` method and should send any errors that occur
// using the `SendError` method. These errors will be considered non-fatal. If
// the process encounters a fatal error it should return an error to the caller
// rather then sending one on the stream
type QueryResultStreamWithHandlers struct {
	itemHandler ItemHandler
	errHandler  ErrHandler
}

// assert interface implementation
var _ QueryResultStream = (*QueryResultStreamWithHandlers)(nil)

// ItemHandler is a function that can be used to handle items as they are
// received from a QueryResultStream
type ItemHandler func(item *sdp.Item)

// ErrHandler is a function that can be used to handle errors as they are
// received from a QueryResultStream
type ErrHandler func(err error)

// NewQueryResultStream creates a new QueryResultStream that calls the provided
// handlers when items and errors are received. Note that the handlers are
// called asynchronously and need to provide for their own thread safety.
func NewQueryResultStream(itemHandler ItemHandler, errHandler ErrHandler) *QueryResultStreamWithHandlers {
	stream := &QueryResultStreamWithHandlers{
		itemHandler: itemHandler,
		errHandler:  errHandler,
	}

	return stream
}

// SendItem sends an item to the stream
func (qrs *QueryResultStreamWithHandlers) SendItem(item *sdp.Item) {
	qrs.itemHandler(item)
}

// SendError sends an error to the stream
func (qrs *QueryResultStreamWithHandlers) SendError(err error) {
	qrs.errHandler(err)
}

type RecordingQueryResultStream struct {
	streamMu sync.Mutex
	items    []*sdp.Item
	errs     []error
}

// assert interface implementation
var _ QueryResultStream = (*RecordingQueryResultStream)(nil)

func NewRecordingQueryResultStream() *RecordingQueryResultStream {
	return &RecordingQueryResultStream{
		items: []*sdp.Item{},
		errs:  []error{},
	}
}

func (r *RecordingQueryResultStream) SendItem(item *sdp.Item) {
	r.streamMu.Lock()
	defer r.streamMu.Unlock()
	r.items = append(r.items, item)
}

func (r *RecordingQueryResultStream) GetItems() []*sdp.Item {
	r.streamMu.Lock()
	defer r.streamMu.Unlock()
	return slices.Clone(r.items)
}

func (r *RecordingQueryResultStream) SendError(err error) {
	r.streamMu.Lock()
	defer r.streamMu.Unlock()
	r.errs = append(r.errs, err)
}

func (r *RecordingQueryResultStream) GetErrors() []error {
	r.streamMu.Lock()
	defer r.streamMu.Unlock()
	return slices.Clone(r.errs)
}
