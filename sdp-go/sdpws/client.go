package sdpws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/protobuf/proto"
)

// Client is the main driver for all interactions with a SDP/Gateway websocket.
//
// Internally it holds a map of all active requests, which are identified by a
// UUID, to multiplex incoming responses to the correct caller. Note that the
// request methods block until the response is received, so to send multiple
// requests in parallel, call requestor methods in goroutines, e.g. using a conc
// Pool:
//
// ```
//
//	pool := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError()
//	pool.Go(func() error {
//	     items, err := client.Query(ctx, q)
//	     if err != nil {
//	         return err
//	     }
//	     // do something with items
//	}
//	// ...
//	pool.Wait()
//
// ```
//
// Alternatively, pass in a GatewayMessageHandler to receive all messages as
// they come in and send messages directly using `Send()` and then `Wait()` for
// all request IDs.
type Client struct {
	conn *websocket.Conn

	handler GatewayMessageHandler

	requestMap   map[uuid.UUID]chan *sdp.GatewayResponse
	requestMapMu sync.RWMutex

	finishedRequestMap     map[uuid.UUID]bool
	finishedRequestMapCond *sync.Cond
	finishedRequestMapMu   sync.Mutex

	err   error
	errMu sync.Mutex

	closed     bool
	closedCond *sync.Cond
	closedMu   sync.Mutex
}

// Dial connects to the given URL and returns a new Client. Pass nil as handler
// if you do not need per-message callbacks.
//
// To stop the client, cancel the provided context:
//
// ```
// ctx, cancel := context.WithCancel(context.Background())
// defer cancel()
// client, err := sdpws.Dial(ctx, gatewayUrl, NewAuthenticatedClient(ctx, otelhttp.DefaultClient), nil)
// ```
func Dial(ctx context.Context, u string, httpClient *http.Client, handler GatewayMessageHandler) (*Client, error) {
	return dialImpl(ctx, u, httpClient, handler, true)
}

// DialBatch connects to the given URL and returns a new Client. Pass nil as
// handler if you do not need per-message callbacks. This method is intended for
// batch processing and sets up opentelemetry propagation. Otherwise this
// equivalent to `Dial()`
func DialBatch(ctx context.Context, u string, httpClient *http.Client, handler GatewayMessageHandler) (*Client, error) {
	return dialImpl(ctx, u, httpClient, handler, false)
}

func dialImpl(ctx context.Context, u string, httpClient *http.Client, handler GatewayMessageHandler, interactive bool) (*Client, error) {
	if httpClient == nil {
		httpClient = otelhttp.DefaultClient
	}
	options := &websocket.DialOptions{
		HTTPClient: httpClient,
	}
	if !interactive {
		options.HTTPHeader = http.Header{
			"X-overmind-interactive": []string{"false"},
		}
	}

	//nolint: bodyclose // github.com/coder/websocket reads the body internally
	conn, _, err := websocket.Dial(ctx, u, options)
	if err != nil {
		return nil, err
	}

	// the default, 32kB is too small for cert bundles and rds-db-cluster-parameter-groups
	conn.SetReadLimit(2 * 1024 * 1024)

	c := &Client{
		conn:               conn,
		handler:            handler,
		requestMap:         make(map[uuid.UUID]chan *sdp.GatewayResponse),
		finishedRequestMap: make(map[uuid.UUID]bool),
	}
	c.closedCond = sync.NewCond(&c.closedMu)
	c.finishedRequestMapCond = sync.NewCond(&c.finishedRequestMapMu)

	go c.receive(ctx)

	return c, nil
}

