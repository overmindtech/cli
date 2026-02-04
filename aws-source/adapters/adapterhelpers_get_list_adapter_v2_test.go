package adapters

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetListAdapterV2Type(t *testing.T) {
	s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
		ItemType: "foo",
	}

	if s.Type() != "foo" {
		t.Errorf("expected type to be foo got %v", s.Type())
	}
}

func TestGetListAdapterV2Name(t *testing.T) {
	s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
		ItemType: "foo",
	}

	if s.Name() != "foo-adapter" {
		t.Errorf("expected type to be foo-adapter got %v", s.Name())
	}
}

func TestGetListAdapterV2Scopes(t *testing.T) {
	s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
		AccountID: "foo",
		Region:    "bar",
	}

	if s.Scopes()[0] != "foo.bar" {
		t.Errorf("expected scope to be foo.bar, got %v", s.Scopes()[0])
	}
}

func TestGetListAdapterV2Get(t *testing.T) {
	t.Run("with no errors", func(t *testing.T) {
		s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ItemMapper: func(query *string, scope, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
			ListTagsFunc: func(ctx context.Context, s1 string, s2 struct{}) (map[string]string, error) {
				return map[string]string{
					"foo": "bar",
				}, nil
			},
			cache: sdpcache.NewNoOpCache(),
		}

		item, err := s.Get(context.Background(), "12345.eu-west-2", "", false)
		if err != nil {
			t.Error(err)
		}

		if item.GetTags()["foo"] != "bar" {
			t.Errorf("expected tag foo to be bar, got %v", item.GetTags()["foo"])
		}
	})

	t.Run("with an error in the GetFunc", func(t *testing.T) {
		s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", errors.New("get func error")
			},
			ItemMapper: func(query *string, scope, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
			cache: sdpcache.NewNoOpCache(),
		}

		if _, err := s.Get(context.Background(), "12345.eu-west-2", "", false); err == nil {
			t.Error("expected error got nil")
		}
	})

	t.Run("with an error in the mapper", func(t *testing.T) {
		s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ItemMapper: func(query *string, scope, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, errors.New("mapper error")
			},
			cache: sdpcache.NewNoOpCache(),
		}

		if _, err := s.Get(context.Background(), "12345.eu-west-2", "", false); err == nil {
			t.Error("expected error got nil")
		}
	})
}

func TestGetListAdapterV2ListStream(t *testing.T) {
	t.Run("with no errors", func(t *testing.T) {
		s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListFunc: func(ctx context.Context, client struct{}, input string) ([]string, error) {
				return []string{"one", "two"}, nil
			},
			ItemMapper: func(query *string, scope, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
			ListExtractor: func(ctx context.Context, output []string, client struct{}) ([]string, error) {
				return output, nil
			},
			ListTagsFunc: func(ctx context.Context, s1 string, s2 struct{}) (map[string]string, error) {
				return map[string]string{
					"foo": "bar",
				}, nil
			},
			InputMapperList: func(scope string) (string, error) {
				return "input", nil
			},
			cache: sdpcache.NewNoOpCache(),
		}

		stream := discovery.NewRecordingQueryResultStream()
		s.ListStream(context.Background(), "12345.eu-west-2", false, stream)

		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Error(errs)
		}

		items := stream.GetItems()
		if len(items) != 2 {
			t.Errorf("expected 2 items, got %v", len(items))
		}
	})

	t.Run("with an error in the ListFunc", func(t *testing.T) {
		s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, errors.New("list func error")
			},
			ItemMapper: func(query *string, scope, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
			cache: sdpcache.NewNoOpCache(),
		}

		stream := discovery.NewRecordingQueryResultStream()
		s.ListStream(context.Background(), "12345.eu-west-2", false, stream)

		errs := stream.GetErrors()
		if len(errs) == 0 {
			t.Error("expected errors got none")
		}
	})

	t.Run("with an error in the mapper", func(t *testing.T) {
		s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListExtractor: func(ctx context.Context, output []string, client struct{}) ([]string, error) {
				return output, nil
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, nil
			},
			ItemMapper: func(query *string, scope, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, errors.New("mapper error")
			},
			InputMapperList: func(scope string) (string, error) {
				return "input", nil
			},
			cache: sdpcache.NewNoOpCache(),
		}

		stream := discovery.NewRecordingQueryResultStream()
		s.ListStream(context.Background(), "12345.eu-west-2", false, stream)

		errs := stream.GetErrors()
		if len(errs) != 2 {
			t.Errorf("expected 2 errors got %v", len(errs))
		}

		items := stream.GetItems()
		if len(items) != 0 {
			t.Errorf("expected no items, got %v", len(items))
		}
	})
}

