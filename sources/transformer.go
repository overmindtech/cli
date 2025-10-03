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
	PredefinedRole() string
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
}

// WrapperToAdapter converts a Wrapper to a StandardAdapter.
func WrapperToAdapter(wrapper Wrapper) StandardAdapter {
	core := standardAdapterCore{
		wrapper: wrapper,
	}

	core.sourceType = "unknown"

	it, ok := wrapper.ItemType().(shared.ItemTypeInstance)
	if ok {
		core.sourceType = string(it.Source)
	}

	// initialize cache
	core.cache = sdpcache.NewCache()

	// Check if wrapper supports both List and Search - if so, return standardSearchableListableAdapterImpl
	if listable, listOk := wrapper.(ListableWrapper); listOk {
		if searchable, searchOk := wrapper.(SearchableWrapper); searchOk {
			listableImpl := &standardListableAdapterImpl{
				listable: listable,
			}

			searchableImpl := &standardSearchableAdapterImpl{
				searchable: searchable,
			}

			// Check for streaming capabilities
			if listStreamable, ok := wrapper.(ListStreamableWrapper); ok {
				listableImpl.listStreamable = listStreamable
			}

			if searchStreamable, ok := wrapper.(SearchStreamableWrapper); ok {
				searchableImpl.searchStreamable = searchStreamable
			}

			// Set the core for delegate implementations
			listableImpl.standardAdapterCore = core
			searchableImpl.standardAdapterCore = core

			a := &standardSearchableListableAdapterImpl{
				listableImpl:        listableImpl,
				searchableImpl:      searchableImpl,
				standardAdapterCore: core,
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

		// Listable only
		a := &standardListableAdapterImpl{
			standardAdapterCore: core,
			listable:            listable,
		}

		if listStreamable, ok := wrapper.(ListStreamableWrapper); ok {
			a.listStreamable = listStreamable
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

	// Check if wrapper is searchable only - return standardSearchableAdapterImpl
	if searchable, ok := wrapper.(SearchableWrapper); ok {
		a := &standardSearchableAdapterImpl{
			standardAdapterCore: core,
			searchable:          searchable,
		}

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

	// For non-listable, non-searchable wrappers, return standardAdapterImpl
	a := &standardAdapterImpl{
		standardAdapterCore: core,
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

type standardAdapterCore struct {
	wrapper    Wrapper
	sourceType string
	cache      *sdpcache.Cache
}

type standardAdapterImpl struct {
	standardAdapterCore
}

type standardListableAdapterImpl struct {
	listable       ListableWrapper
	listStreamable ListStreamableWrapper
	standardAdapterCore
}

type standardSearchableAdapterImpl struct {
	searchable       SearchableWrapper
	searchStreamable SearchStreamableWrapper
	standardAdapterCore
}

type standardSearchableListableAdapterImpl struct {
	listableImpl   *standardListableAdapterImpl
	searchableImpl *standardSearchableAdapterImpl
	standardAdapterCore
}

// Standard Adapter Core methods
// *****************************

// Cache returns the cache of the adapter.
func (s *standardAdapterCore) Cache() *sdpcache.Cache {
	return s.cache
}

// Type returns the type of the adapter.
func (s *standardAdapterCore) Type() string {
	return s.wrapper.Type()
}

// Name returns the name of the adapter.
func (s *standardAdapterCore) Name() string {
	return s.wrapper.Name()
}

// Scopes returns the scopes of the adapter.
func (s *standardAdapterCore) Scopes() []string {
	return s.wrapper.Scopes()
}

func (s *standardAdapterCore) validateScopes(scope string) error {
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

// Get retrieves a single item with a given scope and query.
func (s *standardAdapterCore) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
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

// Standard Adapter Implementation
// *******************************

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

	if s.sourceType == string(gcpshared.GCP) {
		// Validate predefined role and IAM permissions consistency
		if err := validatePredefinedRole(s.wrapper); err != nil {
			return err
		}
	}

	return protovalidate.Validate(s.Metadata())
}

// Listable Adapter Implementation
// ******************************

// List retrieves all items in a given scope.
func (s *standardListableAdapterImpl) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
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

func (s *standardListableAdapterImpl) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
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

// Metadata returns the metadata of the listable adapter.
func (s *standardListableAdapterImpl) Metadata() *sdp.AdapterMetadata {
	if s.wrapper.AdapterMetadata() != nil {
		return s.wrapper.AdapterMetadata()
	}

	a := &sdp.AdapterMetadata{
		Type:              s.wrapper.Type(),
		Category:          s.wrapper.Category(),
		DescriptiveName:   s.wrapper.ItemType().Readable(),
		TerraformMappings: s.wrapper.TerraformMappings(),
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get: true,
			GetDescription: fmt.Sprintf(
				"Get %s by \"%s\"",
				s.wrapper.ItemType().Readable(),
				s.wrapper.GetLookups().ReadableFormat(),
			),
			List: true,
			ListDescription: fmt.Sprintf(
				"List all %s items", s.wrapper.ItemType().Readable(),
			),
		},
	}

	if s.wrapper.PotentialLinks() != nil {
		for link := range s.wrapper.PotentialLinks() {
			a.PotentialLinks = append(a.PotentialLinks, link.String())
		}
	}

	return a
}

// Validate checks if the listable adapter is valid.
func (s *standardListableAdapterImpl) Validate() error {
	if s.cache == nil {
		return fmt.Errorf("cache is not initialized")
	}

	if s.sourceType == string(gcpshared.GCP) {
		// Validate predefined role and IAM permissions consistency
		if err := validatePredefinedRole(s.wrapper); err != nil {
			return err
		}
	}

	return protovalidate.Validate(s.Metadata())
}

// Searchable Adapter Implementation
// *********************************

// Search retrieves items based on a search query.
func (s *standardSearchableAdapterImpl) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
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

func (s *standardSearchableAdapterImpl) SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
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

// Metadata returns the metadata of the searchable adapter.
func (s *standardSearchableAdapterImpl) Metadata() *sdp.AdapterMetadata {
	if s.wrapper.AdapterMetadata() != nil {
		return s.wrapper.AdapterMetadata()
	}

	a := &sdp.AdapterMetadata{
		Type:              s.wrapper.Type(),
		Category:          s.wrapper.Category(),
		DescriptiveName:   s.wrapper.ItemType().Readable(),
		TerraformMappings: s.wrapper.TerraformMappings(),
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get: true,
			GetDescription: fmt.Sprintf(
				"Get %s by \"%s\"",
				s.wrapper.ItemType().Readable(),
				s.wrapper.GetLookups().ReadableFormat(),
			),
			Search: true,
			SearchDescription: fmt.Sprintf(
				"Search for %s by \"%s\"",
				s.wrapper.ItemType().Readable(),
				expectedSearchQueryFormat(s.searchable.SearchLookups()),
			),
		},
	}

	if s.wrapper.PotentialLinks() != nil {
		for link := range s.wrapper.PotentialLinks() {
			a.PotentialLinks = append(a.PotentialLinks, link.String())
		}
	}

	return a
}

// Validate checks if the searchable adapter is valid.
func (s *standardSearchableAdapterImpl) Validate() error {
	if s.cache == nil {
		return fmt.Errorf("cache is not initialized")
	}
	if s.sourceType == string(gcpshared.GCP) {
		// Validate predefined role and IAM permissions consistency
		if err := validatePredefinedRole(s.wrapper); err != nil {
			return err
		}
	}

	return protovalidate.Validate(s.Metadata())
}

// Searchable and Listable Adapter Implementation
// **********************************************

// Metadata returns the metadata of the searchable+listable adapter.
func (s *standardSearchableListableAdapterImpl) Metadata() *sdp.AdapterMetadata {
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
		List: true,
		ListDescription: fmt.Sprintf(
			"List all %s items", s.wrapper.ItemType().Readable()),
		Search: true,
		SearchDescription: fmt.Sprintf(
			"Search for %s by \"%s\"",
			s.wrapper.ItemType().Readable(),
			expectedSearchQueryFormat(s.searchableImpl.searchable.SearchLookups()),
		),
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

// Validate checks if the searchable+listable adapter is valid.
func (s *standardSearchableListableAdapterImpl) Validate() error {
	if s.cache == nil {
		return fmt.Errorf("cache is not initialized")
	}
	if s.sourceType == string(gcpshared.GCP) {
		// Validate predefined role and IAM permissions consistency
		if err := validatePredefinedRole(s.wrapper); err != nil {
			return err
		}
	}

	return protovalidate.Validate(s.Metadata())
}

// List delegates to the listable implementation.
func (s *standardSearchableListableAdapterImpl) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	return s.listableImpl.List(ctx, scope, ignoreCache)
}

// ListStream delegates to the listable implementation.
func (s *standardSearchableListableAdapterImpl) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	s.listableImpl.ListStream(ctx, scope, ignoreCache, stream)
}

// Search delegates to the searchable implementation.
func (s *standardSearchableListableAdapterImpl) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	return s.searchableImpl.Search(ctx, scope, query, ignoreCache)
}

// SearchStream delegates to the searchable implementation.
func (s *standardSearchableListableAdapterImpl) SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	s.searchableImpl.SearchStream(ctx, scope, query, ignoreCache, stream)
}

// expectedSearchQueryFormat generates a readable format for the search query.
func expectedSearchQueryFormat(keywords []ItemTypeLookups) string {
	var readableKeywords []string
	for _, keyword := range keywords {
		readableKeywords = append(readableKeywords, keyword.ReadableFormat())
	}

	return strings.Join(readableKeywords, "\" or \"")
}

// validatePredefinedRole validates that the wrapper's predefined role and IAM permissions are consistent
func validatePredefinedRole(wrapper Wrapper) error {
	predefinedRole := wrapper.PredefinedRole()
	iamPermissions := wrapper.IAMPermissions()

	// Predefined role must be specified
	if predefinedRole == "" {
		return fmt.Errorf("wrapper %s must specify a predefined role", wrapper.Type())
	}

	// Check if the predefined role exists in the map
	role, exists := gcpshared.PredefinedRoles[predefinedRole]
	if !exists {
		return fmt.Errorf("predefined role %s is not found in PredefinedRoles map", predefinedRole)
	}

	// Check if all IAM permissions from the wrapper exist in the predefined role's IAMPermissions
	for _, perm := range iamPermissions {
		found := false
		for _, rolePerm := range role.IAMPermissions {
			if perm == rolePerm {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("IAM permission %s from wrapper is not included in predefined role %s IAMPermissions", perm, predefinedRole)
		}
	}

	return nil
}
