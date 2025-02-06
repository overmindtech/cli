package adapters

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type PodClient struct {
	GetError  error
	ListError error
}

func (p PodClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Pod, error) {
	if p.GetError != nil {
		return nil, p.GetError
	}

	uid := uuid.NewString()

	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			UID:               types.UID(uid),
			ResourceVersion:   "9164",
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					Name: "kube-api-access-hgq4d",
				},
			},
			RestartPolicy:      "Always",
			DNSPolicy:          "ClusterFirst",
			ServiceAccountName: "default",
			NodeName:           "minikube",
			Containers: []v1.Container{
				{
					Env: []v1.EnvVar{
						{
							Name:  "INTERESTING_URL",
							Value: "http://example.com",
						},
					},
				},
			},
		},
		Status: v1.PodStatus{
			Phase:  "Running",
			HostIP: "10.0.0.1",
			PodIP:  "10.244.0.6",
		},
	}, nil
}

func (p PodClient) List(ctx context.Context, opts metav1.ListOptions) (*v1.PodList, error) {
	if p.ListError != nil {
		return nil, p.ListError
	}

	uid := uuid.NewString()

	return &v1.PodList{
		Items: []v1.Pod{
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "foo",
					Namespace:         "default",
					UID:               types.UID(uid),
					ResourceVersion:   "9164",
					CreationTimestamp: metav1.NewTime(time.Now()),
				},
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: "kube-api-access-hgq4d",
						},
					},
					RestartPolicy:      "Always",
					DNSPolicy:          "ClusterFirst",
					ServiceAccountName: "default",
					NodeName:           "minikube",
				},
				Status: v1.PodStatus{
					Phase:  "Running",
					HostIP: "10.0.0.1",
					PodIP:  "10.244.0.6",
				},
			},
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "bar",
					Namespace:         "default",
					UID:               types.UID(uid),
					ResourceVersion:   "9164",
					CreationTimestamp: metav1.NewTime(time.Now()),
				},
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: "kube-api-access-c43w1",
						},
					},
					RestartPolicy:      "Always",
					DNSPolicy:          "ClusterFirst",
					ServiceAccountName: "default",
					NodeName:           "minikube",
				},
				Status: v1.PodStatus{
					Phase:  "Running",
					HostIP: "10.0.0.1",
					PodIP:  "10.244.0.7",
				},
			},
		},
	}, nil
}

func createAdapter(namespaced bool) *KubeTypeAdapter[*v1.Pod, *v1.PodList] {
	var clusterInterfaceBuilder ClusterInterfaceBuilder[*v1.Pod, *v1.PodList]
	var namespacedInterfaceBuilder NamespacedInterfaceBuilder[*v1.Pod, *v1.PodList]

	if namespaced {
		namespacedInterfaceBuilder = func(namespace string) ItemInterface[*v1.Pod, *v1.PodList] {
			return PodClient{}
		}
	} else {
		clusterInterfaceBuilder = func() ItemInterface[*v1.Pod, *v1.PodList] {
			return PodClient{}
		}
	}

	return &KubeTypeAdapter[*v1.Pod, *v1.PodList]{
		ClusterInterfaceBuilder:    clusterInterfaceBuilder,
		NamespacedInterfaceBuilder: namespacedInterfaceBuilder,
		ListExtractor: func(p *v1.PodList) ([]*v1.Pod, error) {
			pods := make([]*v1.Pod, len(p.Items))

			for i := range p.Items {
				pods[i] = &p.Items[i]
			}

			return pods, nil
		},
		LinkedItemQueryExtractor: func(p *v1.Pod, scope string) ([]*sdp.LinkedItemQuery, error) {
			queries := make([]*sdp.LinkedItemQuery, 0)

			if p.Spec.NodeName == "" {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "node",
						Method: sdp.QueryMethod_GET,
						Query:  p.Spec.NodeName,
						Scope:  scope,
					},
				})
			}

			return queries, nil
		},
		HealthExtractor: func(resource *v1.Pod) *sdp.Health {
			return sdp.Health_HEALTH_OK.Enum()
		},
		AutoQueryExtract: true,
		TypeName:         "Pod",
		ClusterName:      "minikube",
		Namespaces:       []string{"default", "app1"},
	}
}

