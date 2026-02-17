package sdp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
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

func TestRunSourceQueryParams(t *testing.T) {
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
		_, err := RunSourceQuery(t.Context(), &q, 0, nil, nil)

		if err == nil {
			t.Error("expected an error when there is not startTimeout")
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

	tc := TestConnection{IgnoreNoResponders: true}

	// Start sending responses
	rs.Start(ctx, &tc, "test", uuid.New())

	// Give it enough time for ~10 responses
	time.Sleep(100 * time.Millisecond)

	// Stop
	rs.DoneWithContext(ctx)

	// Let it drain down
	time.Sleep(100 * time.Millisecond)

	// Inspect what was sent
	tc.MessagesMu.Lock()
	if len(tc.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tc.Messages))
	}

	// Make sure that the final message was a completion one
	finalMessage := tc.Messages[len(tc.Messages)-1]
	tc.MessagesMu.Unlock()

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

	tc := TestConnection{IgnoreNoResponders: true}

	// Start sending responses
	rs.Start(ctx, &tc, "test", uuid.New())

	// Give it enough time for >10 responses
	time.Sleep(120 * time.Millisecond)

	// Stop
	rs.ErrorWithContext(ctx)

	// Let it drain down
	time.Sleep(100 * time.Millisecond)

	// Inspect what was sent
	tc.MessagesMu.Lock()
	if len(tc.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tc.Messages))
	}

	// Make sure that the final message was a completion one
	finalMessage := tc.Messages[len(tc.Messages)-1]
	tc.MessagesMu.Unlock()

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

	tc := TestConnection{IgnoreNoResponders: true}

	// Start sending responses
	rs.Start(ctx, &tc, "test", uuid.New())

	// Give it enough time for >10 responses
	time.Sleep(120 * time.Millisecond)

	// Stop
	rs.CancelWithContext(ctx)

	// Let it drain down
	time.Sleep(100 * time.Millisecond)

	// Inspect what was sent
	tc.MessagesMu.Lock()
	if len(tc.Messages) <= 10 {
		t.Errorf("Expected <= 10 responses to be sent, found %v", len(tc.Messages))
	}

	// Make sure that the final message was a completion one
	finalMessage := tc.Messages[len(tc.Messages)-1]
	tc.MessagesMu.Unlock()

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

// ExpectToMatch Checks that metrics are as expected and returns an error if not
func (expected SourceQueryProgress) ExpectToMatch(qp *SourceQuery) error {
	actual := qp.Progress()

	var err error

	if expected.Working != actual.Working {
		err = errors.Join(err, fmt.Errorf("Expected Working to be %v, got %v", expected.Working, actual.Working))
	}
	if expected.Stalled != actual.Stalled {
		err = errors.Join(err, fmt.Errorf("Expected Stalled to be %v, got %v", expected.Stalled, actual.Stalled))
	}
	if expected.Complete != actual.Complete {
		err = errors.Join(err, fmt.Errorf("Expected Complete to be %v, got %v", expected.Complete, actual.Complete))
	}
	if expected.Error != actual.Error {
		err = errors.Join(err, fmt.Errorf("Expected Error to be %v, got %v", expected.Error, actual.Error))
	}
	if expected.Responders != actual.Responders {
		err = errors.Join(err, fmt.Errorf("Expected Responders to be %v, got %v", expected.Responders, actual.Responders))
	}
	if expected.Cancelled != actual.Cancelled {
		err = errors.Join(err, fmt.Errorf("Expected Cancelled to be %v, got %v", expected.Cancelled, actual.Cancelled))
	}

	return err
}

// Create a channel that discards everything
func devNull() chan<- *QueryResponse {
	c := make(chan *QueryResponse, 128)
	go func() {
		for range c {
		}
	}()
	return c
}

