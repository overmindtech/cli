package adapterhelpers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"buf.build/go/protovalidate"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

// GetListAdapter A adapter for AWS APIs where the Get and List functions both
// return the full item, such as many of the IAM APIs
type GetListAdapter[AWSItem AWSItemType, ClientStruct ClientStructType, Options OptionsType] struct {
	ItemType               string       // The type of items that will be returned
	Client                 ClientStruct // The AWS API client
	AccountID              string       // The AWS account ID
	Region                 string       // The AWS region this is related to
	SupportGlobalResources bool         // If true, this will also support resources in the "aws" scope which are global
	AdapterMetadata        *sdp.AdapterMetadata

	CacheDuration time.Duration   // How long to cache items for
	cache         *sdpcache.Cache // The sdpcache of this adapter
	cacheInitMu   sync.Mutex      // Mutex to ensure cache is only initialised once

	// Disables List(), meaning all calls will return empty results. This does
	// not affect Search()
	DisableList bool

	// GetFunc Gets the details of a specific item, returns the AWS
	// representation of that item, and an error
	GetFunc func(ctx context.Context, client ClientStruct, scope string, query string) (AWSItem, error)

	// ListFunc Lists all items that it can find. Returning a slice of AWS items
	ListFunc func(ctx context.Context, client ClientStruct, scope string) ([]AWSItem, error)

	// Optional search func that will be used for Search Requests. If this is
	// unset, Search will simply use ARNs
	SearchFunc func(ctx context.Context, client ClientStruct, scope string, query string) ([]AWSItem, error)

	// ItemMapper Maps an AWS representation of an item to the SDP version
	ItemMapper func(query, scope string, awsItem AWSItem) (*sdp.Item, error)

	// ListTagsFunc Optional function that will be used to list tags for a
	// resource
	ListTagsFunc func(context.Context, AWSItem, ClientStruct) (map[string]string, error)
}

func (s *GetListAdapter[AWSItem, ClientStruct, Options]) cacheDuration() time.Duration {
	if s.CacheDuration == 0 {
		return DefaultCacheDuration
	}

	return s.CacheDuration
}

func (s *GetListAdapter[AWSItem, ClientStruct, Options]) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
	}
}

func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
}

// Validate Checks that the adapter has been set up correctly
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Validate() error {
	if s.GetFunc == nil {
		return errors.New("GetFunc is nil")
	}

	if !s.DisableList {
		if s.ListFunc == nil {
			return errors.New("ListFunc is nil")
		}
	}

	if s.ItemMapper == nil {
		return errors.New("ItemMapper is nil")
	}

	return protovalidate.Validate(s.AdapterMetadata)
}

func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Type() string {
	return s.ItemType
}

func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Name() string {
	return fmt.Sprintf("%v-adapter", s.ItemType)
}

func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Metadata() *sdp.AdapterMetadata {
	return s.AdapterMetadata
}

// List of scopes that this adapter is capable of find items for. This will be
// in the format {accountID}.{region}
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Scopes() []string {
	scopes := make([]string, 0)

	scopes = append(scopes, FormatScope(s.AccountID, s.Region))

	if s.SupportGlobalResources {
		scopes = append(scopes, "aws")
	}

	return scopes
}

// hasScope Returns whether or not this adapter has the given scope
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) hasScope(scope string) bool {
	if scope == "aws" && s.SupportGlobalResources {
		// There is a special global "account" that is used for global resources
		// called "aws"
		return true
	}

	for _, s := range s.Scopes() {
		if s == scope {
			return true
		}
	}

	return false
}

// Get retrieves an item from the adapter based on the provided scope, query, and
// cache settings. It uses the defined `GetFunc`, `ItemMapper`, and
// `ListTagsFunc` to retrieve and map the item.
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if !s.hasScope(scope) {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		}
	}

	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.ItemType, query, ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		if len(cachedItems) == 0 {
			return nil, nil
		} else {
			return cachedItems[0], nil
		}
	}

	awsItem, err := s.GetFunc(ctx, s.Client, scope, query)
	if err != nil {
		err := WrapAWSError(err)
		if !CanRetry(err) {
			s.cache.StoreError(err, s.cacheDuration(), ck)
		}
		return nil, err
	}

	item, err := s.ItemMapper(query, scope, awsItem)
	if err != nil {
		// Don't cache this as wrapping is very cheap and better to just try
		// again than store in memory
		return nil, WrapAWSError(err)
	}

	if s.ListTagsFunc != nil {
		item.Tags, err = s.ListTagsFunc(ctx, awsItem, s.Client)
		if err != nil {
			item.Tags = HandleTagsError(ctx, err)
		}
	}

	s.cache.StoreItem(item, s.cacheDuration(), ck)

	return item, nil
}