func TestAdapterValidate(t *testing.T) {
	t.Run("fully populated adapter", func(t *testing.T) {
		t.Parallel()
		adapter := createAdapter(false)
		err := adapter.Validate()

		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}
	})

	t.Run("missing ClusterInterfaceBuilder", func(t *testing.T) {
		t.Parallel()
		adapter := createAdapter(false)
		adapter.ClusterInterfaceBuilder = nil

		err := adapter.Validate()

		if err == nil {
			t.Errorf("expected error, got none")
		}
	})

	t.Run("missing ListExtractor", func(t *testing.T) {
		t.Parallel()
		adapter := createAdapter(false)
		adapter.ListExtractor = nil

		err := adapter.Validate()

		if err == nil {
			t.Errorf("expected error, got none")
		}
	})

	t.Run("missing TypeName", func(t *testing.T) {
		t.Parallel()
		adapter := createAdapter(false)
		adapter.TypeName = ""

		err := adapter.Validate()

		if err == nil {
			t.Errorf("expected error, got none")
		}
	})

	t.Run("missing ClusterName", func(t *testing.T) {
		t.Parallel()
		adapter := createAdapter(false)
		adapter.ClusterName = ""

		err := adapter.Validate()

		if err == nil {
			t.Errorf("expected error, got none")
		}
	})

	t.Run("missing namespaces", func(t *testing.T) {
		t.Run("when namespaced", func(t *testing.T) {
			t.Parallel()
			adapter := createAdapter(true)
			adapter.Namespaces = nil

			err := adapter.Validate()

			if err == nil {
				t.Errorf("expected error, got none")
			}

			adapter.Namespaces = []string{}

			err = adapter.Validate()

			if err == nil {
				t.Errorf("expected error, got none")
			}
		})

		t.Run("when not namespaced", func(t *testing.T) {
			t.Parallel()
			adapter := createAdapter(false)
			adapter.Namespaces = nil

			err := adapter.Validate()

			if err != nil {
				t.Errorf("expected no error, got %s", err)
			}

			adapter.Namespaces = []string{}

			err = adapter.Validate()

			if err != nil {
				t.Errorf("expected no error, got %s", err)
			}
		})

	})
}

func TestType(t *testing.T) {
	adapter := createAdapter(false)

	if adapter.Type() != "Pod" {
		t.Errorf("expected type 'Pod', got %s", adapter.Type())
	}
}

func TestName(t *testing.T) {
	adapter := createAdapter(false)

	if adapter.Name() == "" {
		t.Errorf("expected non-empty name, got none")
	}
}

func TestScopes(t *testing.T) {
	t.Run("when namespaced", func(t *testing.T) {
		adapter := createAdapter(true)

		if len(adapter.Scopes()) != len(adapter.Namespaces) {
			t.Errorf("expected %d scopes, got %d", len(adapter.Namespaces), len(adapter.Scopes()))
		}
	})

	t.Run("when not namespaced", func(t *testing.T) {
		adapter := createAdapter(false)

		if len(adapter.Scopes()) != 1 {
			t.Errorf("expected 1 scope, got %d", len(adapter.Scopes()))
		}
	})
}

func TestAdapterGet(t *testing.T) {
	t.Run("get existing item", func(t *testing.T) {
		adapter := createAdapter(false)

		item, err := adapter.Get(context.Background(), "foo", "example", false)

		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if item == nil {
			t.Errorf("expected item, got none")
		}

		if item.UniqueAttributeValue() != "example" {
			t.Errorf("expected item with unique attribute value 'example', got %s", item.UniqueAttributeValue())
		}

		if item.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("expected item with health HEALTH_OK, got %s", item.GetHealth())
		}

		if item.GetType() != "Pod" {
			t.Errorf("expected item with type Pod, got %s", item.GetType())
		}

		var foundAutomaticLink bool
		for _, q := range item.GetLinkedItemQueries() {
			if q.GetQuery().GetType() == "http" && q.GetQuery().GetQuery() == "http://example.com" {
				foundAutomaticLink = true
				break
			}
		}

		if !foundAutomaticLink {
			t.Errorf("expected automatic link to http://example.com, got none")
		}
	})

	t.Run("get non-existent item", func(t *testing.T) {
		adapter := createAdapter(false)
		adapter.ClusterInterfaceBuilder = func() ItemInterface[*v1.Pod, *v1.PodList] {
			return PodClient{
				GetError: &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: "not found",
				},
			}
		}

		_, err := adapter.Get(context.Background(), "foo", "example", false)

		if err == nil {
			t.Errorf("expected error, got none")
		}
	})
}

func TestFailingQueryExtractor(t *testing.T) {
	adapter := createAdapter(false)
	adapter.LinkedItemQueryExtractor = func(_ *v1.Pod, _ string) ([]*sdp.LinkedItemQuery, error) {
		return nil, errors.New("failed to extract queries")
	}

	_, err := adapter.Get(context.Background(), "foo", "example", false)

	if err == nil {
		t.Errorf("expected error, got none")
	}
}

