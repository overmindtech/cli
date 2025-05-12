package adapterhelpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"buf.build/go/protovalidate"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

const DefaultCacheDuration = 1 * time.Hour

// DescribeOnlyAdapter Generates a adapter for AWS APIs that only use a `Describe`
// function for both List and Get operations. EC2 is a good example of this,
// where running Describe with no params returns everything, but params can be
// supplied to reduce the number of results.
type DescribeOnlyAdapter[Input InputType, Output OutputType, ClientStruct ClientStructType, Options OptionsType] struct {
	MaxResultsPerPage int32  // Max results per page when making API queries
	ItemType          string // The type of items that will be returned
	AdapterMetadata   *sdp.AdapterMetadata

	CacheDuration time.Duration   // How long to cache items for
	cache         *sdpcache.Cache // The sdpcache of this adapter
	cacheInitMu   sync.Mutex      // Mutex to ensure cache is only initialised once

	// The function that should be used to describe the resources that this
	// adapter is related to
	DescribeFunc func(ctx context.Context, client ClientStruct, input Input) (Output, error)

	// A function that returns the input object that will be passed to
	// DescribeFunc for a GET request
	InputMapperGet func(scope, query string) (Input, error)

	// A function that returns the input object that will be passed to
	// DescribeFunc for a LIST request
	InputMapperList func(scope string) (Input, error)

	// A function that maps a search query to the required input. If this is
	// unset then a search request will default to searching by ARN
	InputMapperSearch func(ctx context.Context, client ClientStruct, scope string, query string) (Input, error)

	// A PostSearchFilter, if set, will be called after the search has been
	// completed. This can be used to filter the results of the search before
	// they are returned to the user, based on the query. This is used in
	// situations where the underlying API doesn't allow for granular enough
	// searching to match a given query string, and we need to apply some
	// additional filtering to the response.
	//
	// A good example if this is allowing users to search using ARNs that
	// contain IAM-Style wildcards. Since IAM is enforced *after* a query is
	// run, most APIs don't provide detailed enough search options to completely
	// replicate this functionality in the query, and instead we need to filter
	// the results ourselves.
	//
	// This will only be applied when the InputMapperSearch function is also set
	PostSearchFilter func(ctx context.Context, query string, items []*sdp.Item) ([]*sdp.Item, error)

	// A function that returns a paginator for this API. If this is nil, we will
	// assume that the API is not paginated e.g.
	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#using-paginators
	PaginatorBuilder func(client ClientStruct, params Input) Paginator[Output, Options]

	// A function that returns a slice of items for a given output. The scope
	// and input are passed in on order to assist in creating the items if
	// needed, but primarily this function should iterate over the output and
	// create new items for each result
	OutputMapper func(ctx context.Context, client ClientStruct, scope string, input Input, output Output) ([]*sdp.Item, error)

	// The region that this adapter is configured in, each adapter can only be
	// configured for one region. Getting data from many regions requires a
	// adapter per region. This is used in the scope of returned resources
	Region string

	// AccountID The id of the account that is being used. This is used by
	// sources as the first element in the scope
	AccountID string

	// Client The AWS client to use when making requests
	Client ClientStruct

	// UseListForGet If true, the adapter will use the List function to get items
	// This option should be used when the Describe function does not support
	// getting a single item by ID. The adapter will then filter the items
	// itself.
	// InputMapperGet should still be defined. It will be used to create the
	// input for the List function. The output of the List function will be
	// filtered by the adapter to find the item with the matching ID.
	// See the directconnect-virtual-gateway adapter for an example of this.
	UseListForGet bool
}

// Returns the duration that items should be cached for. This will use the
// `CacheDuration` for this adapter if set, otherwise it will use the default
// duration of 1 hour
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) cacheDuration() time.Duration {
	if s.CacheDuration == 0 {
		return DefaultCacheDuration
	}

	return s.CacheDuration
}

func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
	}
}

func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
}

