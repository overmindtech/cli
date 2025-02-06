package adapterhelpers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetListSourceType(t *testing.T) {
	s := GetListAdapter[string, struct{}, struct{}]{
		ItemType: "foo",
	}

	if s.Type() != "foo" {
		t.Errorf("expected type to be foo got %v", s.Type())
	}
}

func TestGetListSourceName(t *testing.T) {
	s := GetListAdapter[string, struct{}, struct{}]{
		ItemType: "foo",
	}

	if s.Name() != "foo-adapter" {
		t.Errorf("expected type to be foo-adapter got %v", s.Name())
	}
}

func TestGetListSourceScopes(t *testing.T) {
	s := GetListAdapter[string, struct{}, struct{}]{
		AccountID: "foo",
		Region:    "bar",
	}

	if s.Scopes()[0] != "foo.bar" {
		t.Errorf("expected scope to be foo.bar, got %v", s.Scopes()[0])
	}
}

func TestGetListSourceGet(t *testing.T) {
	t.Run("with no errors", func(t *testing.T) {
		s := GetListAdapter[string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, nil
			},
			ItemMapper: func(query, scope string, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
			ListTagsFunc: func(ctx context.Context, s1 string, s2 struct{}) (map[string]string, error) {
				return map[string]string{
					"foo": "bar",
				}, nil
			},
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
		s := GetListAdapter[string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", errors.New("get func error")
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, nil
			},
			ItemMapper: func(query, scope string, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
		}

		if _, err := s.Get(context.Background(), "12345.eu-west-2", "", false); err == nil {
			t.Error("expected error got nil")
		}
	})

	t.Run("with an error in the mapper", func(t *testing.T) {
		s := GetListAdapter[string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, nil
			},
			ItemMapper: func(query, scope string, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, errors.New("mapper error")
			},
		}

		if _, err := s.Get(context.Background(), "12345.eu-west-2", "", false); err == nil {
			t.Error("expected error got nil")
		}
	})
}

func TestGetListSourceList(t *testing.T) {
	t.Run("with no errors", func(t *testing.T) {
		s := GetListAdapter[string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, nil
			},
			ItemMapper: func(query, scope string, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
			ListTagsFunc: func(ctx context.Context, s1 string, s2 struct{}) (map[string]string, error) {
				return map[string]string{
					"foo": "bar",
				}, nil
			},
		}

		if items, err := s.List(context.Background(), "12345.eu-west-2", false); err != nil {
			t.Error(err)
		} else {
			if len(items) != 2 {
				t.Errorf("expected 2 items, got %v", len(items))
			}

			if items[0].GetTags()["foo"] != "bar" {
				t.Errorf("expected tag foo to be bar, got %v", items[0].GetTags()["foo"])
			}
		}
	})

	t.Run("with an error in the ListFunc", func(t *testing.T) {
		s := GetListAdapter[string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, errors.New("list func error")
			},
			ItemMapper: func(query, scope string, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
		}

		if _, err := s.List(context.Background(), "12345.eu-west-2", false); err == nil {
			t.Error("expected error got nil")
		}
	})

	t.Run("with an error in the mapper", func(t *testing.T) {
		s := GetListAdapter[string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, nil
			},
			ItemMapper: func(query, scope string, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, errors.New("mapper error")
			},
		}

		if items, err := s.List(context.Background(), "12345.eu-west-2", false); err != nil {
			t.Error(err)
		} else {
			if len(items) != 0 {
				t.Errorf("expected no items, got %v", len(items))
			}
		}
	})
}

