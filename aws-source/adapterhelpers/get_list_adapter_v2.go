package adapterhelpers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

// GetListAdapterV2 A adapter for AWS APIs where the Get and List functions both
// return the full item, such as many of the IAM APIs. This version supports
// paginated APIs and streaming results.
type GetListAdapterV2[ListInput InputType, ListOutput OutputType, AWSItem AWSItemType, ClientStruct ClientStructType, Options OptionsType] struct {
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

	// A function that returns the input object that will be passed to
	// ListFunc for a LIST request
	InputMapperList func(scope string) (ListInput, error)

	// ListFunc Lists all items that it can find this should be used only if the
	// API does not have a paginator, otherwise use ListFuncPaginatorBuilder
	ListFunc func(ctx context.Context, client ClientStruct, input ListInput) (ListOutput, error)

	// A function that returns a paginator for this API. If this is nil, we will
	// assume that the API is not paginated e.g.
	// https://aws.github.io/aws-sdk-go-v2/docs/making-requests/#using-paginators
	//
	// If this is set then ListFunc will be ignored
	ListFuncPaginatorBuilder func(client ClientStruct, params ListInput) Paginator[ListOutput, Options]

	// Extracts the list of items from the output of the ListFunc, these will be
	// passed to the ItemMapper for conversion to SDP items
	ListExtractor func(ctx context.Context, output ListOutput, client ClientStruct) ([]AWSItem, error)

	// NOTE
	//
	// This does not yet support custom searching, this will be added in a
	// future version

	// ItemMapper Maps an AWS representation of an item to the SDP version, the
	// query will be nil if the method was LIST
	ItemMapper func(query *string, scope string, awsItem AWSItem) (*sdp.Item, error)

	// ListTagsFunc Optional function that will be used to list tags for a
	// resource
	ListTagsFunc func(context.Context, AWSItem, ClientStruct) (map[string]string, error)
}

func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) cacheDuration() time.Duration {
	if s.CacheDuration == 0 {
		return DefaultCacheDuration
	}

	return s.CacheDuration
}

func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
	}
}

func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
}

// Validate Checks that the adapter has been set up correctly
func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) Validate() error {
	if s.GetFunc == nil {
		return errors.New("GetFunc is nil")
	}

	if !s.DisableList {
		if s.ListFunc == nil && s.ListFuncPaginatorBuilder == nil {
			return errors.New("ListFunc and ListFuncPaginatorBuilder are nil")
		}

		if s.ListExtractor == nil {
			return errors.New("ListExtractor is nil")
		}

		if s.InputMapperList == nil {
			return errors.New("InputMapperList is nil")
		}
	}

	if s.ItemMapper == nil {
		return errors.New("ItemMapper is nil")
	}

	return nil
}

func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) Type() string {
	return s.ItemType
}

func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) Name() string {
	return fmt.Sprintf("%v-adapter", s.ItemType)
}

func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) Metadata() *sdp.AdapterMetadata {
	return s.AdapterMetadata
}

// List of scopes that this adapter is capable of find items for. This will be
// in the format {accountID}.{region}
func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) Scopes() []string {
	scopes := make([]string, 0)

	scopes = append(scopes, FormatScope(s.AccountID, s.Region))

	if s.SupportGlobalResources {
		scopes = append(scopes, "aws")
	}

	return scopes
}

// hasScope Returns whether or not this adapter has the given scope
func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) hasScope(scope string) bool {
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
func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
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

	item, err := s.ItemMapper(&query, scope, awsItem)
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
func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	if !s.hasScope(scope) {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		})
		return
	}

	if s.DisableList {
		return
	}

	if err := s.Validate(); err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		})
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

	listInput, err := s.InputMapperList(scope)
	if err != nil {
		stream.SendError(WrapAWSError(err))
		return
	}

	// Define the function to send the outputs
	sendOutputs := func(out ListOutput) {
		// Extract the items in the correct format
		awsItems, err := s.ListExtractor(ctx, out, s.Client)
		if err != nil {
			stream.SendError(WrapAWSError(err))
			return
		}

		// Map the items to SDP items, send on the stream, and save to the
		// cache
		for _, awsItem := range awsItems {
			item, err := s.ItemMapper(nil, scope, awsItem)
			if err != nil {
				stream.SendError(WrapAWSError(err))
				continue
			}

			if s.ListTagsFunc != nil {
				item.Tags, err = s.ListTagsFunc(ctx, awsItem, s.Client)
				if err != nil {
					item.Tags = HandleTagsError(ctx, err)
				}
			}

			stream.SendItem(item)
			s.cache.StoreItem(item, s.cacheDuration(), ck)
		}
	}

	// See if this is paginated or not and use the appropriate method
	if s.ListFuncPaginatorBuilder != nil {
		paginator := s.ListFuncPaginatorBuilder(s.Client, listInput)

		for paginator.HasMorePages() {
			out, err := paginator.NextPage(ctx)
			if err != nil {
				stream.SendError(WrapAWSError(err))
				return
			}

			sendOutputs(out)
		}
	} else if s.ListFunc != nil {
		out, err := s.ListFunc(ctx, s.Client, listInput)
		if err != nil {
			stream.SendError(WrapAWSError(err))
			return
		}

		sendOutputs(out)
	}
}

// Search Searches for AWS resources, this can be implemented either as a
// generic ARN search that tries to extract the globally unique name from the
// ARN and pass this to a Get request, or a custom search function that can be
// used to search for items in a different, adapter-specific way
func (s *GetListAdapterV2[ListInput, ListOutput, AWSItem, ClientStruct, Options]) SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	if !s.hasScope(scope) {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match adapter scope %v", scope, s.Scopes()[0]),
		})
		return
	}

	// Parse the ARN
	a, err := ParseARN(query)
	if err != nil {
		stream.SendError(WrapAWSError(err))
		return
	}

	if a.ContainsWildcard() {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: fmt.Sprintf("wildcards are not supported by adapter %v", s.Name()),
			Scope:       scope,
		})
		return
	}

	if arnScope := FormatScope(a.AccountID, a.Region); !s.hasScope(arnScope) {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("ARN scope %v does not match request scope %v", arnScope, scope),
			Scope:       scope,
		})
		return
	}

	// Since this gits the Get method, and this method implements caching, we
	// don't need to implement it here
	item, err := s.Get(ctx, scope, a.ResourceID(), ignoreCache)
	if err != nil {
		stream.SendError(err)
		return
	}

	if item != nil {
		stream.SendItem(item)
	}
}
