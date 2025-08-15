package sources

import (
	"context"
	"fmt"
	"strings"

	"buf.build/go/protovalidate"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// ItemTypeLookups is a slice of ItemTypeLookup.
type ItemTypeLookups []shared.ItemTypeLookup

// ReadableFormat returns a readable format of the ItemTypeLookups
func (lookups ItemTypeLookups) ReadableFormat() string {
	var readableLookups []string
	for _, lookup := range lookups {
		readableLookups = append(readableLookups, lookup.Readable())
	}

	return strings.Join(readableLookups, shared.QuerySeparator)
}

// Wrapper defines the base interface for resource wrappers.
type Wrapper interface {
	Scopes() []string
	GetLookups() ItemTypeLookups
	Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError)
	Type() string
	Name() string
	ItemType() shared.ItemType
	TerraformMappings() []*sdp.TerraformMapping
	Category() sdp.AdapterCategory
	PotentialLinks() map[shared.ItemType]bool
	AdapterMetadata() *sdp.AdapterMetadata
	IAMPermissions() []string
}

// ListableWrapper defines an optional interface for resources that support listing.
type ListableWrapper interface {
	Wrapper
	List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError)
}

// ListStreamableWrapper defines an interface for resources that support listing with streaming.
type ListStreamableWrapper interface {
	Wrapper
	ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey)
}

// SearchableWrapper defines an optional interface for resources that support searching.
type SearchableWrapper interface {
	Wrapper
	SearchLookups() []ItemTypeLookups
	Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError)
}

// SearchStreamableWrapper defines an interface for resources that support searching with streaming.
type SearchStreamableWrapper interface {
	Wrapper
	SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey, queryParts ...string)
}

// SearchableListableWrapper defines an interface for resources that support both searching and listing.
type SearchableListableWrapper interface {
	SearchableWrapper
	ListableWrapper
}

// StandardAdapter defines the standard interface for adapters.
type StandardAdapter interface {
	Validate() error
	discovery.Adapter
	discovery.ListableAdapter
	discovery.StreamingAdapter
	discovery.SearchableAdapter
	discovery.CachingAdapter
}

// WrapperToAdapter converts a Wrapper to a StandardAdapter.
func WrapperToAdapter(wrapper Wrapper) StandardAdapter {
	a := &standardAdapterImpl{
		wrapper: wrapper,
	}

	a.sourceType = "unknown"

	it, ok := wrapper.ItemType().(shared.ItemTypeInstance)
	if ok {
		a.sourceType = string(it.Source)
	}

	// initialize cache
	a.cache = sdpcache.NewCache()

	// Check if the wrapper supports ListableWrapper
	if listable, ok := wrapper.(ListableWrapper); ok {
		a.listable = listable
	}

	// Check if the wrapper supports ListableStreamableWrapper
	if listStreamable, ok := wrapper.(ListStreamableWrapper); ok {
		a.listStreamable = listStreamable
	}

	// Check if the wrapper supports SearchableWrapper
	if searchable, ok := wrapper.(SearchableWrapper); ok {
		a.searchable = searchable
	}

	// Check if the wrapper supports SearchableStreamableWrapper
	if searchStreamable, ok := wrapper.(SearchStreamableWrapper); ok {
		a.searchStreamable = searchStreamable
	}

	if err := a.Validate(); err != nil {
		panic(fmt.Sprintf("failed to validate adapter: %v", err))
	}

	if iamPerms := wrapper.IAMPermissions(); len(iamPerms) > 0 {
		for _, perm := range iamPerms {
			gcpshared.IAMPermissions[perm] = true
		}
	}

	return a
}

type standardAdapterImpl struct {
	wrapper          Wrapper
	listable         ListableWrapper
	listStreamable   ListStreamableWrapper
	searchable       SearchableWrapper
	searchStreamable SearchStreamableWrapper
	cache            *sdpcache.Cache
	sourceType       string
}

// Type returns the type of the adapter.
func (s *standardAdapterImpl) Type() string {
	return s.wrapper.Type()
}

// Name returns the name of the adapter.
func (s *standardAdapterImpl) Name() string {
	return s.wrapper.Name()
}

// Scopes returns the scopes of the adapter.
func (s *standardAdapterImpl) Scopes() []string {
	return s.wrapper.Scopes()
}