// MockPaginator is a mock implementation of the Paginator interface
type MockPaginator struct {
	pages    [][]string
	pageIdx  int
	hasPages bool
}

func (p *MockPaginator) HasMorePages() bool {
	return p.hasPages && p.pageIdx < len(p.pages)
}

func (p *MockPaginator) NextPage(ctx context.Context, opts ...func(struct{})) ([]string, error) {
	if !p.HasMorePages() {
		return nil, errors.New("no more pages available")
	}
	page := p.pages[p.pageIdx]
	p.pageIdx++
	return page, nil
}

func TestListFuncPaginatorBuilder(t *testing.T) {
	adapter := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
		ItemType:  "test-item",
		AccountID: "foo",
		Region:    "eu-west-2",
		Client:    struct{}{},
		InputMapperList: func(scope string) (string, error) {
			return "test-input", nil
		},
		ListFuncPaginatorBuilder: func(client struct{}, input string) Paginator[[]string, struct{}] {
			return &MockPaginator{
				pages: [][]string{
					{"item1", "item2"},
					{"item3", "item4"},
				},
				hasPages: true,
			}
		},
		ListExtractor: func(ctx context.Context, output []string, client struct{}) ([]string, error) {
			return output, nil
		},
		ItemMapper: func(query *string, scope string, awsItem string) (*sdp.Item, error) {
			attrs, _ := sdp.ToAttributes(map[string]interface{}{
				"id": awsItem,
			})
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "id",
				Attributes:      attrs,
				Scope:           scope,
			}, nil
		},
		GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
			return "", nil
		},
		cache: sdpcache.NewNoOpCache(),
	}

	stream := discovery.NewRecordingQueryResultStream()
	adapter.ListStream(context.Background(), "foo.eu-west-2", false, stream)

	errs := stream.GetErrors()
	if len(errs) > 0 {
		t.Error(errs)
	}

	items := stream.GetItems()
	if len(items) != 4 {
		t.Errorf("expected 4 items, got %v", len(items))
	}
}