func (c *Client) receive(ctx context.Context) {
	defer tracing.LogRecoverToReturn(ctx, "sdpws.Client.receive")
	for {
		msg := &sdp.GatewayResponse{}

		typ, r, err := c.conn.Reader(ctx)
		if err != nil {
			c.abort(ctx, fmt.Errorf("failed to initialise websocket reader: %w", err))
			return
		}
		if typ != websocket.MessageBinary {
			c.conn.Close(websocket.StatusUnsupportedData, "expected binary message")
			c.abort(ctx, fmt.Errorf("expected binary message for protobuf but got: %v", typ))
			return
		}

		b := new(bytes.Buffer)
		_, err = b.ReadFrom(r)
		if err != nil {
			c.abort(ctx, fmt.Errorf("failed to read from websocket: %w", err))
			return
		}

		err = proto.Unmarshal(b.Bytes(), msg)
		if err != nil {
			c.abort(ctx, fmt.Errorf("error unmarshalling message: %w", err))
			return
		}

		switch msg.GetResponseType().(type) {
		case *sdp.GatewayResponse_NewItem:
			item := msg.GetNewItem()
			if c.handler != nil {
				c.handler.NewItem(ctx, item)
			}
			u, err := uuid.FromBytes(item.GetMetadata().GetSourceQuery().GetUUID())
			if err == nil {
				c.postRequestChan(u, msg)
			}

		case *sdp.GatewayResponse_NewEdge:
			edge := msg.GetNewEdge()
			if c.handler != nil {
				c.handler.NewEdge(ctx, edge)
			}
			// TODO: edges are not attached to a specific query, so we can't send them to a request channel
			//       maybe that's not a problem anyways?
			// c, ok := c.getRequestChan(uuid.UUID(edge.Metadata.SourceQuery.UUID))
			// if ok {
			// 	c <- msg
			// }

		case *sdp.GatewayResponse_Status:
			status := msg.GetStatus()
			if c.handler != nil {
				c.handler.Status(ctx, status)
			}

		case *sdp.GatewayResponse_QueryError:
			qe := msg.GetQueryError()
			if c.handler != nil {
				c.handler.QueryError(ctx, qe)
			}
			u, err := uuid.FromBytes(qe.GetUUID())
			if err == nil {
				c.postRequestChan(u, msg)
			}

		case *sdp.GatewayResponse_DeleteItem:
			item := msg.GetDeleteItem()
			if c.handler != nil {
				c.handler.DeleteItem(ctx, item)
			}

		case *sdp.GatewayResponse_DeleteEdge:
			edge := msg.GetDeleteEdge()
			if c.handler != nil {
				c.handler.DeleteEdge(ctx, edge)
			}

		case *sdp.GatewayResponse_UpdateItem:
			item := msg.GetUpdateItem()
			if c.handler != nil {
				c.handler.UpdateItem(ctx, item)
			}

		case *sdp.GatewayResponse_SnapshotStoreResult:
			result := msg.GetSnapshotStoreResult()
			if c.handler != nil {
				c.handler.SnapshotStoreResult(ctx, result)
			}
			u, err := uuid.FromBytes(result.GetMsgID())
			if err == nil {
				c.postRequestChan(u, msg)
			}

		case *sdp.GatewayResponse_SnapshotLoadResult:
			result := msg.GetSnapshotLoadResult()
			if c.handler != nil {
				c.handler.SnapshotLoadResult(ctx, result)
			}
			u, err := uuid.FromBytes(result.GetMsgID())
			if err == nil {
				c.postRequestChan(u, msg)
			}

		case *sdp.GatewayResponse_BookmarkStoreResult:
			result := msg.GetBookmarkStoreResult()
			if c.handler != nil {
				c.handler.BookmarkStoreResult(ctx, result)
			}
			u, err := uuid.FromBytes(result.GetMsgID())
			if err == nil {
				c.postRequestChan(u, msg)
			}

		case *sdp.GatewayResponse_BookmarkLoadResult:
			result := msg.GetBookmarkLoadResult()
			if c.handler != nil {
				c.handler.BookmarkLoadResult(ctx, result)
			}
			u, err := uuid.FromBytes(result.GetMsgID())
			if err == nil {
				c.postRequestChan(u, msg)
			}

		case *sdp.GatewayResponse_QueryStatus:
			qs := msg.GetQueryStatus()
			if c.handler != nil {
				c.handler.QueryStatus(ctx, qs)
			}
			u, err := uuid.FromBytes(qs.GetUUID())
			if err == nil {
				c.postRequestChan(u, msg)
			}

			switch qs.GetStatus() { //nolint: exhaustive // ignore sdp.QueryStatus_UNSPECIFIED, sdp.QueryStatus_STARTED
			case sdp.QueryStatus_FINISHED, sdp.QueryStatus_CANCELLED, sdp.QueryStatus_ERRORED:
				c.finishRequestChan(u)
			}

		case *sdp.GatewayResponse_ChatResponse:
			chatResponse := msg.GetChatResponse()
			if c.handler != nil {
				c.handler.ChatResponse(ctx, chatResponse)
			}
			c.postRequestChan(uuid.Nil, msg)

		case *sdp.GatewayResponse_ToolStart:
			toolStart := msg.GetToolStart()
			if c.handler != nil {
				c.handler.ToolStart(ctx, toolStart)
			}
			c.postRequestChan(uuid.Nil, msg)

		case *sdp.GatewayResponse_ToolFinish:
			toolFinish := msg.GetToolFinish()
			if c.handler != nil {
				c.handler.ToolFinish(ctx, toolFinish)
			}
			c.postRequestChan(uuid.Nil, msg)

		default:
			log.WithContext(ctx).WithField("response", msg).WithField("responseType", fmt.Sprintf("%T", msg.GetResponseType())).Warn("unexpected response")
		}
	}
}