// Get retrieves a single item with a given scope and query.
func (s *standardAdapterImpl) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if err := s.validateScopes(scope); err != nil {
		return nil, err
	}

	cacheHit, ck, cachedItem, qErr := s.cache.Lookup(
		ctx,
		s.Name(),
		sdp.QueryMethod_GET,
		scope,
		s.Type(),
		query,
		ignoreCache,
	)
	if qErr != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      s.sourceType,
			"ovm.source.adapter":   s.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_GET.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Error("failed to lookup item in cache")
	}

	if cacheHit && len(cachedItem) > 0 {
		return cachedItem[0], nil
	}

	queryParts := strings.Split(query, shared.QuerySeparator)
	if len(queryParts) != len(s.wrapper.GetLookups()) {
		return nil, fmt.Errorf(
			"invalid query format: %s, expected: %s",
			query,
			s.wrapper.GetLookups().ReadableFormat(),
		)
	}

	item, err := s.wrapper.Get(ctx, queryParts...)
	if err != nil {
		return nil, err
	}

	// Store in cache after successful get
	if s.cache != nil {
		s.cache.StoreItem(item, shared.DefaultCacheDuration, ck)
	}

	return item, nil
}

// List retrieves all items in a given scope.
func (s *standardAdapterImpl) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if err := s.validateScopes(scope); err != nil {
		return nil, err
	}

	if s.listable == nil {
		log.WithField("adapter", s.Name()).Debug("list operation not supported")

		return nil, nil
	}

	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(
		ctx,
		s.Name(),
		sdp.QueryMethod_LIST,
		scope,
		s.Type(),
		"",
		ignoreCache,
	)
	if qErr != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      s.sourceType,
			"ovm.source.adapter":   s.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_LIST.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Error("failed to lookup item in cache")
	}

	if cacheHit {
		return cachedItems, nil
	}

	items, err := s.listable.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		if s.cache != nil {
			s.cache.StoreItem(item, shared.DefaultCacheDuration, ck)
		}
	}

	return items, nil
}

func (s *standardAdapterImpl) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	if err := s.validateScopes(scope); err != nil {
		stream.SendError(err)
		return
	}

	if s.listStreamable == nil {
		log.WithField("adapter", s.Name()).Debug("list stream operation not supported")
		return
	}

	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(
		ctx,
		s.Name(),
		sdp.QueryMethod_LIST,
		scope,
		s.Type(),
		"",
		ignoreCache,
	)
	if qErr != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      s.sourceType,
			"ovm.source.adapter":   s.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_LIST.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Error("failed to lookup item in cache")
	}

	if cacheHit {
		for _, item := range cachedItems {
			stream.SendItem(item)
		}
		return
	}

	s.listStreamable.ListStream(ctx, stream, s.cache, ck)
}

// Search retrieves items based on a search query.
func (s *standardAdapterImpl) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if err := s.validateScopes(scope); err != nil {
		return nil, err
	}

	var queryParts []string
	if s.sourceType == string(gcpshared.GCP) && strings.HasPrefix(query, "projects/") {
		// This must be a terraform query in the format of:
		// projects/{{project}}/datasets/{{dataset}}/tables/{{name}}
		// projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
		//
		// Extract the relevant parts from the query
		// We need to extract the path parameters based on the number of lookups
		queryParts = gcpshared.ExtractPathParamsWithCount(query, len(s.wrapper.GetLookups()))
		if len(queryParts) != len(s.wrapper.GetLookups()) {
			return nil, &sdp.QueryError{
				ErrorType: sdp.QueryError_OTHER,
				ErrorString: fmt.Sprintf(
					"failed to handle terraform mapping from query %s for %s",
					query,
					s.wrapper.ItemType().Readable(),
				),
			}
		}

		item, err := s.Get(ctx, scope, shared.CompositeLookupKey(queryParts...), ignoreCache)
		if err != nil {
			return nil, fmt.Errorf("failed to get item from terraform mapping: %w", err)
		}

		return []*sdp.Item{item}, nil
	}

	if s.searchable == nil {
		log.WithField("adapter", s.Name()).Debug("search operation not supported")

		return nil, nil
	}

	// This must be a regular query in the format of:
	// {{datasetName}}|{{tableName}}
	queryParts = strings.Split(query, shared.QuerySeparator)

	var validQuery bool
	for _, kw := range s.searchable.SearchLookups() {
		if len(kw) == len(queryParts) {
			validQuery = true
			break
		}

		continue
	}

	if !validQuery {
		return nil, fmt.Errorf(
			"invalid search query format: %s, expected: %s",
			query,
			expectedSearchQueryFormat(s.searchable.SearchLookups()),
		)
	}

	items, err := s.searchable.Search(ctx, queryParts...)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *standardAdapterImpl) SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	if err := s.validateScopes(scope); err != nil {
		stream.SendError(err)
		return
	}

	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(
		ctx,
		s.Name(),
		sdp.QueryMethod_SEARCH,
		scope,
		s.Type(),
		query,
		ignoreCache,
	)
	if qErr != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      s.sourceType,
			"ovm.source.adapter":   s.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_SEARCH.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Error("failed to lookup item in cache")
	}

	if cacheHit {
		for _, item := range cachedItems {
			stream.SendItem(item)
		}

		return
	}

	var queryParts []string
	if s.sourceType == string(gcpshared.GCP) && strings.HasPrefix(query, "projects/") {
		// This must be a terraform query in the format of:
		// projects/{{project}}/datasets/{{dataset}}/tables/{{name}}
		// projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
		//
		// Extract the relevant parts from the query
		// We need to extract the path parameters based on the number of lookups
		queryParts = gcpshared.ExtractPathParamsWithCount(query, len(s.wrapper.GetLookups()))
		if len(queryParts) != len(s.wrapper.GetLookups()) {
			stream.SendError(&sdp.QueryError{
				ErrorType: sdp.QueryError_OTHER,
				ErrorString: fmt.Sprintf(
					"failed to handle terraform mapping from query %s for %s",
					query,
					s.wrapper.ItemType().Readable(),
				),
			})
			return
		}

		item, err := s.Get(ctx, scope, shared.CompositeLookupKey(queryParts...), ignoreCache)
		if err != nil {
			stream.SendError(fmt.Errorf("failed to get item from terraform mapping: %w", err))
			return
		}

		s.cache.StoreItem(item, shared.DefaultCacheDuration, ck)

		stream.SendItem(item)
		return
	}

	if s.searchStreamable == nil {
		log.WithField("adapter", s.Name()).Debug("search stream operation not supported")
		return
	}

	// This must be a regular query in the format of:
	// {{datasetName}}|{{tableName}}
	queryParts = strings.Split(query, shared.QuerySeparator)

	var validQuery bool
	for _, kw := range s.searchable.SearchLookups() {
		if len(kw) == len(queryParts) {
			validQuery = true
			break
		}

		continue
	}

	if !validQuery {
		stream.SendError(fmt.Errorf(
			"invalid search query format: %s, expected: %s",
			query,
			expectedSearchQueryFormat(s.searchable.SearchLookups()),
		))
		return
	}

	s.searchStreamable.SearchStream(ctx, stream, s.cache, ck, queryParts...)
}

