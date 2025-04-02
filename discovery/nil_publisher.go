package discovery

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// When testing this library, or running without a real NATS connection, it is
// necessary to create a fake publisher rather than pass in a nil pointer. This
// is due to the fact that the NATS libraries will panic if a method is called
// on a nil pointer
type NilConnection struct{}

// assert interface implementation
var _ sdp.EncodedConnection = (*NilConnection)(nil)

// Publish Does nothing except log an error
func (n NilConnection) Publish(ctx context.Context, subj string, m proto.Message) error {
	log.WithFields(log.Fields{
		"subject": subj,
		"message": fmt.Sprint(m),
	}).Error("Could not publish NATS message due to no connection")

	return nil
}

// PublishRequest Does nothing except log an error
func (n NilConnection) PublishRequest(ctx context.Context, subj, replyTo string, m proto.Message) error {
	log.WithFields(log.Fields{
		"subject": subj,
		"replyTo": replyTo,
		"message": fmt.Sprint(m),
	}).Error("Could not publish NATS message request due to no connection")

	return nil
}

// PublishMsg Does nothing except log an error
func (n NilConnection) PublishMsg(ctx context.Context, msg *nats.Msg) error {
	log.WithFields(log.Fields{
		"subject": msg.Subject,
		"message": fmt.Sprint(msg),
	}).Error("Could not publish NATS message due to no connection")

	return nil
}

// Subscribe Does nothing except log an error
func (n NilConnection) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	log.WithFields(log.Fields{
		"subject": subj,
	}).Error("Could not subscribe to NAT subject due to no connection")

	return nil, nil
}

// QueueSubscribe Does nothing except log an error
func (n NilConnection) QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error) {
	log.WithFields(log.Fields{
		"subject": subj,
		"queue":   queue,
	}).Error("Could not subscribe to NAT subject queue due to no connection")

	return nil, nil
}

// Request Does nothing except log an error
func (n NilConnection) RequestMsg(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	log.WithFields(log.Fields{
		"subject": msg.Subject,
		"message": fmt.Sprint(msg),
	}).Error("Could not publish NATS request due to no connection")

	return nil, nil
}

// Status Always returns nats.CONNECTED
func (n NilConnection) Status() nats.Status {
	return nats.CONNECTED
}

// Stats Always returns empty/zero nats.Statistics
func (n NilConnection) Stats() nats.Statistics {
	return nats.Statistics{}
}

// LastError Always returns nil
func (n NilConnection) LastError() error {
	return nil
}

// Drain Always returns nil
func (n NilConnection) Drain() error {
	return nil
}

// Close Does nothing
func (n NilConnection) Close() {}

// Underlying Always returns nil
func (n NilConnection) Underlying() *nats.Conn {
	return nil
}

// Drop Does nothing
func (n NilConnection) Drop() {}
