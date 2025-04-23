package discovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
)

func TestEngineAddAdapters(t *testing.T) {
	ec := EngineConfig{}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("Error initializing Engine: %v", err)
	}

	adapter := TestAdapter{}

	if err := e.AddAdapters(&adapter); err != nil {
		t.Fatalf("Error adding adapter: %v", err)
	}

	if x := len(e.sh.Adapters()); x != 1 {
		t.Fatalf("Expected 1 adapters, got %v", x)
	}
}

func TestGet(t *testing.T) {
	adapter := TestAdapter{
		ReturnName: "orange",
		ReturnType: "person",
		ReturnScopes: []string{
			"test",
			"empty",
		},
	}

	e := newStartedEngine(t, "TestGet", nil, nil, &adapter)

	t.Run("Basic test", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		_, _, _, err := e.executeQuerySync(context.Background(), &sdp.Query{
			Type:   "person",
			Scope:  "test",
			Query:  "three",
			Method: sdp.QueryMethod_GET,
		})
		if err != nil {
			t.Fatal(err)
		}

		if x := len(adapter.GetCalls); x != 1 {
			t.Fatalf("Expected 1 get call, got %v", x)
		}

		firstCall := adapter.GetCalls[0]

		if firstCall[0] != "test" || firstCall[1] != "three" {
			t.Fatalf("First get call parameters unexpected: %v", firstCall)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		items, edges, errs, err := e.executeQuerySync(context.Background(), &sdp.Query{
			Type:   "person",
			Scope:  "empty",
			Query:  "three",
			Method: sdp.QueryMethod_GET,
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(errs) == 1 {
			if errs[0].GetErrorType() != sdp.QueryError_NOTFOUND {
				t.Errorf("expected ErrorType to be %v, got %v", sdp.QueryError_NOTFOUND, errs[0].GetErrorType())
			}
			if errs[0].GetErrorString() != "no items found" {
				t.Errorf("expected ErrorString to be '%v', got '%v'", "no items found", errs[0].GetErrorString())
			}
			if errs[0].GetScope() != "empty" {
				t.Errorf("expected Scope to be '%v', got '%v'", "empty", errs[0].GetScope())
			}
			if errs[0].GetSourceName() != "testAdapter-orange" {
				t.Errorf("expected Adapter name to be '%v', got '%v'", "testAdapter-orange", errs[0].GetSourceName())
			}
			if errs[0].GetItemType() != "person" {
				t.Errorf("expected ItemType to be '%v', got '%v'", "person", errs[0].GetItemType())
			}
			if errs[0].GetResponderName() != "TestGet" {
				t.Errorf("expected ResponderName to be '%v', got '%v'", "TestGet", errs[0].GetResponderName())
			}
		} else {
			t.Errorf("expected 1 error, got %v", len(errs))
		}

		if len(items) != 0 {
			t.Errorf("expected 0 items, got %v: %v", len(items), items)
		}
		if len(edges) != 0 {
			t.Errorf("expected 0 edges, got %v: %v", len(edges), edges)
		}
	})

	t.Run("Test caching", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		var list1 []*sdp.Item
		var item2 []*sdp.Item
		var item3 []*sdp.Item
		var err error

		req := sdp.Query{
			Type:   "person",
			Scope:  "test",
			Query:  "Dylan",
			Method: sdp.QueryMethod_GET,
		}

		list1, _, _, err = e.executeQuerySync(context.Background(), &req)
		if err != nil {
			t.Error(err)
		}

		time.Sleep(10 * time.Millisecond)
		item2, _, _, err = e.executeQuerySync(context.Background(), &req)
		if err != nil {
			t.Error(err)
		}

		if list1[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() != item2[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() {
			t.Errorf("Get queries 10ms apart had different timestamps, caching not working. %v != %v", list1[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue(), item2[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue())
		}

		time.Sleep(10 * time.Millisecond)
		e.sh.Purge()

		item3, _, _, err = e.executeQuerySync(context.Background(), &req)
		if err != nil {
			t.Error(err)
		}

		if item2[0].GetMetadata().GetTimestamp().String() == item3[0].GetMetadata().GetTimestamp().String() {
			t.Error("Get queries after purging had the same timestamps, cache not expiring")
		}
	})

	t.Run("Test Get() caching errors", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		req := sdp.Query{
			Type:   "person",
			Scope:  "empty",
			Query:  "query",
			Method: sdp.QueryMethod_GET,
		}

		_, _, errs, err := e.executeQuerySync(context.Background(), &req)
		if err != nil {
			t.Fatal(err)
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		_, _, errs, err = e.executeQuerySync(context.Background(), &req)
		if err != nil {
			t.Fatal(err)
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		if l := len(adapter.GetCalls); l != 1 {
			t.Errorf("Expected 1 Get call due to caching og NOTFOUND errors, got %v", l)
		}
	})

	t.Run("Hidden items", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		adapter.IsHidden = true

		t.Run("Get", func(t *testing.T) {
			item, _, _, err := e.executeQuerySync(context.Background(), &sdp.Query{
				Type:   "person",
				Scope:  "test",
				Query:  "three",
				Method: sdp.QueryMethod_GET,
			})
			if err != nil {
				t.Fatal(err)
			}

			if !item[0].GetMetadata().GetHidden() {
				t.Fatal("Item was not marked as hidden in metadata")
			}
		})

		t.Run("List", func(t *testing.T) {
			items, _, _, err := e.executeQuerySync(context.Background(), &sdp.Query{
				Type:   "person",
				Scope:  "test",
				Method: sdp.QueryMethod_LIST,
			})
			if err != nil {
				t.Fatal(err)
			}

			if !items[0].GetMetadata().GetHidden() {
				t.Fatal("Item was not marked as hidden in metadata")
			}
		})

		t.Run("Search", func(t *testing.T) {
			items, _, _, err := e.executeQuerySync(context.Background(), &sdp.Query{
				Type:   "person",
				Scope:  "test",
				Query:  "three",
				Method: sdp.QueryMethod_SEARCH,
			})
			if err != nil {
				t.Fatal(err)
			}

			if !items[0].GetMetadata().GetHidden() {
				t.Fatal("Item was not marked as hidden in metadata")
			}
		})
	})
}

func TestList(t *testing.T) {
	adapter := TestAdapter{}

	e := newStartedEngine(t, "TestList", nil, nil, &adapter)

	_, _, _, err := e.executeQuerySync(context.Background(), &sdp.Query{
		Type:   "person",
		Scope:  "test",
		Method: sdp.QueryMethod_LIST,
	})
	if err != nil {
		t.Fatal(err)
	}

	if x := len(adapter.ListCalls); x != 1 {
		t.Fatalf("Expected 1 find call, got %v", x)
	}

	firstCall := adapter.ListCalls[0]

	if firstCall[0] != "test" {
		t.Fatalf("First find call parameters unexpected: %v", firstCall)
	}
}

func TestSearch(t *testing.T) {
	adapter := TestAdapter{}

	e := newStartedEngine(t, "TestSearch", nil, nil, &adapter)

	_, _, _, err := e.executeQuerySync(context.Background(), &sdp.Query{
		Type:   "person",
		Scope:  "test",
		Query:  "query",
		Method: sdp.QueryMethod_SEARCH,
	})
	if err != nil {
		t.Fatal(err)
	}

	if x := len(adapter.SearchCalls); x != 1 {
		t.Fatalf("Expected 1 Search call, got %v", x)
	}

	firstCall := adapter.SearchCalls[0]

	if firstCall[0] != "test" || firstCall[1] != "query" {
		t.Fatalf("First Search call parameters unexpected: %v", firstCall)
	}
}

func TestListSearchCaching(t *testing.T) {
	adapter := TestAdapter{
		ReturnScopes: []string{
			"test",
			"empty",
			"error",
		},
	}

	e := newStartedEngine(t, "TestListSearchCaching", nil, nil, &adapter)

	t.Run("caching with successful list", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		var list1 []*sdp.Item
		var list2 []*sdp.Item
		var list3 []*sdp.Item
		var err error
		q := sdp.Query{
			Type:   "person",
			Scope:  "test",
			Method: sdp.QueryMethod_LIST,
		}

		list1, _, _, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		time.Sleep(10 * time.Millisecond)

		list2, _, _, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Fatal(err)
		}

		if list1[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() != list2[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() {
			t.Errorf("List queries had different generations, caching not working. %v != %v", list1[0].GetAttributes().GetAttrStruct().GetFields()["generation"], list2[0].GetAttributes().GetAttrStruct().GetFields()["generation"])
		}

		time.Sleep(10 * time.Millisecond)
		e.sh.Purge()

		list3, _, _, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Fatal(err)
		}

		if list2[0].GetAttributes().GetAttrStruct().GetFields()["generation"] == list3[0].GetAttributes().GetAttrStruct().GetFields()["generation"] {
			t.Errorf("List queries after purging had the same generation, caching not working. %v == %v", list2[0].GetAttributes().GetAttrStruct().GetFields()["generation"], list3[0].GetAttributes().GetAttrStruct().GetFields()["generation"])
		}
	})

	t.Run("empty list", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		var err error
		q := sdp.Query{
			Type:   "person",
			Scope:  "empty",
			Method: sdp.QueryMethod_LIST,
		}

		_, _, errs, err := e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Fatal(err)
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		time.Sleep(10 * time.Millisecond)

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Fatal(err)
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		if l := len(adapter.ListCalls); l != 1 {
			t.Errorf("Expected only 1 list call, got %v, cache not working: %v", l, adapter.ListCalls)
		}

		time.Sleep(200 * time.Millisecond)

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Fatal(err)
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		if l := len(adapter.ListCalls); l != 2 {
			t.Errorf("Expected 2 list calls, got %v, cache not clearing: %v", l, adapter.ListCalls)
		}
	})

	t.Run("caching with successful search", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		var list1 []*sdp.Item
		var list2 []*sdp.Item
		var list3 []*sdp.Item
		var err error
		q := sdp.Query{
			Type:   "person",
			Scope:  "test",
			Query:  "query",
			Method: sdp.QueryMethod_SEARCH,
		}

		list1, _, _, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		time.Sleep(10 * time.Millisecond)

		list2, _, _, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		if list1[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() != list2[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() {
			t.Errorf("List queries had different generations, caching not working. %v != %v", list1[0].GetAttributes().GetAttrStruct().GetFields()["generation"], list2[0].GetAttributes().GetAttrStruct().GetFields()["generation"])
		}

		time.Sleep(200 * time.Millisecond)

		list3, _, _, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		if list2[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() == list3[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() {
			t.Errorf("List queries 200ms apart had the same generations, caching not working. %v == %v", list2[0].GetAttributes().GetAttrStruct().GetFields()["generation"], list3[0].GetAttributes().GetAttrStruct().GetFields()["generation"])
		}
	})

	t.Run("empty search", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		var err error
		q := sdp.Query{
			Type:   "person",
			Scope:  "empty",
			Query:  "query",
			Method: sdp.QueryMethod_SEARCH,
		}

		_, _, errs, err := e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		time.Sleep(10 * time.Millisecond)

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		time.Sleep(200 * time.Millisecond)

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		if l := len(adapter.SearchCalls); l != 2 {
			t.Errorf("Expected 2 find calls, got %v, cache not clearing", l)
		}
	})

	t.Run("non-caching of OTHER errors", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		q := sdp.Query{
			Type:   "person",
			Scope:  "error",
			Query:  "query",
			Method: sdp.QueryMethod_GET,
		}

		_, _, errs, err := e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}
		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		if l := len(adapter.GetCalls); l != 2 {
			t.Errorf("Expected 2 get calls, got %v, OTHER errors should not be cached", l)
		}
	})

	t.Run("non-caching when ignoreCache is specified", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		q := sdp.Query{
			Type:   "person",
			Scope:  "error",
			Query:  "query",
			Method: sdp.QueryMethod_GET,
		}

		_, _, errs, err := e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		q.Method = sdp.QueryMethod_LIST

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		q.Method = sdp.QueryMethod_SEARCH

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		_, _, errs, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}
		if len(errs) != 1 {
			t.Fatalf("Expected 1 error, got %v", len(errs))
		}

		if l := len(adapter.GetCalls); l != 2 {
			t.Errorf("Expected 2 get calls, got %v", l)
		}

		if l := len(adapter.ListCalls); l != 2 {
			t.Errorf("Expected 2 List calls, got %v", l)
		}

		if l := len(adapter.SearchCalls); l != 2 {
			t.Errorf("Expected 2 Search calls, got %v", l)
		}
	})
}

func TestSearchGetCaching(t *testing.T) {
	// We want to be sure that if an item has been found via a search and
	// cached, the cache will be hit if a Get is run for that particular item

	adapter := TestAdapter{
		ReturnScopes: []string{
			"test",
		},
	}

	e := newStartedEngine(t, "TestSearchGetCaching", nil, nil, &adapter)

	t.Run("caching with successful search", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
		})

		var searchResult []*sdp.Item
		var searchErrors []*sdp.QueryError
		var getResult []*sdp.Item
		var getErrors []*sdp.QueryError
		var err error
		q := sdp.Query{
			Type:   "person",
			Scope:  "test",
			Query:  "Dylan",
			Method: sdp.QueryMethod_SEARCH,
		}

		t.Logf("Searching for %v", q.GetQuery())
		searchResult, _, searchErrors, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		if len(searchErrors) != 0 {
			for _, err := range searchErrors {
				t.Error(err)
			}
		}

		if len(searchResult) == 0 {
			t.Fatal("Got no results")
		}

		if len(searchResult) > 1 {
			t.Fatalf("Got too many results: %v", searchResult)
		}

		time.Sleep(10 * time.Millisecond)

		// Do a get query for that same item
		q.Method = sdp.QueryMethod_GET
		q.Query = searchResult[0].UniqueAttributeValue()

		t.Logf("Getting %v from cache", q.GetQuery())
		getResult, _, getErrors, err = e.executeQuerySync(context.Background(), &q)
		if err != nil {
			t.Error(err)
		}

		if len(getErrors) != 0 {
			for _, err := range getErrors {
				t.Error(err)
			}
		}

		if len(getResult) == 0 {
			t.Error("No result from GET")
		}

		if searchResult[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() != getResult[0].GetAttributes().GetAttrStruct().GetFields()["generation"].GetNumberValue() {
			t.Errorf("Search and Get queries had different generations, caching not working. %v != %v", searchResult[0].GetAttributes().GetAttrStruct().GetFields()["generation"], getResult[0].GetAttributes().GetAttrStruct().GetFields()["generation"])
		}
	})
}

