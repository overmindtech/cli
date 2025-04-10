package discovery

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"
	"github.com/overmindtech/cli/sdp-go"
)

// LogAdapter is a singleton from the source that handles GetLogRecordsRequest
// that come in via NATS. The discovery Engine takes care of the common
// implementation details like subscribing to NATS, unpacking the request,
// framing the responses, and error handling. Implementors only need to pass
// results into the LogRecordsStream.
type LogAdapter interface {
	// Get gets called when a GetLogRecordsRequest needs to be processed. To
	// return data to the requestor, use the provided `stream` to send
	// `GetLogRecordsResponse` messages back.
	//
	// If the implementation encounters an error, it should return the error as
	// `SourceError`. To indicate that the error is within the source, set the
	// `SourceError.Upstream` field to `false`. To indicate that the error is
	// with the upstream API, set the `SourceError.Upstream` field to `true`.
	// Always make sure that the error detail is set to a human-readable string
	// that is helpful for debugging.
	//
	// Implementations must not hold on to or share the `stream` object outside
	// of the scope of a single call.
	//
	// Concurrency: Every invocation of this method will happen in its own
	// goroutine, so implementors need to take care of ensuring thread safety.
	//
	// Cancellation: The context passed to this method will be cancelled when
	// any errors are encountered, like the NATS connection closing, the
	// requestor going away, or hitting a deadline. Implementations are expected
	// to timely detect the cancellation and clean up on the way out. After
	// `ctx` is cancelled, the implementation should not attempt to send any
	// more messages to the stream.
	Get(ctx context.Context, req *sdp.GetLogRecordsRequest, stream LogRecordsStream) error

	// Scopes returns all scopes this adapter is capable of handling. This is
	// used by the Engine to subscribe to the correct subjects. The Engine will
	// only call this method once, so implementors don't need to cache the
	// result.
	Scopes() []string
}

type LogRecordsStream interface {
	// Send takes a GetLogRecordsResponse, and forwards it to the caller over
	// NATS. Note that the order of responses is relevant and will be preserved.
	//
	// Errors returned from this method should be treated as fatal, and the
	// stream should be closed. The caller should not attempt to send any more
	// messages after this method returns an error. Basically, treat this like a
	// context cancellation on the `LogAdapter.Get` method.
	//
	// Concurrency: This method is not thread safe. The caller needs to ensure
	// that There is only one call of Send active at any time.
	Send(ctx context.Context, r *sdp.GetLogRecordsResponse) error
}

type LogRecordsStreamImpl struct {
	// The NATS stream that is used to send messages
	stream sdp.EncodedConnection
	// The NATS subject that is used to send messages
	subject string
	// responder has gone away
	responderGone bool

	responses int
	records   int
}

// assert interface implementation
var _ LogRecordsStream = (*LogRecordsStreamImpl)(nil)

func (s *LogRecordsStreamImpl) Send(ctx context.Context, r *sdp.GetLogRecordsResponse) error {
	// immediately return if the gateway is gone
	if s.responderGone {
		return nats.ErrNoResponders
	}

	s.responses += 1
	s.records += len(r.GetRecords())

	// Send the message to the NATS stream
	err := s.stream.Publish(ctx, s.subject, &sdp.NATSGetLogRecordsResponse{
		Content: &sdp.NATSGetLogRecordsResponse_Response{
			Response: r,
		},
	})
	if errors.Is(err, nats.ErrNoResponders) {
		s.responderGone = true
		return err
	}
	if err != nil {
		return err
	}

	return nil
}
