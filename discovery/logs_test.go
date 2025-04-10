package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type testLogAdapter struct {
	t         *testing.T
	expected  *sdp.GetLogRecordsRequest
	responses []*sdp.GetLogRecordsResponse
	err       error
}

// assert interface implementation
var _ LogAdapter = (*testLogAdapter)(nil)

func (t *testLogAdapter) Get(ctx context.Context, request *sdp.GetLogRecordsRequest, stream LogRecordsStream) error {
	if t.expected == nil {
		t.t.Fatalf("expected LogAdapter to not get called, but got %v", request)
	}
	if t.expected.GetScope() != request.GetScope() {
		t.t.Errorf("expected scope %s but got %s", t.expected.GetScope(), request.GetScope())
	}
	if t.expected.GetQuery() != request.GetQuery() {
		t.t.Errorf("expected query %s but got %s", t.expected.GetQuery(), request.GetQuery())
	}
	// Compare timestamp values correctly
	if (t.expected.GetFrom() == nil) != (request.GetFrom() == nil) {
		t.t.Errorf("timestamp nullability mismatch: expected from is nil: %v, got from is nil: %v", t.expected.GetFrom() == nil, request.GetFrom() == nil)
	} else if t.expected.GetFrom() != nil && !t.expected.GetFrom().AsTime().Equal(request.GetFrom().AsTime()) {
		t.t.Errorf("expected from %s but got %s", t.expected.GetFrom().AsTime(), request.GetFrom().AsTime())
	}

	if (t.expected.GetTo() == nil) != (request.GetTo() == nil) {
		t.t.Errorf("timestamp nullability mismatch: expected to is nil: %v, got to is nil: %v", t.expected.GetTo() == nil, request.GetTo() == nil)
	} else if t.expected.GetTo() != nil && !t.expected.GetTo().AsTime().Equal(request.GetTo().AsTime()) {
		t.t.Errorf("expected to %s but got %s", t.expected.GetTo().AsTime(), request.GetTo().AsTime())
	}
	if t.expected.GetMaxRecords() != request.GetMaxRecords() {
		t.t.Errorf("expected maxRecords %d but got %d", t.expected.GetMaxRecords(), request.GetMaxRecords())
	}
	if t.expected.GetStartFromOldest() != request.GetStartFromOldest() {
		t.t.Errorf("expected startFromOldest %v but got %v", t.expected.GetStartFromOldest(), request.GetStartFromOldest())
	}

	for _, r := range t.responses {
		err := stream.Send(ctx, r)
		if err != nil {
			return err
		}
	}
	return t.err
}

func (t *testLogAdapter) Scopes() []string {
	return []string{"test"}
}

