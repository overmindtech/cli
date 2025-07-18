package sdpws

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/proto"
)

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestServer is a test server for the websocket client. Note that this can only
// handle a single connection at a time.
type testServer struct {
	url string

	conn   *websocket.Conn
	connMu sync.Mutex

	requests   []*sdp.GatewayRequest
	requestsMu sync.Mutex
}

func newTestServer(_ context.Context, t *testing.T) (*testServer, func()) {
	ts := &testServer{
		requests: make([]*sdp.GatewayRequest, 0),
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = c.Close(websocket.StatusNormalClosure, "")
		}()

		ts.connMu.Lock()
		ts.conn = c
		ts.connMu.Unlock()

		// ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
		// defer cancel()

		for {
			msg := &sdp.GatewayRequest{}

			typ, reader, err := c.Reader(r.Context())
			if err != nil {
				c.Close(websocket.StatusAbnormalClosure, fmt.Sprintf("failed to initialise websocket reader: %v", err))
				return
			}
			if typ != websocket.MessageBinary {
				c.Close(websocket.StatusAbnormalClosure, fmt.Sprintf("expected binary message for protobuf but got: %v", typ))
				t.Fatalf("expected binary message for protobuf but got: %v", typ)
				return
			}

			b := new(bytes.Buffer)
			_, err = b.ReadFrom(reader)
			if err != nil {
				c.Close(websocket.StatusAbnormalClosure, fmt.Sprintf("failed to read from websocket: %v", err))
				t.Fatalf("failed to read from websocket: %v", err)
				return
			}

			err = proto.Unmarshal(b.Bytes(), msg)
			if err != nil {
				c.Close(websocket.StatusAbnormalClosure, fmt.Sprintf("error un marshaling message: %v", err))
				t.Fatalf("error un marshaling message: %v", err)
				return
			}

			ts.requestsMu.Lock()
			ts.requests = append(ts.requests, msg)
			ts.requestsMu.Unlock()
		}
	})

	s := httptest.NewServer(serveMux)
	ts.url = s.URL

	return ts, func() {
		s.Close()
	}
}

func (ts *testServer) inject(ctx context.Context, msg *sdp.GatewayResponse) {
	ts.connMu.Lock()
	c := ts.conn
	ts.connMu.Unlock()

	buf, err := proto.Marshal(msg)
	if err != nil {
		c.Close(websocket.StatusAbnormalClosure, fmt.Sprintf("error marshaling message: %v", err))
		return
	}

	err = c.Write(ctx, websocket.MessageBinary, buf)
	if err != nil {
		c.Close(websocket.StatusAbnormalClosure, fmt.Sprintf("error writing message: %v", err))
		return
	}
}

