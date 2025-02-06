package adapters

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultCacheDuration = 30 * time.Minute

// NamespacedInterfaceBuilder The function that create a client to query a
// namespaced resource. e.g. `CoreV1().Pods`
type NamespacedInterfaceBuilder[Resource metav1.Object, ResourceList any] func(namespace string) ItemInterface[Resource, ResourceList]

// ClusterInterfaceBuilder The function that create a client to query a
// cluster-wide resource. e.g. `CoreV1().Nodes`
type ClusterInterfaceBuilder[Resource metav1.Object, ResourceList any] func() ItemInterface[Resource, ResourceList]

// ItemInterface An interface that matches the `Get` and `List` methods for K8s
// resources since these are the ones that we use for getting Overmind data.
// Kube's clients are usually namespaced when they are created, so this
// interface is expected to only returns items from a single namespace
type ItemInterface[Resource metav1.Object, ResourceList any] interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (Resource, error)
	List(ctx context.Context, opts metav1.ListOptions) (ResourceList, error)
}

type KubeTypeAdapter[Resource metav1.Object, ResourceList any] struct {
	// The function that creates a client to query a namespaced resource. e.g.
	// `CoreV1().Pods`. Either this or `NamespacedInterfaceBuilder` must be
	// specified
	ClusterInterfaceBuilder ClusterInterfaceBuilder[Resource, ResourceList]

	// The function that creates a client to query a cluster-wide resource. e.g.
	// `CoreV1().Nodes`. Either this or `ClusterInterfaceBuilder` must be
	// specified
	NamespacedInterfaceBuilder NamespacedInterfaceBuilder[Resource, ResourceList]

	// A function that extracts a slice of Resources from a ResourceList
	ListExtractor func(ResourceList) ([]Resource, error)

	// A function that returns a list of linked item queries for a given
	// resource and scope
	LinkedItemQueryExtractor func(resource Resource, scope string) ([]*sdp.LinkedItemQuery, error)

	// A function that extracts health from the resource, this is optional
	HealthExtractor func(resource Resource) *sdp.Health

	// A function that redacts sensitive data from the resource, this is
	// optional
	Redact func(resource Resource) Resource

	// Whether to automatically extract the query from the item's attributes.
	// This should be enabled for resources that are likely to include
	// unstructured but interesting data like environment variables
	AutoQueryExtract bool

	// The type of items that this adapter should return. This should be the
	// "Kind" of the kubernetes resources, e.g. "Pod", "Node", "ServiceAccount"
	TypeName string
	// List of namespaces that this adapter should query
	Namespaces []string
	// The name of the cluster that this adapter is for. This is used to generate
	// scopes
	ClusterName string

	// AdapterMetadata for the adapter
	AdapterMetadata *sdp.AdapterMetadata

	CacheDuration time.Duration   // How long to cache items for
	cache         *sdpcache.Cache // The sdpcache of this adapter
	cacheInitMu   sync.Mutex      // Mutex to ensure cache is only initialised once
}

func (s *KubeTypeAdapter[Resource, ResourceList]) cacheDuration() time.Duration {
	if s.CacheDuration == 0 {
		return DefaultCacheDuration
	}

	return s.CacheDuration
}

func (s *KubeTypeAdapter[Resource, ResourceList]) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
	}
}

func (s *KubeTypeAdapter[Resource, ResourceList]) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
}

// validate Validates that the adapter is correctly set up
func (s *KubeTypeAdapter[Resource, ResourceList]) Validate() error {
	if s.NamespacedInterfaceBuilder == nil && s.ClusterInterfaceBuilder == nil {
		return errors.New("either NamespacedInterfaceBuilder or ClusterInterfaceBuilder must be specified")
	}

	if s.ListExtractor == nil {
		return errors.New("listExtractor must be specified")
	}

	if s.TypeName == "" {
		return errors.New("typeName must be specified")
	}

	if s.namespaced() && len(s.Namespaces) == 0 {
		return errors.New("namespaces must be specified when NamespacedInterfaceBuilder is specified")
	}

	if s.ClusterName == "" {
		return errors.New("clusterName must be specified")
	}

	return nil
}

// namespaced Returns whether the adapter is namespaced or not
func (s *KubeTypeAdapter[Resource, ResourceList]) namespaced() bool {
	return s.NamespacedInterfaceBuilder != nil
}

