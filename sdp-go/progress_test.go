package sdp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewQueryProgress(t *testing.T) {
	u := uuid.New()
	q := Query{
		Type:   "person",
		Method: QueryMethod_GET,
		Query:  "dylan",
		RecursionBehaviour: &Query_RecursionBehaviour{
			LinkDepth: 0,
		},
		Scope:       "test",
		IgnoreCache: false,
		UUID:        u[:],
		Deadline:    timestamppb.New(time.Now().Add(20 * time.Second)),
	}

	t.Run("with no start timeout", func(t *testing.T) {
		p := NewQueryProgress(&q, 0)

		if p.StartTimeout != DefaultStartTimeout {
			t.Error("expected StartTimeout to be equal to DefaultStartTimeout")
		}
	})

	t.Run("with a start timeout shorter than the request timeout", func(t *testing.T) {
		timeout := time.Second
		p := NewQueryProgress(&q, timeout)

		if p.StartTimeout != timeout {
			t.Errorf("expected time to be %v got %v", timeout.String(), p.StartTimeout.String())
		}
	})

	t.Run("with a start timeout longer than the request timeout", func(t *testing.T) {
		timeout := 30 * time.Second
		p := NewQueryProgress(&q, timeout)

		if p.StartTimeout != timeout {
			t.Errorf("expected time to be %v got %v", timeout.String(), p.StartTimeout.String())
		}
	})
}

func TestResponseNilPublisher(t *testing.T) {
	ctx := context.Background()

	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	// Start sending responses with a nil connection, should not panic
	rs.Start(ctx, nil, "test", uuid.New())

	// Give it enough time for ~10 responses
	time.Sleep(100 * time.Millisecond)

	// Stop
	rs.DoneWithContext(ctx)
}

func TestResponseSenderDone(t *testing.T) {
	ctx := context.Background()

	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	tp := TestConnection{
		Messages: make([]ResponseMessage, 0),
	}

	// Start sending responses
	rs.Start(ctx, &tp, "test", uuid.New())

	// Give it enough time for ~10 responses
	time.Sleep(100 * time.Millisecond)

	// Stop
	rs.DoneWithContext(ctx)

	// Let it drain down
	time.Sleep(100 * time.Millisecond)

	// Inspect what was sent
	tp.messagesMutex.Lock()
	if len(tp.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tp.Messages))
	}

	// Make sure that the final message was a completion one
	finalMessage := tp.Messages[len(tp.Messages)-1]
	tp.messagesMutex.Unlock()

	if queryResponse, ok := finalMessage.V.(*QueryResponse); ok {
		if finalResponse, ok := queryResponse.GetResponseType().(*QueryResponse_Response); ok {
			if finalResponse.Response.GetState() != ResponderState_COMPLETE {
				t.Errorf("Expected final message state to be COMPLETE (1), found: %v", finalResponse.Response.GetState())
			}
		} else {
			t.Errorf("Final QueryResponse did not contain a valid Response object. Message content type %T", queryResponse.GetResponseType())
		}
	} else {
		t.Errorf("Final message did not contain a valid response object. Message content type %T", finalMessage.V)
	}
}

func TestResponseSenderError(t *testing.T) {
	ctx := context.Background()

	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	tp := TestConnection{
		Messages: make([]ResponseMessage, 0),
	}

	// Start sending responses
	rs.Start(ctx, &tp, "test", uuid.New())

	// Give it enough time for >10 responses
	time.Sleep(120 * time.Millisecond)

	// Stop
	rs.ErrorWithContext(ctx)

	// Let it drain down
	time.Sleep(100 * time.Millisecond)

	// Inspect what was sent
	tp.messagesMutex.Lock()
	if len(tp.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tp.Messages))
	}

	// Make sure that the final message was a completion one
	finalMessage := tp.Messages[len(tp.Messages)-1]
	tp.messagesMutex.Unlock()

	if queryResponse, ok := finalMessage.V.(*QueryResponse); ok {
		if finalResponse, ok := queryResponse.GetResponseType().(*QueryResponse_Response); ok {
			if finalResponse.Response.GetState() != ResponderState_ERROR {
				t.Errorf("Expected final message state to be ERROR, found: %v", finalResponse.Response.GetState())
			}
		} else {
			t.Errorf("Final QueryResponse did not contain a valid Response object. Message content type %T", queryResponse.GetResponseType())
		}
	} else {
		t.Errorf("Final message did not contain a valid response object. Message content type %T", finalMessage.V)
	}
}

