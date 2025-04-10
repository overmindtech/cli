package sdpws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Sends a query on the websocket connection without waiting for responses. Use
// the `Wait()` method to wait for completion of requests based on their UUID
func (c *Client) SendQuery(ctx context.Context, q *sdp.Query) error {
	if c.Closed() {
		return errors.New("client closed")
	}

	log.WithContext(ctx).WithField("query", q).Trace("writing query to websocket")
	err := c.send(ctx, &sdp.GatewayRequest{
		RequestType: &sdp.GatewayRequest_Query{
			Query: q,
		},
		MinStatusInterval: durationpb.New(time.Second),
	})
	if err != nil {
		// c.send already aborts
		// c.abort(ctx, err)
		return fmt.Errorf("error sending query: %w", err)
	}
	return nil
}

// Run a query and wait for it to complete, returning all of the items that were
// found.
func (c *Client) Query(ctx context.Context, q *sdp.Query) ([]*sdp.Item, error) {
	if c.Closed() {
		return nil, errors.New("client closed")
	}

	u := uuid.UUID(q.GetUUID())

	r := c.createRequestChan(u)
	defer c.finishRequestChan(u)

	err := c.SendQuery(ctx, q)
	if err != nil {
		// c.SendQuery already aborts
		// c.abort(ctx, err)
		return nil, err
	}

	items := make([]*sdp.Item, 0)

	var otherErr *sdp.QueryError
readLoop:
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled: %w", ctx.Err())
		case resp, more := <-r:
			if !more {
				break readLoop
			}
			switch resp.GetResponseType().(type) {
			case *sdp.GatewayResponse_NewItem:
				item := resp.GetNewItem()
				log.WithContext(ctx).WithField("query", q).WithField("item", item).Trace("received item")
				items = append(items, item)
			case *sdp.GatewayResponse_QueryError:
				qe := resp.GetQueryError()
				log.WithContext(ctx).WithField("query", q).WithField("queryError", qe).Trace("received query error")
				switch qe.GetErrorType() {
				case sdp.QueryError_OTHER, sdp.QueryError_TIMEOUT, sdp.QueryError_NOSCOPE:
					// record that we received an error, but continue reading
					// if we receive any item, mapping was still successful
					otherErr = qe
					continue readLoop
				case sdp.QueryError_NOTFOUND:
					// never record not found as an error
					continue readLoop
				}
			case *sdp.GatewayResponse_QueryStatus:
				qs := resp.GetQueryStatus()
				span := trace.SpanFromContext(ctx)
				span.SetAttributes(attribute.String("ovm.sdp.lastQueryStatus", qs.String()))
				log.WithContext(ctx).WithField("query", q).WithField("queryStatus", qs).Trace("received query status")
				switch qs.GetStatus() { //nolint:exhaustive // we dont care about sdp.QueryStatus_UNSPECIFIED, sdp.QueryStatus_STARTED
				case sdp.QueryStatus_FINISHED:
					break readLoop
				case sdp.QueryStatus_CANCELLED:
					return nil, errors.New("query cancelled")
				case sdp.QueryStatus_ERRORED:
					// if we already received items, we can ignore the error
					if len(items) == 0 && otherErr != nil {
						err = fmt.Errorf("query errored: %w", otherErr)
						// query errors should not abort the connection
						// c.abort(ctx, err)
						return nil, err
					}
					break readLoop
				}
			default:
				log.WithContext(ctx).WithField("response", resp).WithField("responseType", fmt.Sprintf("%T", resp.GetResponseType())).Warn("unexpected response")
			}
		}
	}

	return items, nil
}

// TODO: CancelQuery
// TODO: Expand

// Sends a LoadSnapshot request on the websocket connection without waiting for
// a response.
func (c *Client) SendLoadSnapshot(ctx context.Context, s *sdp.LoadSnapshot) error {
	if c.Closed() {
		return errors.New("client closed")
	}

	log.WithContext(ctx).WithField("snapshot", s).Trace("loading snapshot via websocket")
	err := c.send(ctx, &sdp.GatewayRequest{
		RequestType: &sdp.GatewayRequest_LoadSnapshot{
			LoadSnapshot: s,
		},
	})
	if err != nil {
		return fmt.Errorf("error sending load snapshot: %w", err)
	}
	return nil
}

// Load a snapshot and wait for it to complete. This will return the
// SnapshotLoadResult from the gateway. A separate error is only returned when
// there is a communication error. Logic errors from the gateway are reported
// through the returned SnapshotLoadResult.
func (c *Client) LoadSnapshot(ctx context.Context, id uuid.UUID) (*sdp.SnapshotLoadResult, error) {
	if c.Closed() {
		return nil, errors.New("client closed")
	}

	u := uuid.New()
	s := &sdp.LoadSnapshot{
		UUID:  id[:],
		MsgID: u[:],
	}
	r := c.createRequestChan(u)

	err := c.SendLoadSnapshot(ctx, s)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled: %w", ctx.Err())
		case resp, more := <-r:
			if !more {
				return nil, errors.New("request channel closed")
			}
			switch resp.GetResponseType().(type) {
			case *sdp.GatewayResponse_SnapshotLoadResult:
				slr := resp.GetSnapshotLoadResult()
				log.WithContext(ctx).WithField("snapshot", s).WithField("snapshotLoadResult", slr).Trace("received snapshot load result")
				return slr, nil
			default:
				log.WithContext(ctx).WithField("response", resp).WithField("responseType", fmt.Sprintf("%T", resp.GetResponseType())).Warn("unexpected response")
				return nil, errors.New("unexpected response")
			}
		}
	}
}

