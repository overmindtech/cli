package adapterhelpers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"buf.build/go/protovalidate"
	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/sourcegraph/conc/pool"
)

// MaxParallel An integer that defaults to 10
type MaxParallel int

// Value Get the value of MaxParallel, defaulting to 10
func (m MaxParallel) Value() int {
	if m == 0 {
		return 10
	}

	return int(m)
}

// AlwaysGetAdapter This adapter is designed for AWS APIs that have separate List
// and Get functions. It also assumes that the results of the list function
// cannot be converted directly into items as they do not contain enough
// information, and therefore they always need to be passed to the Get function
// before returning. An example is the `ListClusters` API in EKS which returns a
// list of cluster names.
type AlwaysGetAdapter[ListInput InputType, ListOutput OutputType, GetInput InputType, GetOutput OutputType, ClientStruct ClientStructType, Options OptionsType] struct {
	ItemType        string       // The type of items to return
	Client          ClientStruct // The AWS API client
	AccountID       string       // The AWS account ID
	Region          string       // The AWS region this is related to
	MaxParallel     MaxParallel  // How many Get request to run in parallel for a single List request
	AdapterMetadata *sdp.AdapterMetadata

	// Disables List(), meaning all calls will return empty results. This does
	// not affect Search()
	DisableList bool

	// A function that gets the details of a given item. This should include the
	// tags if relevant
	GetFunc func(ctx context.Context, client ClientStruct, scope string, input GetInput) (*sdp.Item, error)

	// The input to the ListFunc. This is static
	ListInput ListInput

	// A function that maps from the SDP get inputs to the relevant input for
	// the GetFunc
	GetInputMapper func(scope, query string) GetInput

	// If this is set, Search queries will always use the automatic ARN resolver
	// if the input is an ARN, falling back to the `SearchInputMapper` if it
	// isn't
	AlwaysSearchARNs bool

	// Maps search terms from an SDP Search request into the relevant input for
	// the ListFunc. If this is not set, Search() will handle ARNs like most AWS
	// adapters. Note that this and `SearchGetInputMapper` are mutually exclusive
	SearchInputMapper func(scope, query string) (ListInput, error)

	// Maps search terms from an SDP Search request into the relevant input for
	// the GetFunc. If this is not set, Search() will handle ARNs like most AWS
	// adapters. Note that this and `SearchInputMapper` are mutually exclusive
	SearchGetInputMapper func(scope, query string) (GetInput, error)

	// A function that returns a paginator for the ListFunc
	ListFuncPaginatorBuilder func(client ClientStruct, input ListInput) Paginator[ListOutput, Options]

	// A function that accepts the output of a ListFunc and maps this to a slice
	// of inputs to pass to the GetFunc. The input used for the ListFunc is also
	// included in case it is required
	ListFuncOutputMapper func(output ListOutput, input ListInput) ([]GetInput, error)

	CacheDuration time.Duration   // How long to cache items for
	cache         *sdpcache.Cache // The sdpcache of this adapter
	cacheInitMu   sync.Mutex      // Mutex to ensure cache is only initialised once
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) cacheDuration() time.Duration {
	if s.CacheDuration == 0 {
		return DefaultCacheDuration
	}

	return s.CacheDuration
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
	}
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
}

// Validate Checks that the adapter has been set up correctly
func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) Validate() error {
	if !s.DisableList {
		if s.ListFuncPaginatorBuilder == nil {
			return errors.New("ListFuncPaginatorBuilder is nil")
		}

		if s.ListFuncOutputMapper == nil {
			return errors.New("ListFuncOutputMapper is nil")
		}
	}

	if s.GetFunc == nil {
		return errors.New("GetFunc is nil")
	}

	if s.GetInputMapper == nil {
		return errors.New("GetInputMapper is nil")
	}

	if s.SearchGetInputMapper != nil && s.SearchInputMapper != nil {
		return errors.New("SearchGetInputMapper and SearchInputMapper are mutually exclusive")
	}

	return protovalidate.Validate(s.AdapterMetadata)
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) Type() string {
	return s.ItemType
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) Name() string {
	return fmt.Sprintf("%v-adapter", s.ItemType)
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) Metadata() *sdp.AdapterMetadata {
	return s.AdapterMetadata
}

// List of scopes that this adapter is capable of find items for. This will be
// in the format {accountID}.{region}
func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) Scopes() []string {
	return []string{
		FormatScope(s.AccountID, s.Region),
	}
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != s.Scopes()[0] {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		}
	}

	var err error
	var item *sdp.Item

	if err = s.Validate(); err != nil {
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

	input := s.GetInputMapper(scope, query)

	item, err = s.GetFunc(ctx, s.Client, scope, input)
	if err != nil {
		err := WrapAWSError(err)
		if !CanRetry(err) {
			s.cache.StoreError(err, s.cacheDuration(), ck)
		}
		return nil, err
	}

	s.cache.StoreItem(item, s.cacheDuration(), ck)
	return item, nil
}