func (c *Client) send(ctx context.Context, msg *sdp.GatewayRequest) error {
	buf, err := proto.Marshal(msg)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithField("request", msg).Trace("error marshaling request")
		c.abort(ctx, err)
		return err
	}

	err = c.conn.Write(ctx, websocket.MessageBinary, buf)
	if err != nil {
		log.WithContext(ctx).WithError(err).WithField("request", msg).Trace("error writing request to websocket")
		c.abort(ctx, err)
		return err
	}
	return nil
}

// Wait blocks until all specified requests have been finished. Waiting on a
// closed client returns immediately with no error.
func (c *Client) Wait(ctx context.Context, reqIDs uuid.UUIDs) error {
	for {
		if c.Closed() {
			return nil
		}

		// check for context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// wrap this in a function so defers can be called (otherwise the lock is held for all loop iterations)
		finished := func() bool {
			c.finishedRequestMapMu.Lock()
			defer c.finishedRequestMapMu.Unlock()

			// remove all finished requests from the list of requests to wait for
			reqIDs = slices.DeleteFunc(reqIDs, func(reqID uuid.UUID) bool {
				_, ok := c.finishedRequestMap[reqID]
				return ok
			})
			if len(reqIDs) == 0 {
				return true
			}

			c.finishedRequestMapCond.Wait()
			return false
		}()

		if finished {
			return nil
		}
	}
}

// abort stores the specified error and closes the connection.
func (c *Client) abort(ctx context.Context, err error) {
	c.closedMu.Lock()
	if c.closed {
		c.closedMu.Unlock()
		return
	}
	c.closedMu.Unlock()

	isNormalClosure := false
	var ce websocket.CloseError
	if errors.As(err, &ce) {
		// tear down the connection without a new error if this is a regular close
		isNormalClosure = ce.Code == websocket.StatusNormalClosure
	}

	if err != nil && !isNormalClosure {
		log.WithContext(ctx).WithError(err).Error("aborting client")
	}
	c.errMu.Lock()
	c.err = errors.Join(c.err, err)
	c.errMu.Unlock()

	// call this outside of the lock to avoid deadlock should other parts of the
	// code try to call abort() when crashing out of a read or write
	err = c.conn.Close(websocket.StatusNormalClosure, "normal closure")

	c.errMu.Lock()
	c.err = errors.Join(c.err, err)
	c.errMu.Unlock()

	c.closedMu.Lock()
	if c.closed {
		c.closedMu.Unlock()
		return
	}
	c.closed = true
	c.closedCond.Broadcast()
	c.closedMu.Unlock()

	c.closeAllRequestChans()
}

// Close closes the connection and returns any errors from the underlying connection.
func (c *Client) Close(ctx context.Context) error {
	c.abort(ctx, nil)

	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.err
}

func (c *Client) Closed() bool {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()
	return c.closed
}

func (c *Client) createRequestChan(u uuid.UUID) chan *sdp.GatewayResponse {
	r := make(chan *sdp.GatewayResponse, 1)
	c.requestMapMu.Lock()
	defer c.requestMapMu.Unlock()
	c.requestMap[u] = r
	return r
}

func (c *Client) postRequestChan(u uuid.UUID, msg *sdp.GatewayResponse) {
	c.requestMapMu.RLock()
	defer c.requestMapMu.RUnlock()
	r, ok := c.requestMap[u]
	if ok {
		// this write has to happen under the lock to avoid panics when closing the channel
		r <- msg
	}
}

func (c *Client) finishRequestChan(u uuid.UUID) {
	c.requestMapMu.Lock()
	defer c.requestMapMu.Unlock()

	c.finishedRequestMapMu.Lock()
	defer c.finishedRequestMapMu.Unlock()

	delete(c.requestMap, u)
	c.finishedRequestMap[u] = true
	c.finishedRequestMapCond.Broadcast()
}

func (c *Client) closeAllRequestChans() {
	c.requestMapMu.Lock()
	defer c.requestMapMu.Unlock()

	c.finishedRequestMapMu.Lock()
	defer c.finishedRequestMapMu.Unlock()

	for k, v := range c.requestMap {
		close(v)
		c.finishedRequestMap[k] = true
	}
	// clear the map
	c.requestMap = map[uuid.UUID]chan *sdp.GatewayResponse{}
	c.finishedRequestMapCond.Broadcast()
}