func (s *KubeTypeAdapter[Resource, ResourceList]) Type() string {
	return s.TypeName
}

func (s *KubeTypeAdapter[Resource, ResourceList]) Metadata() *sdp.AdapterMetadata {
	return s.AdapterMetadata
}

func (s *KubeTypeAdapter[Resource, ResourceList]) Name() string {
	return fmt.Sprintf("k8s-%v", s.TypeName)
}

func (s *KubeTypeAdapter[Resource, ResourceList]) Weight() int {
	return 10
}

func (s *KubeTypeAdapter[Resource, ResourceList]) Scopes() []string {
	namespaces := make([]string, 0)

	if s.namespaced() {
		for _, ns := range s.Namespaces {
			sd := ScopeDetails{
				ClusterName: s.ClusterName,
				Namespace:   ns,
			}

			namespaces = append(namespaces, sd.String())
		}
	} else {
		sd := ScopeDetails{
			ClusterName: s.ClusterName,
		}

		namespaces = append(namespaces, sd.String())
	}

	return namespaces
}

func (s *KubeTypeAdapter[Resource, ResourceList]) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query, ignoreCache)
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

	i, err := s.itemInterface(scope)
	if err != nil {
		err = &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
		s.cache.StoreError(err, s.cacheDuration(), ck)
		return nil, err
	}

	resource, err := i.Get(ctx, query, metav1.GetOptions{})
	if err != nil {
		statusErr := new(k8serr.StatusError)

		if errors.As(err, &statusErr) && statusErr.ErrStatus.Code == 404 {
			err = &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: statusErr.ErrStatus.Message,
			}
		}

		s.cache.StoreError(err, s.cacheDuration(), ck)
		return nil, err
	}

	item, err := s.resourceToItem(resource)
	if err != nil {
		s.cache.StoreError(err, s.cacheDuration(), ck)
		return nil, err
	}

	s.cache.StoreItem(item, s.cacheDuration(), ck)
	return item, nil
}

func (s *KubeTypeAdapter[Resource, ResourceList]) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_LIST, scope, s.Type(), "", ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		return cachedItems, nil
	}

	items, err := s.listWithOptions(ctx, scope, metav1.ListOptions{})
	if err != nil {
		s.cache.StoreError(err, s.cacheDuration(), ck)
		return nil, err
	}

	for _, item := range items {
		s.cache.StoreItem(item, s.cacheDuration(), ck)
	}

	return items, nil
}

// listWithOptions Runs the inbuilt list method with the given options
func (s *KubeTypeAdapter[Resource, ResourceList]) listWithOptions(ctx context.Context, scope string, opts metav1.ListOptions) ([]*sdp.Item, error) {
	i, err := s.itemInterface(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	list, err := i.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	resourceList, err := s.ListExtractor(list)
	if err != nil {
		return nil, err
	}

	items, err := s.resourcesToItems(resourceList)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *KubeTypeAdapter[Resource, ResourceList]) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	opts, err := QueryToListOptions(query)
	if err != nil {
		return nil, err
	}

	ck := sdpcache.CacheKeyFromParts(s.Name(), sdp.QueryMethod_SEARCH, scope, s.Type(), query)

	items, err := s.listWithOptions(ctx, scope, opts)
	if err != nil {
		s.cache.StoreError(err, s.cacheDuration(), ck)
		return nil, err
	}

	for _, item := range items {
		s.cache.StoreItem(item, s.cacheDuration(), ck)
	}

	return items, nil
}

// itemInterface Returns the correct interface depending on whether the adapter
// is namespaced or not
func (s *KubeTypeAdapter[Resource, ResourceList]) itemInterface(scope string) (ItemInterface[Resource, ResourceList], error) {
	// If this is a namespaced resource, then parse the scope to get the
	// namespace
	if s.namespaced() {
		details, err := ParseScope(scope, s.namespaced())

		if err != nil {
			return nil, err
		}

		return s.NamespacedInterfaceBuilder(details.Namespace), nil
	} else {
		return s.ClusterInterfaceBuilder(), nil
	}
}

var ignoredMetadataFields = []string{
	"managedFields",
	"binaryData",
	"immutable",
	"stringData",
}