func TestResponseSenderCancel(t *testing.T) {
	ctx := context.Background()

	rs := ResponseSender{
		ResponseInterval: (10 * time.Millisecond),
		ResponseSubject:  "responses",
	}

	tp := TestConnection{
		Messages: make([]ResponseMessage, 0),
	}

	// Start sending responses
	rs.Start(ctx, &tp, "test", uuid.New())

	// Give it enough time for >10 responses
	time.Sleep(120 * time.Millisecond)

	// Stop
	rs.CancelWithContext(ctx)

	// Let it drain down
	time.Sleep(100 * time.Millisecond)

	// Inspect what was sent
	tp.messagesMutex.Lock()
	if len(tp.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tp.Messages))
	}

	// Make sure that the final message was a completion one
	finalMessage := tp.Messages[len(tp.Messages)-1]
	tp.messagesMutex.Unlock()

	if queryResponse, ok := finalMessage.V.(*QueryResponse); ok {
		if finalResponse, ok := queryResponse.GetResponseType().(*QueryResponse_Response); ok {
			if finalResponse.Response.GetState() != ResponderState_CANCELLED {
				t.Errorf("Expected final message state to be CANCELLED, found: %v", finalResponse.Response.GetState())
			}
		} else {
			t.Errorf("Final QueryResponse did not contain a valid Response object. Message content type %T", queryResponse.GetResponseType())
		}
	} else {
		t.Errorf("Final message did not contain a valid response object. Message content type %T", finalMessage.V)
	}
}

func TestDefaultResponseInterval(t *testing.T) {
	ctx := context.Background()

	rs := ResponseSender{}

	rs.Start(ctx, &TestConnection{}, "", uuid.New())
	rs.KillWithContext(ctx)

	if rs.ResponseInterval != DefaultResponseInterval {
		t.Fatal("Response sender interval failed to default")
	}
}

// Test object used for validation that metrics are coming out properly
type ExpectedMetrics struct {
	Working    int
	Stalled    int
	Cancelled  int
	Complete   int
	Error      int
	Responders int
}

// Validate Checks that metrics are as expected and returns an error if not
func (em ExpectedMetrics) Validate(qp *QueryProgress) error {
	if x := qp.NumWorking(); x != em.Working {
		return fmt.Errorf("Expected NumWorking to be %v, got %v", em.Working, x)
	}
	if x := qp.NumStalled(); x != em.Stalled {
		return fmt.Errorf("Expected NumStalled to be %v, got %v", em.Stalled, x)
	}
	if x := qp.NumComplete(); x != em.Complete {
		return fmt.Errorf("Expected NumComplete to be %v, got %v", em.Complete, x)
	}
	if x := qp.NumError(); x != em.Error {
		return fmt.Errorf("Expected NumError to be %v, got %v", em.Error, x)
	}
	if x := qp.NumResponders(); x != em.Responders {
		return fmt.Errorf("Expected NumResponders to be %v, got %v", em.Responders, x)
	}
	if x := qp.NumCancelled(); x != em.Cancelled {
		return fmt.Errorf("Expected NumCancelled to be %v, got %v", em.Cancelled, x)
	}

	rStatus := qp.ResponderStates()

	if len(rStatus) != em.Responders {
		return fmt.Errorf("Expected ResponderStatuses to have %v responders, got %v", em.Responders, len(rStatus))
	}

	return nil
}