func TestQueryProgressNormal(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tc := TestConnection{IgnoreNoResponders: true}
	sq, err := RunSourceQuery(ctx, &query, DefaultStartTimeout, &tc, devNull())
	if err != nil {
		t.Fatal(err)
	}

	ru1 := uuid.New()
	ru2 := uuid.New()
	ru3 := uuid.New()
	t.Logf("UUIDs: %v %v %v", ru1, ru2, ru3)

	// Make sure that the details are correct initially
	var expected SourceQueryProgress

	expected = SourceQueryProgress{
		Working:    0,
		Stalled:    0,
		Complete:   0,
		Error:      0,
		Responders: 0,
	}

	if err := expected.ExpectToMatch(sq); err != nil {
		t.Error(err)
	}

	t.Run("Processing initial response", func(t *testing.T) {
		// Test the initial response
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	t.Run("Processing when other scopes also responding", func(t *testing.T) {
		// Then another scope starts working
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru2[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru3[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    3,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 3,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	t.Run("When some are complete and some are not", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// test 1 still working
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		// Test 2 finishes
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru2[:],
					State:         ResponderState_COMPLETE,
				},
			},
		})

		// Test 3 still working
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru3[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    2,
			Stalled:    0,
			Complete:   1,
			Error:      0,
			Responders: 3,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	t.Run("When one is cancelled", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// test 1 still working
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		// Test 3 cancelled
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru3[:],
					State:         ResponderState_CANCELLED,
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    1,
			Stalled:    0,
			Complete:   1,
			Error:      0,
			Cancelled:  1,
			Responders: 3,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	t.Run("When the final responder finishes", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond)

		// Test 1 finishes
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_COMPLETE,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    0,
			Stalled:    0,
			Complete:   2,
			Error:      0,
			Cancelled:  1,
			Responders: 3,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	if sq.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestQueryProgressParallel(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tc := TestConnection{IgnoreNoResponders: true}
	sq, err := RunSourceQuery(ctx, &query, DefaultStartTimeout, &tc, devNull())
	if err != nil {
		t.Fatal(err)
	}

	ru1 := uuid.New()

	// Make sure that the details are correct initially
	var expected SourceQueryProgress

	expected = SourceQueryProgress{
		Working:    0,
		Stalled:    0,
		Complete:   0,
		Error:      0,
		Responders: 0,
	}

	if err := expected.ExpectToMatch(sq); err != nil {
		t.Error(err)
	}

	t.Run("Processing many bunched responses", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i != 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Test the initial response
				sq.handleQueryResponse(ctx, &QueryResponse{
					ResponseType: &QueryResponse_Response{
						Response: &Response{
							Responder:     "test",
							ResponderUUID: ru1[:],
							State:         ResponderState_WORKING,
							NextUpdateIn:  durationpb.New(10 * time.Millisecond),
						},
					},
				})
			}()
		}

		wg.Wait()

		expected = SourceQueryProgress{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})
}

func TestQueryProgressStalled(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tc := TestConnection{IgnoreNoResponders: true}
	sq, err := RunSourceQuery(ctx, &query, DefaultStartTimeout, &tc, devNull())
	if err != nil {
		t.Fatal(err)
	}

	ru1 := uuid.New()

	// Make sure that the details are correct initially
	var expected SourceQueryProgress

	t.Run("Processing the initial response", func(t *testing.T) {
		// Test the initial response
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	t.Run("After a responder has stalled", func(t *testing.T) {
		// Wait long enough for the thing to be marked as stalled
		time.Sleep(20 * time.Millisecond)

		expected = SourceQueryProgress{
			Working:    0,
			Stalled:    1,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}

		sq.respondersMu.Lock()
		defer sq.respondersMu.Unlock()
		if _, ok := sq.responders[ru1]; !ok {
			t.Error("Could not get responder for scope test1")
		}
	})

	t.Run("After a responder recovers from a stall", func(t *testing.T) {
		// See if it will un-stall itself
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_COMPLETE,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    0,
			Stalled:    0,
			Complete:   1,
			Error:      0,
			Responders: 1,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	if sq.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestRogueResponder(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	tc := TestConnection{IgnoreNoResponders: true}
	sq, err := RunSourceQuery(ctx, &query, 100*time.Millisecond, &tc, devNull())
	if err != nil {
		t.Fatal(err)
	}

	rur := uuid.New()

	// Create our rogue responder that doesn't cancel when it should
	ticker := time.NewTicker(5 * time.Second)
	tickerCtx, tickerCancel := context.WithCancel(context.Background())
	defer tickerCancel()
	defer ticker.Stop()

	go func() {
		// Send an initial response
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: rur[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(5 * time.Second),
				},
			},
		})

		// Now start ticking
		for {
			select {
			case <-ticker.C:
				sq.handleQueryResponse(ctx, &QueryResponse{
					ResponseType: &QueryResponse_Response{
						Response: &Response{
							Responder:     "test",
							ResponderUUID: rur[:],
							State:         ResponderState_WORKING,
							NextUpdateIn:  durationpb.New(5 * time.Second),
						},
					},
				})
			case <-tickerCtx.Done():
				return
			}
		}
	}()

	time.Sleep(300 * time.Millisecond)

	// Check that we've noticed the testRogue responder
	if sq.allDone() == true {
		t.Error("expected allDone() to be false")
	}

	cancel()

	time.Sleep(100 * time.Millisecond)

	// We expect that it has been marked as cancelled, regardless of what the
	// responder actually did
	expected := SourceQueryProgress{
		Working:    0,
		Stalled:    0,
		Complete:   0,
		Error:      0,
		Responders: 1,
		Cancelled:  1,
	}

	if err := expected.ExpectToMatch(sq); err != nil {
		t.Error(err)
	}
}

func TestQueryProgressError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tc := TestConnection{IgnoreNoResponders: true}
	sq, err := RunSourceQuery(ctx, &query, DefaultStartTimeout, &tc, devNull())
	if err != nil {
		t.Fatal(err)
	}

	ru1 := uuid.New()

	// Make sure that the details are correct initially
	var expected SourceQueryProgress

	t.Run("Processing the initial response", func(t *testing.T) {
		// Test the initial response
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_WORKING,
					NextUpdateIn:  durationpb.New(10 * time.Millisecond),
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    1,
			Stalled:    0,
			Complete:   0,
			Error:      0,
			Responders: 1,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	t.Run("After a responder has failed", func(t *testing.T) {
		sq.handleQueryResponse(ctx, &QueryResponse{
			ResponseType: &QueryResponse_Response{
				Response: &Response{
					Responder:     "test",
					ResponderUUID: ru1[:],
					State:         ResponderState_ERROR,
				},
			},
		})

		expected = SourceQueryProgress{
			Working:    0,
			Stalled:    0,
			Complete:   0,
			Error:      1,
			Responders: 1,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	t.Run("Ensuring that a failed responder does not get marked as stalled", func(t *testing.T) {
		time.Sleep(12 * time.Millisecond)

		expected = SourceQueryProgress{
			Working:    0,
			Stalled:    0,
			Complete:   0,
			Error:      1,
			Responders: 1,
		}

		if err := expected.ExpectToMatch(sq); err != nil {
			t.Error(err)
		}
	})

	if sq.allDone() == false {
		t.Error("expected allDone() to be true")
	}
}

func TestStart(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tc := TestConnection{IgnoreNoResponders: true}

	responses := make(chan *QueryResponse, 128)
	// this emulates a source
	sourceHit := atomic.Bool{}

	_, err := tc.Subscribe(fmt.Sprintf("request.scope.%v", query.GetScope()), func(msg *nats.Msg) {
		sourceHit.Store(true)
		response := QueryResponse{
			ResponseType: &QueryResponse_NewItem{
				NewItem: &item,
			},
		}
		// Test that the handlers work
		err := tc.Publish(ctx, query.Subject(), &response)
		if err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = RunSourceQuery(ctx, &query, DefaultStartTimeout, &tc, responses)
	if err != nil {
		t.Fatal(err)
	}

	response := <-responses

	tc.MessagesMu.Lock()
	if len(tc.Messages) != 2 {
		t.Errorf("expected 2 messages to be sent, got %v", len(tc.Messages))
	}
	tc.MessagesMu.Unlock()

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

func TestExecute(t *testing.T) {
	t.Parallel()

	t.Run("with no responders", func(t *testing.T) {
		conn := TestConnection{}
		_, err := conn.Subscribe("request.scope.global", func(msg *nats.Msg) {})
		if err != nil {
			t.Fatal(err)
		}
		u := uuid.New()
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

		_, _, _, err = RunSourceQuerySync(t.Context(), &q, 100*time.Millisecond, &conn)

		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with a full response set", func(t *testing.T) {
		conn := TestConnection{}
		_, err := conn.Subscribe("request.scope.global", func(msg *nats.Msg) {})
		if err != nil {
			t.Fatal(err)
		}
		u := uuid.New()
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

		querySent := make(chan struct{})
		done := make(chan struct{})

		go func() {
			defer close(done)
			// wait for the query to be sent
			<-querySent

			ru1 := uuid.New()

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

			err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{
				ResponseType: &QueryResponse_NewItem{
					NewItem: &item,
				},
			})
			if err != nil {
				t.Error(err)
			}

			err = conn.Publish(context.Background(), q.Subject(), &QueryResponse{
				ResponseType: &QueryResponse_NewItem{
					NewItem: &item,
				},
			})
			if err != nil {
				t.Error(err)
			}

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

		responseChan := make(chan *QueryResponse)
		// items, _, errs, err := RunSourceQuerySync(t.Context(), &q, DefaultStartTimeout, &conn)
		_, err = RunSourceQuery(t.Context(), &q, DefaultStartTimeout, &conn, responseChan)
		if err != nil {
			t.Fatal(err)
		}

		close(querySent)

		items := []*Item{}
		errs := []*QueryError{}

		for r := range responseChan {
			if r == nil {
				t.Fatal("expected a response")
			}
			switch r.GetResponseType().(type) {
			case *QueryResponse_NewItem:
				items = append(items, r.GetNewItem())
			case *QueryResponse_Error:
				errs = append(errs, r.GetError())
			default:
				t.Errorf("unexpected response type: %T", r.GetResponseType())
			}
		}

		<-done

		if len(items) != 2 {
			t.Errorf("expected 2 items got %v: %v", len(items), items)
		}

		if len(errs) != 0 {
			t.Errorf("expected 0 errors got %v: %v", len(errs), errs)
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

	_, err = RunSourceQuery(t.Context(), &q, DefaultStartTimeout, &enc, slowChan)
	if err != nil {
		t.Fatal(err)
	}

	for i := range slowChan {
		time.Sleep(100 * time.Millisecond)

		t.Log(i)
	}
}

func TestFastFinisher(t *testing.T) {
	t.Parallel()

	// Test for a situation where there is one responder that finishes really
	// quickly and results in the other responders not getting a chance to start
	conn := TestConnection{}

	fast := uuid.New()
	slow := uuid.New()

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

	items, _, errs, err := RunSourceQuerySync(t.Context(), newQuery(), 500*time.Millisecond, &conn)

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

// This source will simply respond to any query that it sent with a configured
// number of items, and configurable delays. This is designed to replicate a
// real system at scale
type SimpleSource struct {
	// How many items to return from the query
	NumItemsReturn int

	// How long to wait before starting work on the query
	StartDelay time.Duration

	// How long to wait before sending each item
	PerItemDelay time.Duration

	// How long to wait before sending the completion message
	CompletionDelay time.Duration

	// The connection to use
	Conn *TestConnection

	// The probability of stalling where 0 is no stall and 1 is always stall
	StallProbability float64

	// The probability of failing where 0 is no fail and 1 is always fail
	FailProbability float64

	// The responder name to use
	ResponderName string
}

func (s *SimpleSource) Start(ctx context.Context, t *testing.T) {
	// ignore errors from test connection
	_, _ = s.Conn.Subscribe("request.>", func(msg *nats.Msg) {
		// Run these in parallel
		go func(msg *nats.Msg) {
			query := &Query{}

			err := Unmarshal(ctx, msg.Data, query)
			if err != nil {
				panic(fmt.Errorf("Unmarshal(%v): %w", query, err))
			}

			// Create the number of items that were requested
			items := make([]*Item, s.NumItemsReturn)
			for i := range s.NumItemsReturn {
				items[i] = newItem()
			}

			// Make a UUID for yourself
			responderUUID := uuid.New()

			// Wait for the start delay
			time.Sleep(s.StartDelay)

			// Calculate the expected duration of the query
			expectedQueryDuration := (s.PerItemDelay * time.Duration(s.NumItemsReturn)) + s.CompletionDelay + 500*time.Millisecond

			err = s.Conn.Publish(ctx, query.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
				Responder:     s.ResponderName,
				ResponderUUID: responderUUID[:],
				State:         ResponderState_WORKING,
				NextUpdateIn:  durationpb.New(expectedQueryDuration),
				UUID:          query.GetUUID(),
			}}})
			if err != nil {
				t.Errorf("error publishing response: %v", err)
			}

			for _, item := range items {
				time.Sleep(s.PerItemDelay)
				err = s.Conn.Publish(ctx, query.Subject(), &QueryResponse{ResponseType: &QueryResponse_NewItem{NewItem: item}})
				if err != nil {
					t.Errorf("error publishing item: %v", err)
				}
			}

			// Stall with a certain probability
			if rand.Float64() < s.StallProbability {
				return
			}

			// Fail with a certain probability
			if rand.Float64() < s.FailProbability {
				err = s.Conn.Publish(ctx, query.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
					Responder:     s.ResponderName,
					ResponderUUID: responderUUID[:],
					State:         ResponderState_ERROR,
					UUID:          query.GetUUID(),
				}}})
				if err != nil {
					t.Errorf("error publishing response: %v", err)
				}
				return
			}

			time.Sleep(s.CompletionDelay)
			err = s.Conn.Publish(ctx, query.Subject(), &QueryResponse{ResponseType: &QueryResponse_Response{Response: &Response{
				Responder:     s.ResponderName,
				ResponderUUID: responderUUID[:],
				State:         ResponderState_COMPLETE,
				UUID:          query.GetUUID(),
			}}})
			if err != nil {
				t.Errorf("error publishing response: %v", err)
			}
		}(msg)
	})
}

func TestMassiveScale(t *testing.T) {
	t.Parallel()

	if _, exists := os.LookupEnv("GITHUB_ACTIONS"); exists {
		// Note that in these tests we can push things even further, to 10,000
		// sources for example. The problem is that once the CPU is context
		// switching too heavily you end up in a position where the sources
		// start getting marked as stalled as they don't have enough CPU to send
		// their messages quickly enough and they blow through their expected
		// timeout.
		//
		// They can also fail locally when using -race as this puts a lot more
		// load on the CPU than there would normally be
		t.Skip("These tests are too flaky due to reliance on wall clock time and fast timings")
	}

	tests := []struct {
		// The number of sources to create
		NumSources int
		// The maximum time to wait before starting
		MaxStartDelayMilliseconds int
		// The maximum time to wait between items
		MaxPerItemDelayMilliseconds int
		// The maximum time to wait before completion
		MaxCompletionDelayMilliseconds int
		// The maximum number of items to return
		MaxItemsToReturn int
		// The probability of a source stalling where 0 is no stall and 1 is
		// always stall
		StallProbability float64
		// The probability of a source failing where 0 is no fail and 1 is
		// always fail
		FailProbability float64
		// How long to give sources to start responding, over and above the
		// maxStartDelayMilliseconds
		StartDelayGracePeriodMilliseconds int
	}{
		{
			NumSources:                        100,
			MaxStartDelayMilliseconds:         100,
			MaxPerItemDelayMilliseconds:       10,
			MaxCompletionDelayMilliseconds:    100,
			MaxItemsToReturn:                  100,
			StallProbability:                  0.0,
			FailProbability:                   0.0,
			StartDelayGracePeriodMilliseconds: 100,
		},
		{
			NumSources:                        1_000,
			MaxStartDelayMilliseconds:         100,
			MaxPerItemDelayMilliseconds:       10,
			MaxCompletionDelayMilliseconds:    100,
			MaxItemsToReturn:                  100,
			StallProbability:                  0.0,
			FailProbability:                   0.0,
			StartDelayGracePeriodMilliseconds: 100,
		},
		{
			NumSources:                        100,
			MaxStartDelayMilliseconds:         100,
			MaxPerItemDelayMilliseconds:       10,
			MaxCompletionDelayMilliseconds:    100,
			MaxItemsToReturn:                  100,
			StallProbability:                  0.3,
			FailProbability:                   0.0,
			StartDelayGracePeriodMilliseconds: 100,
		},
		{
			NumSources:                        1_000,
			MaxStartDelayMilliseconds:         100,
			MaxPerItemDelayMilliseconds:       10,
			MaxCompletionDelayMilliseconds:    100,
			MaxItemsToReturn:                  100,
			StallProbability:                  0.3,
			FailProbability:                   0.0,
			StartDelayGracePeriodMilliseconds: 100,
		},
		{
			NumSources:                        100,
			MaxStartDelayMilliseconds:         100,
			MaxPerItemDelayMilliseconds:       10,
			MaxCompletionDelayMilliseconds:    100,
			MaxItemsToReturn:                  100,
			StallProbability:                  0.3,
			FailProbability:                   0.3,
			StartDelayGracePeriodMilliseconds: 100,
		},
		{
			NumSources:                        1_000,
			MaxStartDelayMilliseconds:         100,
			MaxPerItemDelayMilliseconds:       10,
			MaxCompletionDelayMilliseconds:    100,
			MaxItemsToReturn:                  100,
			StallProbability:                  0.3,
			FailProbability:                   0.3,
			StartDelayGracePeriodMilliseconds: 100,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("NumSources %v, MaxStartDelay %v, MaxPerItemDelay %v, MaxCompletionDelay %v, MaxItemsToReturn %v, StallProbability %v, FailProbability %v, StartDelayGracePeriod %v",
			test.NumSources,
			test.MaxStartDelayMilliseconds,
			test.MaxPerItemDelayMilliseconds,
			test.MaxCompletionDelayMilliseconds,
			test.MaxItemsToReturn,
			test.StallProbability,
			test.FailProbability,
			test.StartDelayGracePeriodMilliseconds,
		), func(t *testing.T) {
			tConn := TestConnection{}

			// Generate a random duration between 0 and maxDuration
			randomDuration := func(maxDuration int) time.Duration {
				return time.Duration(rand.Intn(maxDuration)) * time.Millisecond
			}

			expectedItems := 0

			// Start all the sources
			sources := make([]*SimpleSource, test.NumSources)
			for i := range sources {
				numItemsReturn := rand.Intn(test.MaxItemsToReturn)
				expectedItems += numItemsReturn // Count how many items we expect to receive
				startDelay := randomDuration(test.MaxStartDelayMilliseconds)
				perItemDelay := randomDuration(test.MaxPerItemDelayMilliseconds)
				completionDelay := randomDuration(test.MaxCompletionDelayMilliseconds)

				sources[i] = &SimpleSource{
					NumItemsReturn:   numItemsReturn,
					StartDelay:       startDelay,
					PerItemDelay:     perItemDelay,
					CompletionDelay:  completionDelay,
					StallProbability: test.StallProbability,
					FailProbability:  test.FailProbability,
					Conn:             &tConn,
					ResponderName: fmt.Sprintf("NumItems %v, StartDelay %v, PerItemDelay %v CompletionDelay %v",
						numItemsReturn,
						startDelay.String(),
						perItemDelay.String(),
						completionDelay.String(),
					),
				}

				sources[i].Start(context.Background(), t)
			}

			// Create the query
			u := uuid.New()
			q := Query{
				Type:     "massive-scale-test",
				Method:   QueryMethod_GET,
				Query:    "GO!!!!!",
				Scope:    "test",
				UUID:     u[:],
				Deadline: timestamppb.New(time.Now().Add(60 * time.Second)),
			}

			responseChan := make(chan *QueryResponse)
			doneChan := make(chan struct{})

			// Begin handling the responses
			actualItems := 0
			go func() {
				for {
					select {
					case <-t.Context().Done():
						return
					case response, ok := <-responseChan:
						if !ok {
							// Channel closed
							close(doneChan)
							return
						}

						switch response.GetResponseType().(type) {
						case *QueryResponse_NewItem:
							actualItems++
						}
					}
				}
			}()

			// Start the query
			startTimeout := time.Duration(test.MaxStartDelayMilliseconds+test.StartDelayGracePeriodMilliseconds) * time.Millisecond
			qp, err := RunSourceQuery(t.Context(), &q, startTimeout, &tConn, responseChan)
			if err != nil {
				t.Fatal(err)
			}

			// Wait for the query to finish
			<-doneChan

			if actualItems != expectedItems {
				t.Errorf("Expected %v items, got %v", expectedItems, actualItems)
			}

			progress := qp.Progress()

			if progress.Responders != test.NumSources {
				t.Errorf("Expected %v responders, got %v", test.NumSources, progress.Responders)
			}

			fmt.Printf("Num Complete: %v\n", progress.Complete)
			fmt.Printf("Num Working: %v\n", progress.Working)
			fmt.Printf("Num Stalled: %v\n", progress.Stalled)
			fmt.Printf("Num Error: %v\n", progress.Error)
			fmt.Printf("Num Cancelled: %v\n", progress.Cancelled)
			fmt.Printf("Num Responders: %v\n", progress.Responders)
			fmt.Printf("Num Items: %v\n", actualItems)
		})
	}
}