// Validate Checks that the adapter is correctly set up and returns an error if
// not
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Validate() error {
	if s.DescribeFunc == nil {
		return errors.New("adapter describe func is nil")
	}

	if s.MaxResultsPerPage == 0 {
		s.MaxResultsPerPage = DefaultMaxResultsPerPage
	}

	if s.InputMapperGet == nil {
		return errors.New("adapter get input mapper is nil")
	}

	if s.OutputMapper == nil {
		return errors.New("adapter output mapper is nil")
	}

	return protovalidate.Validate(s.AdapterMetadata)
}

// Paginated returns whether or not this adapter is using a paginated API
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Paginated() bool {
	return s.PaginatorBuilder != nil
}

func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Type() string {
	return s.ItemType
}

func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Name() string {
	return fmt.Sprintf("%v-adapter", s.ItemType)
}

func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Metadata() *sdp.AdapterMetadata {
	return s.AdapterMetadata
}

// List of scopes that this adapter is capable of find items for. This will be
// in the format {accountID}.{region}
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Scopes() []string {
	return []string{
		FormatScope(s.AccountID, s.Region),
	}
}

// Get Get a single item with a given scope and query. The item returned
// should have a UniqueAttributeValue that matches the `query` parameter. The
// ctx parameter contains a golang context object which should be used to allow
// this adapter to timeout or be cancelled when executing potentially
// long-running actions
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != s.Scopes()[0] {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		}
	}

	var input Input
	var output Output
	var err error
	var items []*sdp.Item

	err = s.Validate()
	if err != nil {
		return nil, WrapAWSError(err)
	}

	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.ItemType, query, ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		if len(cachedItems) > 0 {
			return cachedItems[0], nil
		} else {
			return nil, nil
		}
	}

	// Get the input object
	input, err = s.InputMapperGet(scope, query)
	if err != nil {
		err = s.processError(err, ck)
		return nil, err
	}

	// Call the API using the object
	output, err = s.DescribeFunc(ctx, s.Client, input)
	if err != nil {
		err = s.processError(err, ck)
		return nil, err
	}

	items, err = s.OutputMapper(ctx, s.Client, scope, input, output)
	if err != nil {
		err = s.processError(err, ck)
		return nil, err
	}

	if s.UseListForGet {
		// If we're using List for Get, we need to filter the items ourselves
		var filteredItems []*sdp.Item
		for _, item := range items {
			if item.UniqueAttributeValue() == query {
				filteredItems = append(filteredItems, item)
				break
			}
		}
		items = filteredItems
	}

	numItems := len(items)

	switch {
	case numItems > 1:
		itemNames := make([]string, 0, len(items))

		// Get the names for logging
		for i := range items {
			itemNames = append(itemNames, items[i].GloballyUniqueName())
		}

		qErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("Request returned > 1 item for a GET request. Items: %v", strings.Join(itemNames, ", ")),
		}
		s.cache.StoreError(qErr, s.cacheDuration(), ck)

		return nil, qErr
	case numItems == 0:
		qErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: fmt.Sprintf("%v %v not found", s.Type(), query),
		}
		s.cache.StoreError(qErr, s.cacheDuration(), ck)
		return nil, qErr
	}

	s.cache.StoreItem(items[0], s.cacheDuration(), ck)
	return items[0], nil
}

// List Lists all items in a given scope
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	if scope != s.Scopes()[0] {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		})
		return
	}

	if s.InputMapperList == nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: fmt.Sprintf("list is not supported for %v resources", s.ItemType),
		})
		return
	}

	err := s.Validate()
	if err != nil {
		stream.SendError(WrapAWSError(err))
		return
	}

	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_LIST, scope, s.ItemType, "", ignoreCache)
	if qErr != nil {
		stream.SendError(qErr)
		return
	}
	if cacheHit {
		for _, item := range cachedItems {
			stream.SendItem(item)
		}
		return
	}

	input, err := s.InputMapperList(scope)
	if err != nil {
		err = s.processError(err, ck)
		stream.SendError(err)
		return
	}

	s.describe(ctx, nil, input, scope, ck, stream)
}

// Search Searches for AWS resources by ARN
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	if scope != s.Scopes()[0] {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		})
		return
	}

	if s.InputMapperSearch == nil {
		s.searchARN(ctx, scope, query, ignoreCache, stream)
	} else {
		s.searchCustom(ctx, scope, query, ignoreCache, stream)
	}
}