// List Lists all available items. This is done by running the ListFunc, then
// passing these results to GetFunc in order to get the details
func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	if scope != s.Scopes()[0] {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		})
		return
	}

	if err := s.Validate(); err != nil {
		stream.SendError(WrapAWSError(err))
		return
	}

	// Check to see if we have supplied the required functions
	if s.DisableList {
		// In this case we can't run list, so just return empty
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

	s.listInternal(ctx, scope, s.ListInput, ck, stream)
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) listInternal(ctx context.Context, scope string, input ListInput, ck sdpcache.CacheKey, stream discovery.QueryResultStream) {
	paginator := s.ListFuncPaginatorBuilder(s.Client, input)
	var newGetInputs []GetInput
	p := pool.New().WithContext(ctx).WithMaxGoroutines(s.MaxParallel.Value())
	defer func() {
		// Always wait for everything to be completed before returning
		err := p.Wait()
		if err != nil {
			sentry.CaptureException(err)
		}
	}()

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			err := WrapAWSError(err)
			if !CanRetry(err) {
				s.cache.StoreError(err, s.cacheDuration(), ck)
			}
			stream.SendError(err)
			return
		}

		newGetInputs, err = s.ListFuncOutputMapper(output, input)
		if err != nil {
			err := WrapAWSError(err)
			if !CanRetry(err) {
				s.cache.StoreError(err, s.cacheDuration(), ck)
			}
			stream.SendError(err)
			return
		}

		for _, input := range newGetInputs {
			// This call will block if no workers are available, and therefore
			// we will only load new pages once there are workers ready to
			// accept that work
			p.Go(func(ctx context.Context) error {
				item, err := s.GetFunc(ctx, s.Client, scope, input)
				if err != nil {
					// Don't cache individual errors as they are cheap to re-run
					stream.SendError(WrapAWSError(err))
				}
				if item != nil {
					s.cache.StoreItem(item, s.cacheDuration(), ck)
					stream.SendItem(item)
				}

				return nil
			})
		}
	}
}

// Search Searches for AWS resources by ARN
func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	if scope != s.Scopes()[0] {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		})
		return
	}

	if err := s.Validate(); err != nil {
		stream.SendError(WrapAWSError(err))
		return
	}

	if s.SearchInputMapper == nil && s.SearchGetInputMapper == nil {
		s.SearchARN(ctx, scope, query, ignoreCache, stream)
	} else {
		// If we should always look for ARNs first, do that
		if s.AlwaysSearchARNs {
			if _, err := ParseARN(query); err == nil {
				s.SearchARN(ctx, scope, query, ignoreCache, stream)
			} else {
				s.SearchCustom(ctx, scope, query, ignoreCache, stream)
			}
		} else {
			s.SearchCustom(ctx, scope, query, ignoreCache, stream)
		}
	}
}

// SearchCustom Searches using custom mapping logic. The SearchInputMapper is
// used to create an input for ListFunc, at which point the usual logic is used
func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) SearchCustom(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
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

	if s.SearchInputMapper != nil {
		input, err := s.SearchInputMapper(scope, query)
		if err != nil {
			// Don't bother caching this error since it costs nearly nothing
			stream.SendError(WrapAWSError(err))
			return
		}

		s.listInternal(ctx, scope, input, ck, stream)
	} else if s.SearchGetInputMapper != nil {
		input, err := s.SearchGetInputMapper(scope, query)
		if err != nil {
			// Don't cache this as it costs nearly nothing
			stream.SendError(WrapAWSError(err))
			return
		}

		item, err := s.GetFunc(ctx, s.Client, scope, input)
		if err != nil {
			err := WrapAWSError(err)
			if !CanRetry(err) {
				s.cache.StoreError(err, s.cacheDuration(), ck)
			}
			stream.SendError(err)
			return
		}

		if item != nil {
			s.cache.StoreItem(item, s.cacheDuration(), ck)
			stream.SendItem(item)
		}
	} else {
		stream.SendError(errors.New("SearchCustom called without SearchInputMapper or SearchGetInputMapper"))
		return
	}
}

func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) SearchARN(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
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

	item, err := s.Get(ctx, scope, a.ResourceID(), ignoreCache)
	if err != nil {
		stream.SendError(WrapAWSError(err))
		return
	}

	if item != nil {
		stream.SendItem(item)
	}
}

// Weight Returns the priority weighting of items returned by this sourcs.
// This is used to resolve conflicts where two sources of the same type
// return an item for a GET request. In this instance only one item can be
// seen on, so the one with the higher weight value will win.
func (s *AlwaysGetAdapter[ListInput, ListOutput, GetInput, GetOutput, ClientStruct, Options]) Weight() int {
	return 100
}