// Cache returns the cache of the adapter.
func (s *standardAdapterImpl) Cache() *sdpcache.Cache {
	return s.cache
}

// Metadata returns the metadata of the adapter.
// It uses the wrapper's metadata if available, otherwise constructs it based on the wrapper's type and capabilities.
func (s *standardAdapterImpl) Metadata() *sdp.AdapterMetadata {
	if s.wrapper.AdapterMetadata() != nil {

		return s.wrapper.AdapterMetadata()
	}

	supportedQueryMethods := &sdp.AdapterSupportedQueryMethods{
		Get: true,
		GetDescription: fmt.Sprintf(
			"Get %s by \"%s\"",
			s.wrapper.ItemType().Readable(),
			s.wrapper.GetLookups().ReadableFormat(),
		),
	}

	if s.listable != nil {
		supportedQueryMethods.List = true
		supportedQueryMethods.ListDescription = fmt.Sprintf(
			"List all %s items", s.wrapper.ItemType().Readable())
	}

	if s.searchable != nil {
		supportedQueryMethods.Search = true
		supportedQueryMethods.SearchDescription = fmt.Sprintf(
			"Search for %s by \"%s\"",
			s.wrapper.ItemType().Readable(),
			expectedSearchQueryFormat(s.searchable.SearchLookups()),
		)
	}

	a := &sdp.AdapterMetadata{
		Type:                  s.wrapper.Type(),
		Category:              s.wrapper.Category(),
		DescriptiveName:       s.wrapper.ItemType().Readable(),
		TerraformMappings:     s.wrapper.TerraformMappings(),
		SupportedQueryMethods: supportedQueryMethods,
	}

	if s.wrapper.PotentialLinks() != nil {
		for link := range s.wrapper.PotentialLinks() {
			a.PotentialLinks = append(a.PotentialLinks, link.String())
		}
	}

	return a
}

// Validate checks if the adapter is valid.
func (s *standardAdapterImpl) Validate() error {
	if s.cache == nil {
		return fmt.Errorf("cache is not initialized")
	}

	return protovalidate.Validate(s.Metadata())
}

func (s *standardAdapterImpl) validateScopes(scope string) error {
	for _, expectedScope := range s.Scopes() {
		if scope == expectedScope {
			return nil
		}
	}

	return &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOSCOPE,
		ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, s.Scopes()),
	}
}

// expectedSearchQueryFormat generates a readable format for the search query.
func expectedSearchQueryFormat(keywords []ItemTypeLookups) string {
	var readableKeywords []string
	for _, keyword := range keywords {
		readableKeywords = append(readableKeywords, keyword.ReadableFormat())
	}

	return strings.Join(readableKeywords, "\" or \"")
}