func TestNewQueryResultStream(t *testing.T) {
	items := make(chan *sdp.Item, 10)
	errs := make(chan error, 10)

	itemHandler := func(item *sdp.Item) {
		time.Sleep(10 * time.Millisecond)
		items <- item
	}

	errHandler := func(err error) {
		time.Sleep(10 * time.Millisecond)
		errs <- err
	}

	stream := NewQueryResultStream(itemHandler, errHandler)

	// Test Initialization
	if stream == nil {
		t.Fatal("Expected stream to be initialized, got nil")
	}
	if stream.itemHandler == nil || stream.errHandler == nil {
		t.Fatal("Expected handlers to be set")
	}

	// Test SendItem
	testItem := &sdp.Item{}
	stream.SendItem(testItem)

	// Due to the fact that the handlers are executed in a goroutine it
	// essentially gives us a buffered channel with a buffer depth of 1 since
	// the item can be pulled off the internal items channel immediately then
	// wait on the handler in parallel. That's what allows this test to work
	// without extra synchronization
	if x := <-items; x != testItem {
		t.Fatalf("Expected item to be %v, got %v", testItem, x)
	}

	// Test SendError
	testErr := errors.New("test error")
	stream.SendError(testErr)

	if x := <-errs; x.Error() != testErr.Error() {
		t.Fatalf("Expected error to be %v, got %v", testErr, x)
	}
}