func TestList(t *testing.T) {
	t.Run("when namespaced", func(t *testing.T) {
		adapter := createAdapter(true)

		items, err := adapter.List(context.Background(), "foo.bar", false)

		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if len(items) != 2 {
			t.Errorf("expected 2 items, got %d", len(items))
		}

		if items[0].GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("expected item with health HEALTH_OK, got %s", items[0].GetHealth())
		}
	})

	t.Run("when not namespaced", func(t *testing.T) {
		adapter := createAdapter(false)

		items, err := adapter.List(context.Background(), "foo", false)

		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if len(items) != 2 {
			t.Errorf("expected 2 items, got %d", len(items))
		}
	})

	t.Run("with failing list extractor", func(t *testing.T) {
		adapter := createAdapter(false)
		adapter.ListExtractor = func(_ *v1.PodList) ([]*v1.Pod, error) {
			return nil, errors.New("failed to extract list")
		}

		_, err := adapter.List(context.Background(), "foo", false)

		if err == nil {
			t.Errorf("expected error, got none")
		}
	})

	t.Run("with failing query extractor", func(t *testing.T) {
		adapter := createAdapter(false)
		adapter.LinkedItemQueryExtractor = func(_ *v1.Pod, _ string) ([]*sdp.LinkedItemQuery, error) {
			return nil, errors.New("failed to extract queries")
		}

		_, err := adapter.List(context.Background(), "foo", false)

		if err == nil {
			t.Errorf("expected error, got none")
		}
	})
}

func TestSearch(t *testing.T) {
	t.Run("with a valid query", func(t *testing.T) {
		adapter := createAdapter(false)

		items, err := adapter.Search(context.Background(), "foo", "{\"labelSelector\":\"app=foo\"}", false)

		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if len(items) != 2 {
			t.Errorf("expected 2 item, got %d", len(items))
		}
	})

	t.Run("with an invalid query", func(t *testing.T) {
		adapter := createAdapter(false)

		_, err := adapter.Search(context.Background(), "foo", "{{{{}", false)

		if err == nil {
			t.Errorf("expected error, got none")
		}
	})
}

func TestRedact(t *testing.T) {
	adapter := createAdapter(true)
	adapter.Redact = func(resource *v1.Pod) *v1.Pod {
		resource.Spec.Hostname = "redacted"

		return resource
	}

	item, err := adapter.Get(context.Background(), "cluster.namespace", "test", false)

	if err != nil {
		t.Error(err)
	}

	hostname, err := item.GetAttributes().Get("spec.hostname")

	if err != nil {
		t.Error(err)
	}

	if hostname != "redacted" {
		t.Errorf("expected hostname to be redacted, got %v", hostname)
	}
}

type QueryTest struct {
	ExpectedType   string
	ExpectedMethod sdp.QueryMethod
	ExpectedQuery  string
	ExpectedScope  string

	// Expect the query to match a regex, this takes precedence over
	// ExpectedQuery
	ExpectedQueryMatches *regexp.Regexp
}

type QueryTests []QueryTest

func (i QueryTests) Execute(t *testing.T, item *sdp.Item) {
	t.Helper()

	for _, test := range i {
		var found bool

		for _, lir := range item.GetLinkedItemQueries() {
			if lirMatches(test, lir) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("could not find linked item request in %v requests.\nType: %v\nQuery: %v\nScope: %v", len(item.GetLinkedItemQueries()), test.ExpectedType, test.ExpectedQuery, test.ExpectedScope)
		}
	}
}

func lirMatches(test QueryTest, req *sdp.LinkedItemQuery) bool {
	if req.GetQuery() != nil {
		if test.ExpectedMethod != req.GetQuery().GetMethod() {
			return false
		}
		if test.ExpectedScope != req.GetQuery().GetScope() {
			return false
		}
		if test.ExpectedType != req.GetQuery().GetType() {
			return false
		}

		if test.ExpectedQueryMatches != nil {
			if !test.ExpectedQueryMatches.MatchString(req.GetQuery().GetQuery()) {
				return false
			}
		} else {
			if test.ExpectedQuery != req.GetQuery().GetQuery() {
				return false
			}
		}
	}

	// TODO: check for blast radius differences

	return true
}

type AdapterTests struct {
	// The adapter under test
	Adapter discovery.ListableAdapter

	// The get query to test
	GetQuery      string
	GetScope      string
	GetQueryTests QueryTests

	// If this is set,. the get query is determined by running a list, then
	// finding the first item that matches this regexp
	GetQueryRegexp *regexp.Regexp

	// YAML to apply before testing, it will be removed after
	SetupYAML string

	// An optional function to wait to return true before running the tests. It
	// is passed the current item that Get tests will be run against, and should
	// return a boolean indicating whether the tests should continue or wait
	Wait func(item *sdp.Item) bool
}

