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
}