func TestGetListAdapterV2Caching(t *testing.T) {
	ctx := context.Background()
	generation := 0
	s := GetListAdapterV2[string, []string, string, struct{}, struct{}]{
		ItemType:  "test-type",
		Region:    "eu-west-2",
		AccountID: "foo",
		cache:     sdpcache.NewCache(ctx),
		GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
			generation += 1
			return fmt.Sprintf("%v", generation), nil
		},
		ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
			generation += 1
			return []string{fmt.Sprintf("%v", generation)}, nil
		},
		ListExtractor: func(ctx context.Context, output []string, client struct{}) ([]string, error) {
			return output, nil
		},
		InputMapperList: func(scope string) (string, error) {
			return "input", nil
		},
		ItemMapper: func(query *string, scope string, output string) (*sdp.Item, error) {
			return &sdp.Item{
				Scope:           "foo.eu-west-2",
				Type:            "test-type",
				UniqueAttribute: "name",
				Attributes: &sdp.ItemAttributes{
					AttrStruct: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name":       structpb.NewStringValue("test-item"),
							"generation": structpb.NewStringValue(output),
						},
					},
				},
			}, nil
		},
	}

	t.Run("get", func(t *testing.T) {
		// get
		first, err := s.Get(ctx, "foo.eu-west-2", "test-item", false)
		if err != nil {
			t.Fatal(err)
		}
		firstGen, err := first.GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}

		// get again
		withCache, err := s.Get(ctx, "foo.eu-west-2", "test-item", false)
		if err != nil {
			t.Fatal(err)
		}
		withCacheGen, err := withCache.GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}

		if firstGen != withCacheGen {
			t.Errorf("with cache: expected generation %v, got %v", firstGen, withCacheGen)
		}

		// get ignore cache
		withoutCache, err := s.Get(ctx, "foo.eu-west-2", "test-item", true)
		if err != nil {
			t.Fatal(err)
		}
		withoutCacheGen, err := withoutCache.GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}
		if withoutCacheGen == firstGen {
			t.Errorf("with cache: expected generation %v, got %v", firstGen, withoutCacheGen)
		}
	})

	t.Run("list", func(t *testing.T) {
		stream := discovery.NewRecordingQueryResultStream()

		// First call
		s.ListStream(ctx, "foo.eu-west-2", false, stream)
		// Second call with caching
		s.ListStream(ctx, "foo.eu-west-2", false, stream)
		// Third call without caching
		s.ListStream(ctx, "foo.eu-west-2", true, stream)

		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Error(errs)
		}

		items := stream.GetItems()
		firstGen, err := items[0].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}
		withCacheGen, err := items[1].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}
		withoutCacheGen, err := items[2].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}

		if firstGen != withCacheGen {
			t.Errorf("with cache: expected generation %v, got %v", firstGen, withCacheGen)
		}

		if withoutCacheGen == firstGen {
			t.Errorf("with cache: expected generation %v, got %v", firstGen, withoutCacheGen)
		}
	})
}

// TestGetListAdapterV2_ListExtractorErrorNoNotFoundCache tests that when ListExtractor fails,
// we don't incorrectly cache NOTFOUND. The error should be sent, but NOTFOUND should not be cached
// because the failure was due to extraction errors, not because items don't exist.
func TestGetListAdapterV2_ListExtractorErrorNoNotFoundCache(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	listCalls := 0

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapterV2[*MockInput, *MockOutput, *MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			return nil, errors.New("should not be called in LIST test")
		},
		InputMapperList: func(scope string) (*MockInput, error) {
			return &MockInput{}, nil
		},
		ListFunc: func(ctx context.Context, client *MockClient, input *MockInput) (*MockOutput, error) {
			listCalls++
			// Return a valid output that indicates items exist
			return &MockOutput{}, nil
		},
		ListExtractor: func(ctx context.Context, output *MockOutput, client *MockClient) ([]*MockAWSItem, error) {
			// Simulate extraction failure - this should NOT result in NOTFOUND caching
			return nil, errors.New("extraction failed")
		},
		ItemMapper: func(query *string, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "name",
				Attributes:      &sdp.ItemAttributes{},
				Scope:           scope,
			}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call - ListExtractor fails, should send error but NOT cache NOTFOUND
	stream1 := discovery.NewRecordingQueryResultStream()
	adapter.ListStream(ctx, "123456789012.us-east-1", false, stream1)

	if len(stream1.GetItems()) != 0 {
		t.Errorf("Expected 0 items, got %d", len(stream1.GetItems()))
	}
	if len(stream1.GetErrors()) != 1 {
		t.Errorf("Expected 1 error from ListExtractor failure, got %d", len(stream1.GetErrors()))
	}
	if listCalls != 1 {
		t.Errorf("Expected 1 ListFunc call, got %d", listCalls)
	}

	// Second call - should NOT hit cache (NOTFOUND was not cached), should try again
	stream2 := discovery.NewRecordingQueryResultStream()
	adapter.ListStream(ctx, "123456789012.us-east-1", false, stream2)

	if listCalls != 2 {
		t.Errorf("Expected 2 ListFunc calls (no cache hit because NOTFOUND was not cached), got %d", listCalls)
	}
	if len(stream2.GetItems()) != 0 {
		t.Errorf("Expected 0 items, got %d", len(stream2.GetItems()))
	}
	if len(stream2.GetErrors()) != 1 {
		t.Errorf("Expected 1 error from ListExtractor failure, got %d", len(stream2.GetErrors()))
	}
}