// List Lists all available items. This is done by running the ListFunc, then
// passing these results to GetFunc in order to get the details
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if !s.hasScope(scope) {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		}
	}

	if s.DisableList {
		return []*sdp.Item{}, nil
	}

	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_LIST, scope, s.ItemType, "", ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		return cachedItems, nil
	}

	awsItems, err := s.ListFunc(ctx, s.Client, scope)
	if err != nil {
		return nil, WrapAWSError(err)
	}

	items := make([]*sdp.Item, 0)
	for _, awsItem := range awsItems {
		item, err := s.ItemMapper("", scope, awsItem)
		if err != nil {
			continue
		}

		if s.ListTagsFunc != nil {
			item.Tags, err = s.ListTagsFunc(ctx, awsItem, s.Client)
			if err != nil {
				item.Tags = HandleTagsError(ctx, err)
			}
		}

		items = append(items, item)
		s.cache.StoreItem(item, s.cacheDuration(), ck)
	}

	return items, nil
}

// Search Searches for AWS resources, this can be implemented either as a
// generic ARN search that tries to extract the globally unique name from the
// ARN and pass this to a Get request, or a custom search function that can be
// used to search for items in a different, adapter-specific way
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if !s.hasScope(scope) {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		}
	}

	if s.SearchFunc != nil {
		return s.SearchCustom(ctx, scope, query, ignoreCache)
	} else {
		return s.SearchARN(ctx, scope, query, ignoreCache)
	}
}

// Extracts the `ResourceID` and scope from the ARN, then calls `Get` with the
// extracted `ResourceID`
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) SearchARN(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	// Parse the ARN
	a, err := ParseARN(query)

	if err != nil {
		return nil, WrapAWSError(err)
	}

	if a.ContainsWildcard() {
		// We can't handle wildcards by default so bail out
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: fmt.Sprintf("wildcards are not supported by adapter %v", s.Name()),
			Scope:       scope,
		}
	}

	if arnScope := FormatScope(a.AccountID, a.Region); !s.hasScope(arnScope) {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("ARN scope %v does not match request scope %v", arnScope, scope),
			Scope:       scope,
		}
	}

	// Since this gits the Get method, and this method implements caching, we
	// don't need to implement it here
	item, err := s.Get(ctx, scope, a.ResourceID(), ignoreCache)

	if err != nil {
		return nil, WrapAWSError(err)
	}

	return []*sdp.Item{item}, nil
}

// Custom search function that can be used to search for items in a different,
// adapter-specific way
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) SearchCustom(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	// We need to cache here since this is the only place it'll be called
	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_SEARCH, scope, s.ItemType, query, ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		return cachedItems, nil
	}

	awsItems, err := s.SearchFunc(ctx, s.Client, scope, query)
	if err != nil {
		err = WrapAWSError(err)
		s.cache.StoreError(err, s.cacheDuration(), ck)
		return nil, err
	}

	items := make([]*sdp.Item, 0)
	var item *sdp.Item

	for _, awsItem := range awsItems {
		item, err = s.ItemMapper(query, scope, awsItem)
		if err != nil {
			continue
		}

		items = append(items, item)
		s.cache.StoreItem(item, s.cacheDuration(), ck)
	}

	return items, nil
}

// Weight Returns the priority weighting of items returned by this adapter.
// This is used to resolve conflicts where two adapters of the same type
// return an item for a GET request. In this instance only one item can be
// seen on, so the one with the higher weight value will win.
func (s *GetListAdapter[AWSItem, ClientStruct, Options]) Weight() int {
	return 100
}