func TestGetListSourceSearch(t *testing.T) {
	t.Run("with ARN search", func(t *testing.T) {
		s := GetListAdapter[string, struct{}, struct{}]{
			ItemType:  "person",
			Region:    "eu-west-2",
			AccountID: "12345",
			GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
				return "", nil
			},
			ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
				return []string{"", ""}, nil
			},
			ItemMapper: func(query, scope string, awsItem string) (*sdp.Item, error) {
				return &sdp.Item{}, nil
			},
		}

		t.Run("bad ARN", func(t *testing.T) {
			_, err := s.Search(context.Background(), "12345.eu-west-2", "query", false)

			if err == nil {
				t.Error("expected error because the ARN was bad")
			}
		})

		t.Run("good ARN but bad scope", func(t *testing.T) {
			_, err := s.Search(context.Background(), "12345.eu-west-2", "arn:aws:service:region:account:type/id", false)

			if err == nil {
				t.Error("expected error because the ARN had a bad scope")
			}
		})

		t.Run("good ARN", func(t *testing.T) {
			_, err := s.Search(context.Background(), "12345.eu-west-2", "arn:aws:service:eu-west-2:12345:type/id", false)
			if err != nil {
				t.Error(err)
			}
		})
	})
}

func TestGetListSourceCaching(t *testing.T) {
	ctx := context.Background()
	generation := 0
	s := GetListAdapter[string, struct{}, struct{}]{
		ItemType:  "test-type",
		Region:    "eu-west-2",
		AccountID: "foo",
		GetFunc: func(ctx context.Context, client struct{}, scope, query string) (string, error) {
			generation += 1
			return fmt.Sprintf("%v", generation), nil
		},
		ListFunc: func(ctx context.Context, client struct{}, scope string) ([]string, error) {
			generation += 1
			return []string{fmt.Sprintf("%v", generation)}, nil
		},
		SearchFunc: func(ctx context.Context, client struct{}, scope, query string) ([]string, error) {
			generation += 1
			return []string{fmt.Sprintf("%v", generation)}, nil
		},
		ItemMapper: func(query, scope string, output string) (*sdp.Item, error) {
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
		// list
		first, err := s.List(ctx, "foo.eu-west-2", false)
		if err != nil {
			t.Fatal(err)
		}
		firstGen, err := first[0].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}

		// list again
		withCache, err := s.List(ctx, "foo.eu-west-2", false)
		if err != nil {
			t.Fatal(err)
		}
		withCacheGen, err := withCache[0].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}

		if firstGen != withCacheGen {
			t.Errorf("with cache: expected generation %v, got %v", firstGen, withCacheGen)
		}

		// list ignore cache
		withoutCache, err := s.List(ctx, "foo.eu-west-2", true)
		if err != nil {
			t.Fatal(err)
		}
		withoutCacheGen, err := withoutCache[0].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}

		if withoutCacheGen == firstGen {
			t.Errorf("with cache: expected generation %v, got %v", firstGen, withoutCacheGen)
		}
	})

	t.Run("search", func(t *testing.T) {
		// search
		first, err := s.Search(ctx, "foo.eu-west-2", "arn:aws:test-type:eu-west-2:foo:test-item", false)
		if err != nil {
			t.Fatal(err)
		}
		firstGen, err := first[0].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}

		// Get the result of the search
		getCachedItem, err := s.Get(ctx, "foo.eu-west-2", "test-item", false)
		if err != nil {
			t.Fatal(err)
		}

		// Check that we get a valid item
		if err := getCachedItem.Validate(); err != nil {
			t.Fatal(err)
		}

		// Check the generation to make sure it was actually served from the cache
		cachedGeneration, _ := getCachedItem.GetAttributes().Get("generation")
		if firstGen != cachedGeneration {
			t.Errorf("expected generation %v, got %v", firstGen, cachedGeneration)
		}

		// search again
		withCache, err := s.Search(ctx, "foo.eu-west-2", "arn:aws:test-type:eu-west-2:foo:test-item", false)
		if err != nil {
			t.Fatal(err)
		}
		withCacheGen, err := withCache[0].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}

		if firstGen != withCacheGen {
			t.Errorf("with cache: expected generation %v, got %v", firstGen, withCacheGen)
		}

		// search ignore cache
		withoutCache, err := s.Search(ctx, "foo.eu-west-2", "arn:aws:test-type:eu-west-2:foo:test-item", true)
		if err != nil {
			t.Fatal(err)
		}
		withoutCacheGen, err := withoutCache[0].GetAttributes().Get("generation")
		if err != nil {
			t.Fatal(err)
		}
		if withoutCacheGen == firstGen {
			t.Errorf("with cache: expected generation %v, got %v", firstGen, withoutCacheGen)
		}
	})
}
