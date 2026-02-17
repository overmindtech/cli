package sdp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	sync "sync"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type ResponseMessage struct {
	Subject string
	V       interface{}
}

// TestConnection Used to mock a NATS connection for testing
type TestConnection struct {
	Messages   []ResponseMessage
	MessagesMu sync.Mutex

	// If set, the test connection will not return ErrNoResponders if someone
	// tries to publish a message to a subject with no responders
	IgnoreNoResponders bool

	Subscriptions      map[*regexp.Regexp][]nats.MsgHandler
	subscriptionsMutex sync.RWMutex
}

// assert interface implementation
var _ EncodedConnection = (*TestConnection)(nil)

// Publish Test publish method, notes down the subject and the message
func (t *TestConnection) Publish(ctx context.Context, subj string, m proto.Message) error {
	t.MessagesMu.Lock()
	t.Messages = append(t.Messages, ResponseMessage{
		Subject: subj,
		V:       m,
	})
	t.MessagesMu.Unlock()

	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	msg := nats.Msg{
		Subject: subj,
		Data:    data,
	}
	return t.runHandlers(&msg)
}

// PublishRequest Test publish method, notes down the subject and the message
func (t *TestConnection) PublishRequest(ctx context.Context, subj, replyTo string, m proto.Message) error {
	t.MessagesMu.Lock()
	t.Messages = append(t.Messages, ResponseMessage{
		Subject: subj,
		V:       m,
	})
	t.MessagesMu.Unlock()

	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	msg := nats.Msg{
		Subject: subj,
		Data:    data,
		Header:  nats.Header{},
	}
	msg.Header.Add("reply-to", replyTo)
	return t.runHandlers(&msg)
}

// PublishMsg Test publish method, notes down the subject and the message
func (t *TestConnection) PublishMsg(ctx context.Context, msg *nats.Msg) error {
	t.MessagesMu.Lock()
	t.Messages = append(t.Messages, ResponseMessage{
		Subject: msg.Subject,
		V:       msg.Data,
	})
	t.MessagesMu.Unlock()

	err := t.runHandlers(msg)
	if err != nil {
		return err
	}

	return nil
}

func (t *TestConnection) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	t.subscriptionsMutex.Lock()
	defer t.subscriptionsMutex.Unlock()

	if t.Subscriptions == nil {
		t.Subscriptions = make(map[*regexp.Regexp][]nats.MsgHandler)
	}

	regex := t.subjectToRegexp(subj)

	t.Subscriptions[regex] = append(t.Subscriptions[regex], cb)

	return nil, nil
}

func (t *TestConnection) QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error) {
	// TODO: implement queue groups here
	return t.Subscribe(subj, cb)
}

func (r *TestConnection) subjectToRegexp(subject string) *regexp.Regexp {
	// If the subject contains a > then handle this
	if strings.Contains(subject, ">") {
		// Escape regex to literal
		quoted := regexp.QuoteMeta(subject)

		// Replace > with .*$
		return regexp.MustCompile(strings.ReplaceAll(quoted, ">", ".*$"))
	}

	if strings.Contains(subject, "*") {
		// Escape regex to literal
		quoted := regexp.QuoteMeta(subject)

		// Replace \* with \w+
		return regexp.MustCompile(strings.ReplaceAll(quoted, `\*`, `\w+`))
	}

	return regexp.MustCompile(regexp.QuoteMeta(subject))
}

// RequestMsg Simulates a request on the given subject, assigns a random
// response subject then calls the handler on the given subject, we are
// expecting the handler to be in the format: func(msg *nats.Msg)
func (t *TestConnection) RequestMsg(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	replySubject := randSeq(10)
	msg.Reply = replySubject
	replies := make(chan interface{}, 128)

	// Subscribe to the reply subject
	_, err := t.Subscribe(replySubject, func(msg *nats.Msg) {
		replies <- msg
	})
	if err != nil {
		return nil, err
	}
	// Run the handlers
	err = t.runHandlers(msg)
	if err != nil {
		return nil, err
	}

	// Return the first result
	select {
	case reply, ok := <-replies:
		if ok {
			if m, ok := reply.(*nats.Msg); ok {
				return &nats.Msg{
					Subject: replySubject,
					Data:    m.Data,
				}, nil
			} else {
				return nil, fmt.Errorf("reply was not a *nats.Msg, but a %T", reply)
			}
		} else {
			return nil, errors.New("no replies")
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Status Always returns nats.CONNECTED
func (n *TestConnection) Status() nats.Status {
	return nats.CONNECTED
}

// Stats Always returns empty/zero nats.Statistics
func (n *TestConnection) Stats() nats.Statistics {
	return nats.Statistics{}
}

// LastError Always returns nil
func (n *TestConnection) LastError() error {
	return nil
}

// Drain Always returns nil
func (n *TestConnection) Drain() error {
	return nil
}

// Close Does nothing
func (n *TestConnection) Close() {}

// Underlying Always returns nil
func (n *TestConnection) Underlying() *nats.Conn {
	return &nats.Conn{}
}

// Drop Does nothing
func (n *TestConnection) Drop() {}

// runHandlers Runs the handlers for a given subject
func (t *TestConnection) runHandlers(msg *nats.Msg) error {
	t.subscriptionsMutex.RLock()
	defer t.subscriptionsMutex.RUnlock()

	var hasResponder bool

	for subjectRegex, handlers := range t.Subscriptions {
		if subjectRegex.MatchString(msg.Subject) {
			for _, handler := range handlers {
				hasResponder = true
				handler(msg)
			}
		}
	}

	if hasResponder || t.IgnoreNoResponders {
		return nil
	} else {
		return nats.ErrNoResponders
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] //nolint:gosec // This is not for security
	}
	return string(b)
}