func TestLogAdapter_HappyPath(t *testing.T) {
	t.Parallel()

	ts := timestamppb.Now()
	tla := &testLogAdapter{
		t: t,
		expected: &sdp.GetLogRecordsRequest{
			Scope:           "test",
			Query:           "test",
			From:            ts,
			To:              ts,
			MaxRecords:      10,
			StartFromOldest: false,
		},
		responses: []*sdp.GetLogRecordsResponse{
			{
				Records: []*sdp.LogRecord{
					{
						CreatedAt:  timestamppb.Now(),
						ObservedAt: timestamppb.Now(),
						Severity:   sdp.LogSeverity_INFO,
						Body:       "page1/record1",
					},
					{
						CreatedAt:  timestamppb.Now(),
						ObservedAt: timestamppb.Now(),
						Severity:   sdp.LogSeverity_INFO,
						Body:       "page1/record2",
					},
				},
			},
			{
				Records: []*sdp.LogRecord{
					{
						CreatedAt:  timestamppb.Now(),
						ObservedAt: timestamppb.Now(),
						Severity:   sdp.LogSeverity_INFO,
						Body:       "page2/record1",
					},
					{
						CreatedAt:  timestamppb.Now(),
						ObservedAt: timestamppb.Now(),
						Severity:   sdp.LogSeverity_INFO,
						Body:       "page2/record2",
					},
				},
			},
		},
	}

	tc := &sdp.TestConnection{
		Messages: make([]sdp.ResponseMessage, 0),
	}

	e := newEngine(t, "logs.happyPath", nil, tc)
	if e == nil {
		t.Fatal("failed to create engine")
	}

	err := e.SetLogAdapter(tla)
	if err != nil {
		t.Fatal(err)
	}

	err = e.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = e.Stop()
	}()

	_, _ = tc.Subscribe("logs.records.test", sdp.NewNATSGetLogRecordsResponseHandler(
		"",
		func(ctx context.Context, msg *sdp.NATSGetLogRecordsResponse) {
			t.Log("Received message:", msg)
		},
	))

	err = tc.PublishRequest(t.Context(), "logs.scope.test", "logs.records.test", &sdp.NATSGetLogRecordsRequest{
		Request: &sdp.GetLogRecordsRequest{
			Scope:           "test",
			Query:           "test",
			From:            ts,
			To:              ts,
			MaxRecords:      10,
			StartFromOldest: false,
		},
	})
	if err != nil {
		t.Log("Subscriptions:", tc.Subscriptions)
		t.Fatal(err)
	}

	// TODO: properly sync the test to wait for the messages to be sent
	time.Sleep(1 * time.Second)

	tc.MessagesMu.Lock()
	defer tc.MessagesMu.Unlock()

	if len(tc.Messages) != 5 {
		t.Fatalf("expected 5 messages but got %d: %v", len(tc.Messages), tc.Messages)
	}

	started := tc.Messages[1]
	if started.V.(*sdp.NATSGetLogRecordsResponse).GetStatus().GetStatus() != sdp.NATSGetLogRecordsResponseStatus_STARTED {
		t.Errorf("expected status STARTED but got %v", started.V)
	}

	page1 := tc.Messages[2]
	records := page1.V.(*sdp.NATSGetLogRecordsResponse).GetResponse().GetRecords()
	if len(records) != 2 {
		t.Errorf("expected 2 records but got %d: %v", len(records), records)
	}
	if records[0].GetBody() != "page1/record1" {
		t.Errorf("expected page1/record1 but got %v", page1.V)
	}

	page2 := tc.Messages[3]
	records = page2.V.(*sdp.NATSGetLogRecordsResponse).GetResponse().GetRecords()
	if len(records) != 2 {
		t.Errorf("expected 2 records but got %d: %v", len(records), records)
	}
	if records[0].GetBody() != "page2/record1" {
		t.Errorf("expected page2/record1 but got %v", page2.V)
	}

	finished := tc.Messages[4]
	if finished.V.(*sdp.NATSGetLogRecordsResponse).GetStatus().GetStatus() != sdp.NATSGetLogRecordsResponseStatus_FINISHED {
		t.Errorf("expected status FINISHED but got %v", finished.V)
	}
}

func TestLogAdapter_Validation_Scope(t *testing.T) {
	t.Parallel()

	ts := timestamppb.Now()
	tla := &testLogAdapter{
		t:        t,
		expected: nil,
	}

	tc := &sdp.TestConnection{
		Messages: make([]sdp.ResponseMessage, 0),
	}

	e := newEngine(t, "logs.validation_scope", nil, tc)
	if e == nil {
		t.Fatal("failed to create engine")
	}

	err := e.SetLogAdapter(tla)
	if err != nil {
		t.Fatal(err)
	}

	err = e.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = e.Stop()
	}()

	_, _ = tc.Subscribe("logs.records.test", sdp.NewNATSGetLogRecordsResponseHandler(
		"",
		func(ctx context.Context, msg *sdp.NATSGetLogRecordsResponse) {
			t.Log("Received message:", msg)
		},
	))

	err = tc.PublishRequest(t.Context(), "logs.scope.test", "logs.records.test", &sdp.NATSGetLogRecordsRequest{
		Request: &sdp.GetLogRecordsRequest{
			Scope:           "different-scope",
			Query:           "test",
			From:            ts,
			To:              ts,
			MaxRecords:      10,
			StartFromOldest: false,
		},
	})
	if err != nil {
		t.Log("Subscriptions:", tc.Subscriptions)
		t.Fatal(err)
	}

	// TODO: properly sync the test to wait for the messages to be sent
	time.Sleep(1 * time.Second)

	tc.MessagesMu.Lock()
	defer tc.MessagesMu.Unlock()

	if len(tc.Messages) == 0 {
		t.Fatalf("expected messages but got none: %v", tc.Messages)
	}

	msg := tc.Messages[len(tc.Messages)-1]
	if msg.V.(*sdp.NATSGetLogRecordsResponse).GetStatus().GetStatus() != sdp.NATSGetLogRecordsResponseStatus_ERRORED {
		t.Errorf("expected status ERRORED but got %v", msg.V)
	}
}