func TestQueryProgressNormal(t *testing.T) {
	ctx := context.Background()
	rp := NewQueryProgress(&query, 0)
	rp.DrainDelay = 0

	ru1 := uuid.New()
	ru2 := uuid.New()
	ru3 := uuid.New()
	t.Logf("UUIDs: %v %v %v", ru1, ru2, ru3)

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	expected = ExpectedMetrics{
		Working:    0,
		Stalled:    0,
		Complete:   0,
		Error:      0,
		Responders: 0,
	}

	if err := expected.Validate(rp); err != nil {
		t.Error(err)
	}

	t.Run("Processing initial response", func(t *testing.T) {
		// Test the initial response
		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru1[:],
			State:         ResponderState_WORKING,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("Processing when other scopes also responding", func(t *testing.T) {
		// Then another scope starts working
		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru2[:],
			State:         ResponderState_WORKING,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru3[:],
			State:         ResponderState_WORKING,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    3,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 3,
		}

		if err := expected.Validate(rp); err != nil {
			t.Log(rp.ResponderStates())
			t.Error(err)
		}
	})

	t.Run("When some are complete and some are not", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// test 1 still working
		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru1[:],
			State:         ResponderState_WORKING,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		// Test 2 finishes
		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru2[:],
			State:         ResponderState_COMPLETE,
		})

		// Test 3 still working
		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru3[:],
			State:         ResponderState_WORKING,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    2,
			Stalled:    0,
			Complete:   1,
			Error:      0,
			Responders: 3,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("When one is cancelled", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// test 1 still working
		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru1[:],
			State:         ResponderState_WORKING,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		// Test 3 cancelled
		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru3[:],
			State:         ResponderState_CANCELLED,
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   1,
			Error:      0,
			Cancelled:  1,
			Responders: 3,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("When the final responder finishes", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// Test 1 finishes
		rp.ProcessResponse(ctx, &Response{
			Responder:     "test",
			ResponderUUID: ru1[:],
			State:         ResponderState_COMPLETE,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   2,
			Error:      0,
			Cancelled:  1,
			Responders: 3,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	if rp.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestQueryProgressParallel(t *testing.T) {
	rp := NewQueryProgress(&query, 0)
	rp.DrainDelay = 0

	ru1 := uuid.New()

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	expected = ExpectedMetrics{
		Working:    0,
		Stalled:    0,
		Complete:   0,
		Error:      0,
		Responders: 0,
	}

	if err := expected.Validate(rp); err != nil {
		t.Error(err)
	}

	t.Run("Processing many bunched responses", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i != 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Test the initial response
				rp.ProcessResponse(context.Background(), &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				})
			}()
		}

		wg.Wait()

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})
}

func TestQueryProgressStalled(t *testing.T) {
	rp := NewQueryProgress(&query, 0)
	rp.DrainDelay = 0

	ru1 := uuid.New()

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	t.Run("Processing the initial response", func(t *testing.T) {
		// Test the initial response
		rp.ProcessResponse(context.Background(), &Response{
			Responder:     "test",
			ResponderUUID: ru1[:],
			State:         ResponderState_WORKING,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("After a responder has stalled", func(t *testing.T) {
		// Wait long enough for the thing to be marked as stalled
		time.Sleep(20 * time.Millisecond)

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    1,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}

		if _, ok := rp.responders[ru1]; !ok {
			t.Error("Could not get responder for scope test1")
		}
	})

	t.Run("After a responder recovers from a stall", func(t *testing.T) {
		// See if it will un-stall itself
		rp.ProcessResponse(context.Background(), &Response{
			Responder:     "test",
			ResponderUUID: ru1[:],
			State:         ResponderState_COMPLETE,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   1,
			Error:      0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	if rp.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestRogueResponder(t *testing.T) {
	rp := NewQueryProgress(&query, 100*time.Millisecond)
	rp.DrainDelay = 0

	rur := uuid.New()

	// Create our rogue responder that doesn't cancel when it should
	ticker := time.NewTicker(5 * time.Second)
	tickerCtx, tickerCancel := context.WithCancel(context.Background())
	defer tickerCancel()
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				rp.ProcessResponse(context.Background(), &Response{ //nolint: contextcheck // testing a rogue responder
					Responder:     "test",
					ResponderUUID: rur[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(5 * time.Second),
				})
			case <-tickerCtx.Done():
				return
			}
		}
	}()

	time.Sleep(300 * time.Millisecond)

	// Check that we've noticed the testRogue responder
	if rp.allDone() == true {
		t.Error("expected allDone() to be false")
	}

	cancelCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Try to cancel the request. This will never get a response to say it's all
	// done so instead we're expecting it to be forcibly cancelled
	forced := rp.Cancel(cancelCtx, nil)

	if !forced {
		t.Error("expected cancellation to be forced")
	}
}

func TestQueryProgressError(t *testing.T) {
	rp := NewQueryProgress(&query, 0)
	rp.DrainDelay = 0

	ru1 := uuid.New()

	// Make sure that the details are correct initially
	var expected ExpectedMetrics

	t.Run("Processing the initial response", func(t *testing.T) {
		// Test the initial response
		rp.ProcessResponse(context.Background(), &Response{
			Responder:     "test",
			ResponderUUID: ru1[:],
			State:         ResponderState_WORKING,
			NextUpdateIn:  durationpb.New(10 * time.Millisecond),
		})

		expected = ExpectedMetrics{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("After a responder has failed", func(t *testing.T) {
		rp.ProcessResponse(context.Background(), &Response{
			Responder:     "test",
			ResponderUUID: ru1[:],
			State:         ResponderState_ERROR,
		})

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   0,
			Error:      1,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	t.Run("Ensuring that a failed responder does not get marked as stalled", func(t *testing.T) {
		time.Sleep(12 * time.Millisecond)

		expected = ExpectedMetrics{
			Working:    0,
			Stalled:    0,
			Complete:   0,
			Error:      1,
			Responders: 1,
		}

		if err := expected.Validate(rp); err != nil {
			t.Error(err)
		}
	})

	if rp.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestStart(t *testing.T) {
	rp := NewQueryProgress(&query, 0)
	rp.DrainDelay = 0

	conn := TestConnection{}
	responses := make(chan *QueryResponse, 128)
	// this emulates a source
	sourceHit := atomic.Bool{}

	_, err := conn.Subscribe(fmt.Sprintf("request.scope.%v", query.GetScope()), func(msg *nats.Msg) {
		sourceHit.Store(true)
		response := QueryResponse{
			ResponseType: &QueryResponse_NewItem{
				NewItem: &item,
			},
		}
		// Test that the handlers work
		err := conn.Publish(context.Background(), query.Subject(), &response)
		if err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	err = rp.Start(context.Background(), &conn, responses)
	if err != nil {
		t.Fatal(err)
	}

	response := <-responses

	conn.messagesMutex.Lock()
	if len(conn.Messages) != 2 {
		t.Errorf("expected 2 messages to be sent, got %v", len(conn.Messages))
	}
	conn.messagesMutex.Unlock()

	returnedItem := response.GetNewItem()
	if returnedItem == nil {
		t.Fatal("expected item to be returned")
	}
	if returnedItem.Hash() != item.Hash() {
		t.Error("item hash mismatch")
	}
	if !sourceHit.Load() {
		t.Error("source was not hit")
	}
}

func TestAsyncCancel(t *testing.T) {
	t.Run("With no responders", func(t *testing.T) {
		conn := TestConnection{}
		_, err := conn.Subscribe("test", func(msg *nats.Msg) {})
		if err != nil {
			t.Fatal(err)
		}

		rp := NewQueryProgress(&query, 0)
		rp.DrainDelay = 0

		responseChan := make(chan *QueryResponse, 128)
		err = rp.Start(context.Background(), &conn, responseChan)
		if err != nil {
			t.Fatal(err)
		}

		err = rp.AsyncCancel(&conn)
		if err != nil {
			t.Fatal(err)
		}

		t.Run("ensuring the cancel is sent", func(t *testing.T) {
			time.Sleep(100 * time.Millisecond)

			if len(conn.Messages) != 2 {
				t.Fatal("did not receive cancellation message")
			}
		})

		t.Run("ensure it is marked as done", func(t *testing.T) {
			expected := ExpectedMetrics{
				Working:    0,
				Stalled:    0,
				Complete:   0,
				Error:      0,
				Responders: 0,
			}

			if err := expected.Validate(rp); err != nil {
				t.Error(err)
			}
		})

		t.Run("making sure channels closed", func(t *testing.T) {
			// If the chan is still open this will block forever
			<-responseChan
		})
	})

}

func TestExecute(t *testing.T) {
	conn := TestConnection{}
	_, err := conn.Subscribe("request.scope.global", func(msg *nats.Msg) {})
	if err != nil {
		t.Fatal(err)
	}
	u := uuid.New()

	t.Run("with no responders", func(t *testing.T) {
		q := Query{
			Type:   "user",
			Method: QueryMethod_GET,
			Query:  "Dylan",
			RecursionBehaviour: &Query_RecursionBehaviour{
				LinkDepth: 0,
			},
			Scope:       "global",
			IgnoreCache: false,
			UUID:        u[:],
			Deadline:    timestamppb.New(time.Now().Add(10 * time.Second)),
		}

		rp := NewQueryProgress(&q, 100*time.Millisecond)
		rp.DrainDelay = 0

		_, _, err := rp.Execute(context.Background(), &conn)

		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with a full response set", func(t *testing.T) {
		q := Query{
			Type:   "user",
			Method: QueryMethod_GET,
			Query:  "Dylan",
			RecursionBehaviour: &Query_RecursionBehaviour{
				LinkDepth: 0,
			},
			Scope:       "global",
			IgnoreCache: false,
			UUID:        u[:],
			Deadline:    timestamppb.New(time.Now().Add(10 * time.Second)),
		}

		rp := NewQueryProgress(&q, 0)
		rp.DrainDelay = 0

		go func() {
			ru1 := uuid.New()

			delay := 100 * time.Millisecond
			time.Sleep(delay)

			err := conn.Publish(context.Background(), q.Subject(), &QueryResponse{
				ResponseType: &QueryResponse_Response{
					Response: &Response{
						Responder:     "test",
						ResponderUUID: ru1[:],
						State:         ResponderState_WORKING,
						UUID:          q.GetUUID(),
						NextUpdateIn: &durationpb.Duration{
							Seconds: 10,
							Nanos:   0,
						},
					},
				},
			})
			if err != nil {
				t.Error(err)
			}

			time.Sleep(delay)

			err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{
				ResponseType: &QueryResponse_NewItem{
					NewItem: &item,
				},
			})
			if err != nil {
				t.Error(err)
			}

			time.Sleep(delay)

			err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{
				ResponseType: &QueryResponse_NewItem{
					NewItem: &item,
				},
			})
			if err != nil {
				t.Error(err)
			}

			time.Sleep(delay)

			err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{
				ResponseType: &QueryResponse_Response{
					Response: &Response{
						Responder:     "test",
						ResponderUUID: ru1[:],
						State:         ResponderState_COMPLETE,
						UUID:          q.GetUUID(),
					},
				},
			})
			if err != nil {
				t.Error(err)
			}
		}()

		items, errs, err := rp.Execute(context.Background(), &conn)
		if err != nil {
			t.Fatal(err)
		}

		if len(errs) != 0 {
			t.Fatal(errs)
		}

		if rp.NumComplete() != 1 {
			t.Errorf("expected num complete to be 1, got %v", rp.NumComplete())
		}

		if len(items) != 2 {
			t.Errorf("expected 2 items got %v: %v", len(items), items)
		}
	})
}

func TestRealNats(t *testing.T) {
	nc, err := nats.Connect("nats://localhost,nats://nats")
	if err != nil {
		t.Skip("No NATS connection")
	}

	enc := EncodedConnectionImpl{Conn: nc}

	u := uuid.New()
	q := Query{
		Type:   "person",
		Method: QueryMethod_GET,
		Query:  "dylan",
		Scope:  "global",
		UUID:   u[:],
	}

	rp := NewQueryProgress(&q, 0)
	rp.DrainDelay = 0

	ru1 := uuid.New()

	ready := make(chan bool)

	go func() {
		_, err := enc.Subscribe("request.scope.global", NewQueryHandler("test", func(ctx context.Context, handledQuery *Query) {
			delay := 100 * time.Millisecond

			time.Sleep(delay)

			err := enc.Publish(ctx, q.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
				Responder:     "test",
				ResponderUUID: ru1[:],
				State:         ResponderState_WORKING,
				UUID:          q.GetUUID(),
				NextUpdateIn: &durationpb.Duration{
					Seconds: 10,
					Nanos:   0,
				},
			}}})
			if err != nil {
				t.Error(err)
			}

			time.Sleep(delay)

			err = enc.Publish(ctx, q.Subject(), &QueryResponse{ResponseType: &QueryResponse_NewItem{NewItem: &item}})
			if err != nil {
				t.Error(err)
			}

			err = enc.Publish(ctx, q.Subject(), &QueryResponse{ResponseType: &QueryResponse_NewItem{NewItem: &item}})
			if err != nil {
				t.Error(err)
			}

			err = enc.Publish(ctx, q.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
				Responder:     "test",
				ResponderUUID: ru1[:],
				State:         ResponderState_COMPLETE,
				UUID:          q.GetUUID(),
			}}})
			if err != nil {
				t.Error(err)
			}
		}))
		if err != nil {
			t.Error(err)
		}
		ready <- true
	}()

	<-ready

	slowChan := make(chan *QueryResponse)
	err = rp.Start(context.Background(), &enc, slowChan)

	if err != nil {
		t.Fatal(err)
	}

	for i := range slowChan {
		time.Sleep(100 * time.Millisecond)

		t.Log(i)
	}
}

func TestFastFinisher(t *testing.T) {
	// Test for a situation where there is one responder that finishes really
	// quickly and results in the other responders not getting a chance to start
	conn := TestConnection{}

	fast := uuid.New()
	slow := uuid.New()

	progress := NewQueryProgress(newQuery(), 500*time.Millisecond)

	// Set up the fast responder, it should respond immediately and take only
	// 100ms to complete its work
	_, err := conn.Subscribe("request.scope.global", func(msg *nats.Msg) {
		// Make sure this is the request
		var q Query

		err := proto.Unmarshal(msg.Data, &q)

		if err != nil {
			t.Error(err)
		}

		// Respond immediately saying we're started
		err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
			Responder:     "test",
			ResponderUUID: fast[:],
			State:         ResponderState_WORKING,
			UUID:          q.GetUUID(),
			NextUpdateIn: &durationpb.Duration{
				Seconds: 1,
				Nanos:   0,
			},
		}}})
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(100 * time.Millisecond)

		// Send an item
		err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{ResponseType: &QueryResponse_NewItem{NewItem: newItem()}})
		if err != nil {
			t.Fatal(err)
		}

		// Send a complete message
		err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
			Responder:     "test",
			ResponderUUID: fast[:],
			State:         ResponderState_COMPLETE,
			UUID:          q.GetUUID(),
		}}})
		if err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	// Set up another responder that takes 250ms to start
	_, err = conn.Subscribe("request.scope.global", func(msg *nats.Msg) {
		// Unmarshal the query
		var q Query

		err := proto.Unmarshal(msg.Data, &q)

		if err != nil {
			t.Error(err)
		}

		// Wait 250ms before starting
		time.Sleep(250 * time.Millisecond)

		err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
			Responder:     "test",
			ResponderUUID: slow[:],
			State:         ResponderState_WORKING,
			UUID:          q.GetUUID(),
			NextUpdateIn: &durationpb.Duration{
				Seconds: 1,
				Nanos:   0,
			},
		}}})
		if err != nil {
			t.Fatal(err)
		}

		// Send an item
		item := newItem()
		err = item.GetAttributes().Set("name", "baz")
		if err != nil {
			t.Fatal(err)
		}
		err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{ResponseType: &QueryResponse_NewItem{NewItem: item}})
		if err != nil {
			t.Fatal(err)
		}

		// Send a complete message
		err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
			Responder:     "test",
			ResponderUUID: slow[:],
			State:         ResponderState_COMPLETE,
			UUID:          q.GetUUID(),
		}}})
		if err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	items, errs, err := progress.Execute(context.Background(), &conn)

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d: %v", len(items), items)
	}

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}
}