// Sends a StoreSnapshot request on the websocket connection without waiting for
// a response.
func (c *Client) SendStoreSnapshot(ctx context.Context, s *sdp.StoreSnapshot) error {
	if c.Closed() {
		return errors.New("client closed")
	}

	log.WithContext(ctx).WithField("snapshot", s).Trace("storing snapshot via websocket")
	err := c.send(ctx, &sdp.GatewayRequest{
		RequestType: &sdp.GatewayRequest_StoreSnapshot{
			StoreSnapshot: s,
		},
	})
	if err != nil {
		return fmt.Errorf("error sending store snapshot: %w", err)
	}
	return nil
}

// Store a snapshot and wait for it to complete, returning the UUID of the
// snapshot that was created.
func (c *Client) StoreSnapshot(ctx context.Context, name, description string) (uuid.UUID, error) {
	if c.Closed() {
		return uuid.UUID{}, errors.New("client closed")
	}

	u := uuid.New()
	s := &sdp.StoreSnapshot{
		Name:        name,
		Description: description,
		MsgID:       u[:],
	}
	r := c.createRequestChan(u)

	err := c.SendStoreSnapshot(ctx, s)
	if err != nil {
		return uuid.UUID{}, err
	}

	for {
		select {
		case <-ctx.Done():
			return uuid.UUID{}, fmt.Errorf("context canceled: %w", ctx.Err())
		case resp, more := <-r:
			if !more {
				return uuid.UUID{}, errors.New("request channel closed")
			}
			switch resp.GetResponseType().(type) {
			case *sdp.GatewayResponse_SnapshotStoreResult:
				ssr := resp.GetSnapshotStoreResult()
				log.WithContext(ctx).WithField("Snapshot", s).WithField("snapshotStoreResult", ssr).Trace("received snapshot store result")
				if ssr.GetSuccess() {
					return uuid.UUID(ssr.GetSnapshotID()), nil
				}
				return uuid.UUID{}, fmt.Errorf("snapshot store failed: %v", ssr.GetErrorMessage())
			default:
				log.WithContext(ctx).WithField("response", resp).WithField("responseType", fmt.Sprintf("%T", resp.GetResponseType())).Warn("unexpected response")
				return uuid.UUID{}, errors.New("unexpected response")
			}
		}
	}
}

func (c *Client) SendLoadBookmark(ctx context.Context, b *sdp.LoadBookmark) error {
	if c.Closed() {
		return errors.New("client closed")
	}

	log.WithContext(ctx).WithField("bookmark", b).Trace("loading bookmark via websocket")
	err := c.send(ctx, &sdp.GatewayRequest{
		RequestType: &sdp.GatewayRequest_LoadBookmark{
			LoadBookmark: b,
		},
	})
	if err != nil {
		return fmt.Errorf("error sending load bookmark: %w", err)
	}
	return nil
}

// Sends a StoreBookmark request on the websocket connection without waiting for
// a response.
func (c *Client) SendStoreBookmark(ctx context.Context, b *sdp.StoreBookmark) error {
	if c.Closed() {
		return errors.New("client closed")
	}

	log.WithContext(ctx).WithField("bookmark", b).Trace("storing bookmark via websocket")
	err := c.send(ctx, &sdp.GatewayRequest{
		RequestType: &sdp.GatewayRequest_StoreBookmark{
			StoreBookmark: b,
		},
	})
	if err != nil {
		return fmt.Errorf("error sending store bookmark: %w", err)
	}
	return nil
}

// Store a bookmark and wait for it to complete, returning the UUID of the
// bookmark that was created.
func (c *Client) StoreBookmark(ctx context.Context, name, description string, isSystem bool) (uuid.UUID, error) {
	if c.Closed() {
		return uuid.UUID{}, errors.New("client closed")
	}

	u := uuid.New()
	b := &sdp.StoreBookmark{
		Name:        name,
		Description: description,
		MsgID:       u[:],
		IsSystem:    true,
	}
	r := c.createRequestChan(u)

	err := c.SendStoreBookmark(ctx, b)
	if err != nil {
		return uuid.UUID{}, err
	}

	for {
		select {
		case <-ctx.Done():
			return uuid.UUID{}, fmt.Errorf("context canceled: %w", ctx.Err())
		case resp, more := <-r:
			if !more {
				return uuid.UUID{}, errors.New("request channel closed")
			}
			switch resp.GetResponseType().(type) {
			case *sdp.GatewayResponse_BookmarkStoreResult:
				bsr := resp.GetBookmarkStoreResult()
				log.WithContext(ctx).WithField("bookmark", b).WithField("bookmarkStoreResult", bsr).Trace("received bookmark store result")
				if bsr.GetSuccess() {
					return uuid.UUID(bsr.GetBookmarkID()), nil
				}
				return uuid.UUID{}, fmt.Errorf("bookmark store failed: %v", bsr.GetErrorMessage())
			default:
				log.WithContext(ctx).WithField("response", resp).WithField("responseType", fmt.Sprintf("%T", resp.GetResponseType())).Warn("unexpected response")
				return uuid.UUID{}, errors.New("unexpected response")
			}
		}
	}
}

// TODO: LoadBookmark

// send chatMessage to the assistant
func (c *Client) SendChatMessage(ctx context.Context, m *sdp.ChatMessage) error {
	if c.Closed() {
		return errors.New("client closed")
	}

	log.WithContext(ctx).WithField("message", m).Trace("sending chat message via websocket")
	err := c.send(ctx, &sdp.GatewayRequest{
		RequestType: &sdp.GatewayRequest_ChatMessage{
			ChatMessage: m,
		},
	})
	if err != nil {
		return fmt.Errorf("error sending chat message: %w", err)
	}
	return nil
}