func TestClient(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("Query", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		ctx := context.Background()

		ts, closeFn := newTestServer(ctx, t)
		defer closeFn()

		c, err := Dial(ctx, ts.url, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = c.Close(ctx)
		}()

		u := uuid.New()

		q := &sdp.Query{
			UUID:               u[:],
			Type:               "",
			Method:             0,
			Query:              "",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
			Scope:              "",
			IgnoreCache:        false,
		}

		go func() {
			time.Sleep(100 * time.Millisecond)
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_QueryStatus{
					QueryStatus: &sdp.QueryStatus{
						UUID:   u[:],
						Status: sdp.QueryStatus_FINISHED,
					},
				},
			})
		}()

		// this will block until the above goroutine has injected the response
		_, err = c.Query(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		err = c.Wait(ctx, uuid.UUIDs{u})
		if err != nil {
			t.Fatal(err)
		}

		ts.requestsMu.Lock()
		defer ts.requestsMu.Unlock()

		if len(ts.requests) != 1 {
			t.Fatalf("expected 1 request, got %v: %v", len(ts.requests), ts.requests)
		}

		recvQ, ok := ts.requests[0].GetRequestType().(*sdp.GatewayRequest_Query)
		if !ok || uuid.UUID(recvQ.Query.GetUUID()) != u {
			t.Fatalf("expected query, got %v", ts.requests[0])
		}
	})

	t.Run("QueryNotFound", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		ctx := context.Background()

		ts, closeFn := newTestServer(ctx, t)
		defer closeFn()

		c, err := Dial(ctx, ts.url, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = c.Close(ctx)
		}()

		u := uuid.New()

		q := &sdp.Query{
			UUID:               u[:],
			Type:               "",
			Method:             0,
			Query:              "",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
			Scope:              "",
			IgnoreCache:        false,
		}

		go func() {
			time.Sleep(100 * time.Millisecond)
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_QueryStatus{
					QueryStatus: &sdp.QueryStatus{
						UUID:   u[:],
						Status: sdp.QueryStatus_STARTED,
					},
				},
			})
			time.Sleep(100 * time.Millisecond)
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_QueryError{
					QueryError: &sdp.QueryError{
						UUID:          u[:],
						ErrorType:     sdp.QueryError_NOTFOUND,
						ErrorString:   "not found",
						Scope:         "scope",
						SourceName:    "src name",
						ItemType:      "item type",
						ResponderName: "responder name",
					},
				},
			})
			time.Sleep(100 * time.Millisecond)
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_QueryStatus{
					QueryStatus: &sdp.QueryStatus{
						UUID:   u[:],
						Status: sdp.QueryStatus_ERRORED,
					},
				},
			})
		}()

		// this will block until the above goroutine has injected the response
		_, err = c.Query(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		err = c.Wait(ctx, uuid.UUIDs{u})
		if err != nil {
			t.Fatal(err)
		}

		ts.requestsMu.Lock()
		defer ts.requestsMu.Unlock()

		if len(ts.requests) != 1 {
			t.Fatalf("expected 1 request, got %v: %v", len(ts.requests), ts.requests)
		}

		recvQ, ok := ts.requests[0].GetRequestType().(*sdp.GatewayRequest_Query)
		if !ok || uuid.UUID(recvQ.Query.GetUUID()) != u {
			t.Fatalf("expected query, got %v", ts.requests[0])
		}
	})

	t.Run("StoreSnapshot", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		ctx := context.Background()

		ts, closeFn := newTestServer(ctx, t)
		defer closeFn()

		c, err := Dial(ctx, ts.url, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = c.Close(ctx)
		}()

		u := uuid.New()

		go func() {
			time.Sleep(100 * time.Millisecond)
			ts.requestsMu.Lock()
			msgID := ts.requests[0].GetStoreSnapshot().GetMsgID()
			ts.requestsMu.Unlock()

			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_SnapshotStoreResult{
					SnapshotStoreResult: &sdp.SnapshotStoreResult{
						Success:      true,
						ErrorMessage: "",
						MsgID:        msgID,
						SnapshotID:   u[:],
					},
				},
			})
		}()

		// this will block until the above goroutine has injected the response
		snapu, err := c.StoreSnapshot(ctx, "name", "description")
		if err != nil {
			t.Fatal(err)
		}
		if snapu != u {
			t.Errorf("expected snapshot id %v, got %v", u, snapu)
		}

		ts.requestsMu.Lock()
		defer ts.requestsMu.Unlock()

		if len(ts.requests) != 1 {
			t.Fatalf("expected 1 request, got %v: %v", len(ts.requests), ts.requests)
		}
	})

	t.Run("StoreBookmark", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		ctx := context.Background()

		ts, closeFn := newTestServer(ctx, t)
		defer closeFn()

		c, err := Dial(ctx, ts.url, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = c.Close(ctx)
		}()

		u := uuid.New()

		go func() {
			time.Sleep(100 * time.Millisecond)
			ts.requestsMu.Lock()
			msgID := ts.requests[0].GetStoreBookmark().GetMsgID()
			ts.requestsMu.Unlock()

			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_BookmarkStoreResult{
					BookmarkStoreResult: &sdp.BookmarkStoreResult{
						Success:      true,
						ErrorMessage: "",
						MsgID:        msgID,
						BookmarkID:   u[:],
					},
				},
			})
		}()

		// this will block until the above goroutine has injected the response
		snapu, err := c.StoreBookmark(ctx, "name", "description", true)
		if err != nil {
			t.Fatal(err)
		}
		if snapu != u {
			t.Errorf("expected bookmark id %v, got %v", u, snapu)
		}

		ts.requestsMu.Lock()
		defer ts.requestsMu.Unlock()

		if len(ts.requests) != 1 {
			t.Fatalf("expected 1 request, got %v: %v", len(ts.requests), ts.requests)
		}
	})

	t.Run("ConcurrentQueries", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		ctx := context.Background()

		ts, closeFn := newTestServer(ctx, t)
		defer closeFn()

		c, err := Dial(ctx, ts.url, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = c.Close(ctx)
		}()

		// Create multiple queries with different UUIDs
		numQueries := 5
		queries := make([]*sdp.Query, numQueries)
		expectedItems := make(map[string]*sdp.Item)

		for i := range numQueries {
			u := uuid.New()
			queries[i] = &sdp.Query{
				UUID:               u[:],
				Type:               "test",
				Method:             sdp.QueryMethod_GET,
				Query:              fmt.Sprintf("query-%d", i),
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
				Scope:              "test",
				IgnoreCache:        false,
			}

			// Create expected items that should be returned for each query
			expectedItems[u.String()] = &sdp.Item{
				Type:            "test",
				UniqueAttribute: fmt.Sprintf("item-%d", i),
				Scope:           "test",
				Metadata: &sdp.Metadata{
					SourceQuery: queries[i],
				},
			}
		}

		// Inject responses in a different order than queries to test proper routing
		go func() {
			time.Sleep(50 * time.Millisecond)

			// Send responses in reverse order to test UUID-based routing
			for i := numQueries - 1; i >= 0; i-- {
				u := uuid.UUID(queries[i].GetUUID())

				// Send an item response first
				ts.inject(ctx, &sdp.GatewayResponse{
					ResponseType: &sdp.GatewayResponse_NewItem{
						NewItem: expectedItems[u.String()],
					},
				})

				// Then send the completion status
				ts.inject(ctx, &sdp.GatewayResponse{
					ResponseType: &sdp.GatewayResponse_QueryStatus{
						QueryStatus: &sdp.QueryStatus{
							UUID:   u[:],
							Status: sdp.QueryStatus_FINISHED,
						},
					},
				})

				// Add a small delay between responses to make race conditions more likely
				time.Sleep(10 * time.Millisecond)
			}
		}()

		// Execute all queries concurrently
		type queryResult struct {
			index int
			items []*sdp.Item
			err   error
		}

		results := make([]queryResult, numQueries)
		var wg sync.WaitGroup

		for i := range numQueries {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				items, err := c.Query(ctx, queries[index])
				results[index] = queryResult{
					index: index,
					items: items,
					err:   err,
				}
			}(i)
		}

		wg.Wait()

		// Verify that each query got the correct response
		for i, result := range results {
			if result.err != nil {
				t.Errorf("Query %d failed: %v", i, result.err)
				continue
			}

			if len(result.items) != 1 {
				t.Errorf("Query %d: expected 1 item, got %d", i, len(result.items))
				continue
			}

			receivedItem := result.items[0]
			expectedUniqueAttr := fmt.Sprintf("item-%d", i)

			if receivedItem.GetUniqueAttribute() != expectedUniqueAttr {
				t.Errorf("Query %d: expected item with unique attribute %s, got %s",
					i, expectedUniqueAttr, receivedItem.GetUniqueAttribute())
			}

			// Verify the item's metadata contains the correct source query
			if receivedItem.GetMetadata() == nil || receivedItem.GetMetadata().GetSourceQuery() == nil {
				t.Errorf("Query %d: item missing metadata or source query", i)
				continue
			}

			sourceQueryUUID := uuid.UUID(receivedItem.GetMetadata().GetSourceQuery().GetUUID())
			expectedUUID := uuid.UUID(queries[i].GetUUID())

			if sourceQueryUUID != expectedUUID {
				t.Errorf("Query %d: expected source query UUID %s, got %s",
					i, expectedUUID, sourceQueryUUID)
			}
		}

		// Verify that the server received all queries
		ts.requestsMu.Lock()
		defer ts.requestsMu.Unlock()

		if len(ts.requests) != numQueries {
			t.Fatalf("expected %d requests, got %d", numQueries, len(ts.requests))
		}
	})

	t.Run("ResponseMixupPrevention", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		ctx := context.Background()

		ts, closeFn := newTestServer(ctx, t)
		defer closeFn()

		c, err := Dial(ctx, ts.url, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = c.Close(ctx)
		}()

		// Create two queries with different UUIDs
		query1UUID := uuid.New()
		query2UUID := uuid.New()

		query1 := &sdp.Query{
			UUID:               query1UUID[:],
			Type:               "test",
			Method:             sdp.QueryMethod_GET,
			Query:              "query-1",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
			Scope:              "test",
			IgnoreCache:        false,
		}

		query2 := &sdp.Query{
			UUID:               query2UUID[:],
			Type:               "test",
			Method:             sdp.QueryMethod_GET,
			Query:              "query-2",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
			Scope:              "test",
			IgnoreCache:        false,
		}

		// Items that should be returned for each query
		item1 := &sdp.Item{
			Type:            "test",
			UniqueAttribute: "item-for-query-1",
			Scope:           "test",
			Metadata: &sdp.Metadata{
				SourceQuery: query1,
			},
		}

		item2 := &sdp.Item{
			Type:            "test",
			UniqueAttribute: "item-for-query-2",
			Scope:           "test",
			Metadata: &sdp.Metadata{
				SourceQuery: query2,
			},
		}

		// Inject responses in a way that could cause mixup if UUIDs aren't handled correctly
		go func() {
			time.Sleep(50 * time.Millisecond)

			// Send responses for query2 first, then query1
			// If the client doesn't properly route by UUID, responses could get mixed up

			// Send multiple items for query2
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_NewItem{
					NewItem: item2,
				},
			})

			// Send an item for query1
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_NewItem{
					NewItem: item1,
				},
			})

			// Send another item for query2 to test multiple items per query
			item2_duplicate := &sdp.Item{
				Type:            "test",
				UniqueAttribute: "item-for-query-2-duplicate",
				Scope:           "test",
				Metadata: &sdp.Metadata{
					SourceQuery: query2,
				},
			}
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_NewItem{
					NewItem: item2_duplicate,
				},
			})

			// Complete query1 first (even though we sent its response second)
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_QueryStatus{
					QueryStatus: &sdp.QueryStatus{
						UUID:   query1UUID[:],
						Status: sdp.QueryStatus_FINISHED,
					},
				},
			})

			// Complete query2 after query1
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_QueryStatus{
					QueryStatus: &sdp.QueryStatus{
						UUID:   query2UUID[:],
						Status: sdp.QueryStatus_FINISHED,
					},
				},
			})
		}()

		// Execute both queries concurrently
		type result struct {
			items []*sdp.Item
			err   error
		}

		var wg sync.WaitGroup
		results := make([]result, 2)

		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := c.Query(ctx, query1)
			results[0] = result{items: items, err: err}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := c.Query(ctx, query2)
			results[1] = result{items: items, err: err}
		}()

		wg.Wait()

		// Verify query1 got the correct response
		if results[0].err != nil {
			t.Errorf("Query 1 failed: %v", results[0].err)
		} else {
			if len(results[0].items) != 1 {
				t.Errorf("Query 1: expected 1 item, got %d", len(results[0].items))
			} else if results[0].items[0].GetUniqueAttribute() != "item-for-query-1" {
				t.Errorf("Query 1: got wrong item: %s", results[0].items[0].GetUniqueAttribute())
			}
		}

		// Verify query2 got the correct responses
		if results[1].err != nil {
			t.Errorf("Query 2 failed: %v", results[1].err)
		} else {
			if len(results[1].items) != 2 {
				t.Errorf("Query 2: expected 2 items, got %d", len(results[1].items))
			} else {
				// Check that both items are for query2
				for i, item := range results[1].items {
					if !contains([]string{"item-for-query-2", "item-for-query-2-duplicate"}, item.GetUniqueAttribute()) {
						t.Errorf("Query 2, item %d: got wrong item: %s", i, item.GetUniqueAttribute())
					}
				}
			}
		}
	})

	t.Run("UUIDRoutingValidation", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		ctx := context.Background()

		ts, closeFn := newTestServer(ctx, t)
		defer closeFn()

		c, err := Dial(ctx, ts.url, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = c.Close(ctx)
		}()

		// This test validates that responses are properly routed by UUID
		// If the client were reading responses in order (FIFO) instead of by UUID,
		// this test would fail because we send responses out of order

		queryA_UUID := uuid.New()
		queryB_UUID := uuid.New()

		queryA := &sdp.Query{
			UUID:               queryA_UUID[:],
			Type:               "test",
			Method:             sdp.QueryMethod_GET,
			Query:              "query-A",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
			Scope:              "test",
			IgnoreCache:        false,
		}

		queryB := &sdp.Query{
			UUID:               queryB_UUID[:],
			Type:               "test",
			Method:             sdp.QueryMethod_GET,
			Query:              "query-B",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{},
			Scope:              "test",
			IgnoreCache:        false,
		}

		// Items that should be returned for each query
		itemA := &sdp.Item{
			Type:            "test",
			UniqueAttribute: "item-A",
			Scope:           "test",
			Metadata: &sdp.Metadata{
				SourceQuery: queryA,
			},
		}

		itemB := &sdp.Item{
			Type:            "test",
			UniqueAttribute: "item-B",
			Scope:           "test",
			Metadata: &sdp.Metadata{
				SourceQuery: queryB,
			},
		}

		// Inject responses deliberately out of order
		go func() {
			time.Sleep(50 * time.Millisecond)

			// Send itemB first (for queryB), then itemA (for queryA)
			// If the client doesn't route by UUID, queryA might get itemB
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_NewItem{
					NewItem: itemB,
				},
			})

			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_NewItem{
					NewItem: itemA,
				},
			})

			// Complete queryA first (even though itemA was sent second)
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_QueryStatus{
					QueryStatus: &sdp.QueryStatus{
						UUID:   queryA_UUID[:],
						Status: sdp.QueryStatus_FINISHED,
					},
				},
			})

			// Complete queryB second
			ts.inject(ctx, &sdp.GatewayResponse{
				ResponseType: &sdp.GatewayResponse_QueryStatus{
					QueryStatus: &sdp.QueryStatus{
						UUID:   queryB_UUID[:],
						Status: sdp.QueryStatus_FINISHED,
					},
				},
			})
		}()

		// Execute queryA - it should get itemA despite itemB being sent first
		var wg sync.WaitGroup
		type result struct {
			items []*sdp.Item
			err   error
		}

		resultsA := make([]result, 1)
		resultsB := make([]result, 1)

		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := c.Query(ctx, queryA)
			resultsA[0] = result{items: items, err: err}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			items, err := c.Query(ctx, queryB)
			resultsB[0] = result{items: items, err: err}
		}()

		wg.Wait()

		// Verify queryA got the correct item
		if resultsA[0].err != nil {
			t.Fatalf("Query A failed: %v", resultsA[0].err)
		}

		if len(resultsA[0].items) != 1 {
			t.Fatalf("Query A: expected 1 item, got %d", len(resultsA[0].items))
		}

		if resultsA[0].items[0].GetUniqueAttribute() != "item-A" {
			t.Errorf("Query A got wrong item: expected 'item-A', got '%s'", resultsA[0].items[0].GetUniqueAttribute())
		}

		// Verify queryB got the correct item
		if resultsB[0].err != nil {
			t.Fatalf("Query B failed: %v", resultsB[0].err)
		}

		if len(resultsB[0].items) != 1 {
			t.Fatalf("Query B: expected 1 item, got %d", len(resultsB[0].items))
		}

		if resultsB[0].items[0].GetUniqueAttribute() != "item-B" {
			t.Errorf("Query B got wrong item: expected 'item-B', got '%s'", resultsB[0].items[0].GetUniqueAttribute())
		}
	})
}