func (s AdapterTests) Execute(t *testing.T) {
	t.Helper()

	if s.SetupYAML != "" {
		err := CurrentCluster.Apply(s.SetupYAML)
		if err != nil {
			t.Fatal(fmt.Errorf("failed to apply setup YAML: %w", err))
		}

		t.Cleanup(func() {
			err = CurrentCluster.Delete(s.SetupYAML)
			if err != nil {
				t.Fatal(fmt.Errorf("failed to delete setup YAML: %w", err))
			}
		})
	}

	var getQuery string

	if s.GetQueryRegexp != nil {
		items, err := s.Adapter.List(context.Background(), s.GetScope, false)

		if err != nil {
			t.Fatal(err)
		}

		for _, item := range items {
			if s.GetQueryRegexp.MatchString(item.UniqueAttributeValue()) {
				getQuery = item.UniqueAttributeValue()
				break
			}
		}
	} else {
		getQuery = s.GetQuery
	}

	if s.Wait != nil {
		t.Log("waiting before executing tests")
		err := WaitFor(20*time.Second, func() bool {
			item, err := s.Adapter.Get(context.Background(), s.GetScope, getQuery, true)

			if err != nil {
				return false
			}

			return s.Wait(item)
		})

		if err != nil {
			t.Fatalf("timed out waiting before starting tests: %v", err)
		}
	}

	t.Run(s.Adapter.Name(), func(t *testing.T) {
		if getQuery != "" {
			t.Run(fmt.Sprintf("GET:%v", getQuery), func(t *testing.T) {
				item, err := s.Adapter.Get(context.Background(), s.GetScope, getQuery, false)

				if err != nil {
					t.Fatal(err)
				}

				if item == nil {
					t.Errorf("expected item, got none")
				}

				if err = item.Validate(); err != nil {
					t.Error(err)
				}

				s.GetQueryTests.Execute(t, item)
			})
		}

		t.Run("LIST", func(t *testing.T) {
			items, err := s.Adapter.List(context.Background(), s.GetScope, false)

			if err != nil {
				t.Fatal(err)
			}

			if len(items) == 0 {
				t.Errorf("expected items, got none")
			}

			itemMap := make(map[string]*sdp.Item)

			for _, item := range items {
				itemMap[item.UniqueAttributeValue()] = item

				if err = item.Validate(); err != nil {
					t.Error(err)
				}
			}

			if len(itemMap) != len(items) {
				t.Errorf("expected %v unique items, got %v", len(items), len(itemMap))
			}
		})

		t.Run("Adapter Metadata", func(t *testing.T) {
			metadata := s.Adapter.Metadata()
			if metadata == nil {
				t.Fatal("expected metadata, got none")
			}

			if metadata.GetType() == "" {
				t.Error("expected metadata type, got none")
			}

			if metadata.GetDescriptiveName() == "" {
				t.Error("expected metadata descriptive name, got none")
			}
		})
	})
}

// WaitFor waits for a condition to be true, or returns an error if the timeout
func WaitFor(timeout time.Duration, run func() bool) error {
	start := time.Now()

	for {
		if run() {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("timeout exceeded")
		}

		time.Sleep(250 * time.Millisecond)
	}
}

func TestObjectReferenceToQuery(t *testing.T) {
	t.Run("with a valid object reference", func(t *testing.T) {
		ref := &v1.ObjectReference{
			Kind:      "Pod",
			Namespace: "default",
			Name:      "foo",
		}

		b := &sdp.BlastPropagation{}

		query := ObjectReferenceToQuery(ref, ScopeDetails{
			ClusterName: "test-cluster",
			Namespace:   "default",
		}, b)

		if query.GetQuery().GetType() != "Pod" {
			t.Errorf("expected type Pod, got %s", query.GetQuery().GetType())
		}

		if query.GetQuery().GetQuery() != "foo" {
			t.Errorf("expected query to be foo, got %s", query.GetQuery().GetQuery())
		}

		if query.GetQuery().GetScope() != "test-cluster.default" {
			t.Errorf("expected scope to be test-cluster.default, got %s", query.GetQuery().GetScope())
		}

		if query.GetBlastPropagation() != b {
			t.Errorf("expected blast propagation to be %v, got %v", b, query.GetBlastPropagation())
		}
	})

	t.Run("with a nil object reference", func(t *testing.T) {
		query := ObjectReferenceToQuery(nil, ScopeDetails{}, nil)

		if query != nil {
			t.Errorf("expected nil query, got %v", query)
		}
	})
}