func ignored(key string) bool {
	for _, ignoredKey := range ignoredMetadataFields {
		if key == ignoredKey {
			return true
		}
	}

	return false
}

// resourcesToItems Converts a slice of resources to a slice of items
func (s *KubeTypeAdapter[Resource, ResourceList]) resourcesToItems(resourceList []Resource) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, len(resourceList))

	var err error

	for i := range resourceList {
		items[i], err = s.resourceToItem(resourceList[i])

		if err != nil {
			return nil, err
		}

	}

	return items, nil
}

// resourceToItem Converts a resource to an item
func (s *KubeTypeAdapter[Resource, ResourceList]) resourceToItem(resource Resource) (*sdp.Item, error) {
	sd := ScopeDetails{
		ClusterName: s.ClusterName,
		Namespace:   resource.GetNamespace(),
	}

	// Redact sensitive data if required
	if s.Redact != nil {
		resource = s.Redact(resource)
	}

	attributes, err := sdp.ToAttributesViaJson(resource)

	if err != nil {
		return nil, err
	}

	// Promote the metadata to the top level
	if metadata, err := attributes.Get("metadata"); err == nil {
		// Cast to a type we can iterate over
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			for key, value := range metadataMap {
				// Check that the key isn't in the ignored list
				if !ignored(key) {
					attributes.Set(key, value)
				}
			}
		}

		// Remove the metadata attribute
		delete(attributes.GetAttrStruct().GetFields(), "metadata")
	}

	// Make sure the name is set
	attributes.Set("name", resource.GetName())

	item := &sdp.Item{
		Type:            s.TypeName,
		UniqueAttribute: "name",
		Scope:           sd.String(),
		Attributes:      attributes,
		Tags:            resource.GetLabels(),
	}

	// Automatically create links to owner references
	for _, ref := range resource.GetOwnerReferences() {
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   ref.Kind,
				Method: sdp.QueryMethod_GET,
				Query:  ref.Name,
				Scope:  sd.String(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to the owner will definitely affect the owned e.g.
				// changes to a deployment will affect the pods in that
				// deployment
				In: true,
				// Changes to the owned may affect the owner e.g. changing a
				// secret could affect a pod, but if all pods used that secret
				// then the change should propagate from the pods to the
				// deployment too
				Out: true,
			},
		})
	}

	if s.LinkedItemQueryExtractor != nil {
		// Add linked items
		newQueries, err := s.LinkedItemQueryExtractor(resource, sd.String())

		if err != nil {
			return nil, err
		}

		item.LinkedItemQueries = append(item.LinkedItemQueries, newQueries...)
	}

	if s.AutoQueryExtract {
		// Automatically extract queries from the item's attributes
		item.LinkedItemQueries = append(item.LinkedItemQueries, sdp.ExtractLinksFromAttributes(attributes)...)
	}

	if s.HealthExtractor != nil {
		item.Health = s.HealthExtractor(resource)
	}

	return item, nil
}

// ObjectReferenceToQuery Converts a K8s ObjectReference to a linked item
// request. Note that you must provide the parent scope since the reference
// could be an object in a different namespace, if it is we need to re-use the
// cluster name from the parent scope
func ObjectReferenceToQuery(ref *corev1.ObjectReference, parentScope ScopeDetails, blastProp *sdp.BlastPropagation) *sdp.LinkedItemQuery {
	if ref == nil {
		return nil
	}

	// Update the namespace, but keep the cluster the same
	parentScope.Namespace = ref.Namespace

	return &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   ref.Kind,
			Method: sdp.QueryMethod_GET, // Object references are to a specific object
			Query:  ref.Name,
			Scope:  parentScope.String(),
		},
		BlastPropagation: blastProp,
	}
}

// Returns the default supported query methods for a given resource type. The
// user must pass in the name of the resource type e.g. "Config Map"
func DefaultSupportedQueryMethods(name string) *sdp.AdapterSupportedQueryMethods {
	return &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		GetDescription:    fmt.Sprintf("Get a %v by name", name),
		List:              true,
		ListDescription:   fmt.Sprintf("List all %vs", name),
		Search:            true,
		SearchDescription: fmt.Sprintf(`Search for a %v using the ListOptions JSON format e.g. {"labelSelector": "app=wordpress"}`, name),
	}
}
