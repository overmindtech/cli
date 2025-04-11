package sdp

import (
	"context"
	"fmt"
	reflect "reflect"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// EncodedConnection is an interface that allows messages to be published to it.
// In production this would always be filled by a *nats.EncodedConn, however in
// testing we will mock this with something that does nothing
type EncodedConnection interface {
	Publish(ctx context.Context, subj string, m proto.Message) error
	PublishRequest(ctx context.Context, subj, replyTo string, m proto.Message) error
	PublishMsg(ctx context.Context, msg *nats.Msg) error
	Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
	QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error)
	RequestMsg(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)

	Status() nats.Status
	Stats() nats.Statistics
	LastError() error

	Drain() error
	Close()

	Underlying() *nats.Conn
	Drop()
}

type EncodedConnectionImpl struct {
	Conn *nats.Conn
}

// assert interface implementation
var _ EncodedConnection = (*EncodedConnectionImpl)(nil)

func recordMessage(ctx context.Context, name, subj, typ, msg string) {
	log.WithContext(ctx).WithFields(log.Fields{
		"msg":  msg,
		"subj": subj,
		"typ":  typ,
	}).Trace(name)
	// avoid spamming honeycomb
	if log.GetLevel() == log.TraceLevel {
		span := trace.SpanFromContext(ctx)
		span.AddEvent(name, trace.WithAttributes(
			attribute.String("ovm.sdp.subject", subj),
			attribute.String("ovm.sdp.message", msg),
		))
	}
}

func (ec *EncodedConnectionImpl) Publish(ctx context.Context, subj string, m proto.Message) error {
	// TODO: protojson.Format is pretty expensive, replace with summarized data
	recordMessage(ctx, "Publish", subj, fmt.Sprint(reflect.TypeOf(m)), protojson.Format(m))

	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: subj,
		Data:    data,
	}
	InjectOtelTraceContext(ctx, msg)
	return ec.Conn.PublishMsg(msg)
}

func (ec *EncodedConnectionImpl) PublishRequest(ctx context.Context, subj, replyTo string, m proto.Message) error {
	// TODO: protojson.Format is pretty expensive, replace with summarized data
	recordMessage(ctx, "Publish", subj, fmt.Sprint(reflect.TypeOf(m)), protojson.Format(m))

	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}

	msg := &nats.Msg{
		Subject: subj,
		Data:    data,
	}
	msg.Header.Add("reply-to", replyTo)
	InjectOtelTraceContext(ctx, msg)
	return ec.Conn.PublishMsg(msg)
}

func (ec *EncodedConnectionImpl) PublishMsg(ctx context.Context, msg *nats.Msg) error {
	recordMessage(ctx, "Publish", msg.Subject, "[]byte", "binary")

	InjectOtelTraceContext(ctx, msg)
	return ec.Conn.PublishMsg(msg)
}

// Subscribe Use genhandler to get a nats.MsgHandler with otel propagation and protobuf marshaling
func (ec *EncodedConnectionImpl) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return ec.Conn.Subscribe(subj, cb)
}

// QueueSubscribe Use genhandler to get a nats.MsgHandler with otel propagation and protobuf marshaling
func (ec *EncodedConnectionImpl) QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return ec.Conn.QueueSubscribe(subj, queue, cb)
}

func (ec *EncodedConnectionImpl) RequestMsg(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	recordMessage(ctx, "RequestMsg", msg.Subject, "[]byte", "binary")
	InjectOtelTraceContext(ctx, msg)
	reply, err := ec.Conn.RequestMsgWithContext(ctx, msg)

	if err != nil {
		recordMessage(ctx, "RequestMsg Error", msg.Subject, fmt.Sprint(reflect.TypeOf(err)), err.Error())
	} else {
		recordMessage(ctx, "RequestMsg Reply", msg.Subject, "[]byte", "binary")
	}
	return reply, err
}

func (ec *EncodedConnectionImpl) Drain() error {
	return ec.Conn.Drain()
}
func (ec *EncodedConnectionImpl) Close() {
	ec.Conn.Close()
}

func (ec *EncodedConnectionImpl) Status() nats.Status {
	return ec.Conn.Status()
}

func (ec *EncodedConnectionImpl) Stats() nats.Statistics {
	return ec.Conn.Stats()
}

func (ec *EncodedConnectionImpl) LastError() error {
	return ec.Conn.LastError()
}

func (ec *EncodedConnectionImpl) Underlying() *nats.Conn {
	return ec.Conn
}

// Drop Drops the underlying connection completely
func (ec *EncodedConnectionImpl) Drop() {
	ec.Conn = nil
}

// Unmarshal Does a proto.Unmarshal and logs errors in a consistent way. The
// user should still validate that the message is valid as it's possible to
// unmarshal data from one message format into another without an error.
// Validation should be based on the type that the data is being unmarshaled
// into.
func Unmarshal(ctx context.Context, b []byte, m proto.Message) error {
	err := proto.Unmarshal(b, m)
	if err != nil {
		recordMessage(ctx, "Unmarshal err", "unknown", fmt.Sprint(reflect.TypeOf(err)), err.Error())
		log.WithContext(ctx).Errorf("Error parsing message: %v", err)
		trace.SpanFromContext(ctx).SetStatus(codes.Error, fmt.Sprintf("Error parsing message: %v", err))
		return err
	}

	recordMessage(ctx, "Unmarshal", "unknown", fmt.Sprint(reflect.TypeOf(m)), protojson.Format(m))
	return nil
}

//go:generate go run genhandler.go Query
//go:generate go run genhandler.go QueryResponse
//go:generate go run genhandler.go CancelQuery

//go:generate go run genhandler.go GatewayResponse

//go:generate go run genhandler.go NATSGetLogRecordsRequest
//go:generate go run genhandler.go NATSGetLogRecordsResponse