func TestLogAdapter_Validation_Empty(t *testing.T) {
	t.Parallel()

	ts := timestamppb.Now()
	tla := &testLogAdapter{
		t:        t,
		expected: nil,
	}

	tc := &sdp.TestConnection{
		Messages: make([]sdp.ResponseMessage, 0),
	}

	e := newEngine(t, "logs.validation_scope", nil, tc)
	if e == nil {
		t.Fatal("failed to create engine")
	}

	err := e.SetLogAdapter(tla)
	if err != nil {
		t.Fatal(err)
	}

	err = e.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = e.Stop()
	}()

	_, _ = tc.Subscribe("logs.records.test", sdp.NewNATSGetLogRecordsResponseHandler(
		"",
		func(ctx context.Context, msg *sdp.NATSGetLogRecordsResponse) {
			t.Log("Received message:", msg)
		},
	))

	err = tc.PublishRequest(t.Context(), "logs.scope.test", "logs.records.test", &sdp.NATSGetLogRecordsRequest{
		Request: &sdp.GetLogRecordsRequest{
			Scope:           "test",
			Query:           "",
			From:            ts,
			To:              ts,
			MaxRecords:      10,
			StartFromOldest: false,
		},
	})
	if err != nil {
		t.Log("Subscriptions:", tc.Subscriptions)
		t.Fatal(err)
	}

	// TODO: properly sync the test to wait for the messages to be sent
	time.Sleep(1 * time.Second)

	tc.MessagesMu.Lock()
	defer tc.MessagesMu.Unlock()

	if len(tc.Messages) == 0 {
		t.Fatalf("expected messages but got none: %v", tc.Messages)
	}

	msg := tc.Messages[len(tc.Messages)-1]
	if msg.V.(*sdp.NATSGetLogRecordsResponse).GetStatus().GetStatus() != sdp.NATSGetLogRecordsResponseStatus_ERRORED {
		t.Errorf("expected status ERRORED but got %v", msg.V)
	}
}

func TestLogAdapter_Validation_NoReplyTo(t *testing.T) {
	t.Parallel()

	ts := timestamppb.Now()
	tla := &testLogAdapter{
		t:        t,
		expected: nil,
	}

	tc := &sdp.TestConnection{
		Messages: make([]sdp.ResponseMessage, 0),
	}

	e := newEngine(t, "logs.validation_scope", nil, tc)
	if e == nil {
		t.Fatal("failed to create engine")
	}

	err := e.SetLogAdapter(tla)
	if err != nil {
		t.Fatal(err)
	}

	err = e.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = e.Stop()
	}()

	_, _ = tc.Subscribe("logs.records.test", sdp.NewNATSGetLogRecordsResponseHandler(
		"",
		func(ctx context.Context, msg *sdp.NATSGetLogRecordsResponse) {
			t.Log("Received message:", msg)
		},
	))

	err = tc.Publish(t.Context(), "logs.scope.test", &sdp.NATSGetLogRecordsRequest{
		Request: &sdp.GetLogRecordsRequest{
			Scope:           "test",
			Query:           "test",
			From:            ts,
			To:              ts,
			MaxRecords:      10,
			StartFromOldest: false,
		},
	})
	if err != nil {
		t.Log("Subscriptions:", tc.Subscriptions)
		t.Fatal(err)
	}

	// TODO: properly sync the test to wait for the messages to be sent
	time.Sleep(1 * time.Second)

	tc.MessagesMu.Lock()
	defer tc.MessagesMu.Unlock()

	// only the Request message should be sent, no responses
	if len(tc.Messages) != 1 {
		t.Fatalf("expected 1 message but got %d: %v", len(tc.Messages), tc.Messages)
	}
}