func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) searchARN(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	// Parse the ARN
	a, err := ParseARN(query)
	if err != nil {
		stream.SendError(WrapAWSError(err))
		return
	}

	if a.ContainsWildcard() {
		// We can't handle wildcards by default so bail out
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: fmt.Sprintf("wildcards are not supported by adapter %v", s.Name()),
			Scope:       scope,
		})
		return
	}

	if arnScope := FormatScope(a.AccountID, a.Region); arnScope != scope {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("ARN scope %v does not match request scope %v", arnScope, scope),
			Scope:       scope,
		})
		return
	}

	// this already uses the cache, so needs no extra handling
	item, err := s.Get(ctx, scope, a.ResourceID(), ignoreCache)
	if err != nil {
		stream.SendError(err)
		return
	}

	stream.SendItem(item)
}

// searchCustom Runs custom search logic using the `InputMapperSearch` function
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) searchCustom(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	// We need to cache here since this is the only place it'll be called
	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_SEARCH, scope, s.ItemType, query, ignoreCache)
	if qErr != nil {
		stream.SendError(qErr)
		return
	}
	if cacheHit {
		for _, item := range cachedItems {
			stream.SendItem(item)
		}
		return
	}

	input, err := s.InputMapperSearch(ctx, s.Client, scope, query)
	if err != nil {
		stream.SendError(WrapAWSError(err))
		return
	}

	s.describe(ctx, &query, input, scope, ck, stream)
}

// Processes an error returned by the AWS API so that it can be handled by
// Overmind. This includes extracting the correct error type, wrapping in an SDP
// error, and caching that error if it is non-transient (like a 404)
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) processError(err error, cacheKey sdpcache.CacheKey) error {
	var sdpErr *sdp.QueryError

	if err != nil {
		sdpErr = WrapAWSError(err)

		// Only cache the error if is something that won't be fixed by retrying
		if sdpErr.GetErrorType() == sdp.QueryError_NOTFOUND || sdpErr.GetErrorType() == sdp.QueryError_NOSCOPE {
			s.cache.StoreError(sdpErr, s.cacheDuration(), cacheKey)
		}
	}

	return sdpErr
}

// describe Runs describe on the given input, intelligently choosing whether to
// run the paginated or unpaginated query. This handles caching, error handling,
// and post-search filtering if the query param is passed
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) describe(ctx context.Context, query *string, input Input, scope string, ck sdpcache.CacheKey, stream discovery.QueryResultStream) {
	if s.Paginated() {
		paginator := s.PaginatorBuilder(s.Client, input)

		for paginator.HasMorePages() {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				stream.SendError(s.processError(err, ck))
				return
			}

			items, err := s.OutputMapper(ctx, s.Client, scope, input, output)
			if err != nil {
				stream.SendError(s.processError(err, ck))
				return
			}

			if query != nil && s.PostSearchFilter != nil {
				items, err = s.PostSearchFilter(ctx, *query, items)
				if err != nil {
					stream.SendError(s.processError(err, ck))
					return
				}
			}

			for _, item := range items {
				s.cache.StoreItem(item, s.cacheDuration(), ck)
				stream.SendItem(item)
			}
		}
	} else {
		output, err := s.DescribeFunc(ctx, s.Client, input)
		if err != nil {
			stream.SendError(s.processError(err, ck))
			return
		}

		items, err := s.OutputMapper(ctx, s.Client, scope, input, output)
		if err != nil {
			stream.SendError(s.processError(err, ck))
			return
		}

		if query != nil && s.PostSearchFilter != nil {
			items, err = s.PostSearchFilter(ctx, *query, items)
			if err != nil {
				stream.SendError(s.processError(err, ck))
				return
			}
		}

		for _, item := range items {
			s.cache.StoreItem(item, s.cacheDuration(), ck)
			stream.SendItem(item)
		}
	}
}

// Weight Returns the priority weighting of items returned by this adapter.
// This is used to resolve conflicts where two sources of the same type
// return an item for a GET request. In this instance only one item can be
// seen on, so the one with the higher weight value will win.
func (s *DescribeOnlyAdapter[Input, Output, ClientStruct, Options]) Weight() int {
	return 100
}
