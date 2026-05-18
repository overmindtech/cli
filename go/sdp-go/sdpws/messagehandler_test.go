package sdpws

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/overmindtech/cli/go/sdp-go"
)

const benchEventCount = 1000

// doneStatus returns a GatewayRequestStatus whose Done() returns true.
func doneStatus() *sdp.GatewayRequestStatus {
	return &sdp.GatewayRequestStatus{
		Summary: &sdp.GatewayRequestStatus_Summary{
			Working:    0,
			Complete:   1,
			Responders: 1,
		},
		PostProcessingComplete: true,
	}
}

// notDoneStatus returns a GatewayRequestStatus whose Done() returns false.
func notDoneStatus() *sdp.GatewayRequestStatus {
	return &sdp.GatewayRequestStatus{
		Summary: &sdp.GatewayRequestStatus_Summary{
			Working:    1,
			Complete:   0,
			Responders: 1,
		},
		PostProcessingComplete: false,
	}
}

func feedItemsAndEdges(h GatewayMessageHandler, n int) {
	ctx := context.Background()
	for range n {
		h.NewItem(ctx, &sdp.Item{Type: "test", UniqueAttribute: "name"})
		h.NewEdge(ctx, &sdp.Edge{})
	}
}

// TestStoreItemsOnlyHandler asserts items accumulate and edges are dropped.
func TestStoreItemsOnlyHandler(t *testing.T) {
	h := &StoreItemsOnlyHandler{}
	feedItemsAndEdges(h, benchEventCount)

	if got := len(h.Items); got != benchEventCount {
		t.Errorf("expected %d items retained, got %d", benchEventCount, got)
	}
	// StoreItemsOnlyHandler has no Edges field by design — the absence is
	// enforced by the type system. The test exists to lock in the
	// no-edges-stored invariant for future readers.
}

// TestWaitForAllQueriesHandler_NoStorage asserts the safe default does not
// retain items or edges. The absence of the slices is enforced by the type
// system (no fields named Items/Edges); this test verifies the runtime path
// is also free of side effects and that the DoneCallback semantics are
// preserved.
func TestWaitForAllQueriesHandler_NoStorage(t *testing.T) {
	var calls atomic.Int32
	h := &WaitForAllQueriesHandler{DoneCallback: func() { calls.Add(1) }}

	feedItemsAndEdges(h, benchEventCount)

	// Status with Done() == false: no callback.
	h.Status(context.Background(), notDoneStatus())
	if got := calls.Load(); got != 0 {
		t.Errorf("DoneCallback fired %d times on non-done status; expected 0", got)
	}

	// Status with Done() == true: exactly one callback.
	h.Status(context.Background(), doneStatus())
	if got := calls.Load(); got != 1 {
		t.Errorf("DoneCallback fired %d times on done status; expected 1", got)
	}
}

func TestWaitForAllQueriesItemsOnlyHandler(t *testing.T) {
	var calls atomic.Int32
	h := &WaitForAllQueriesItemsOnlyHandler{DoneCallback: func() { calls.Add(1) }}

	feedItemsAndEdges(h, benchEventCount)

	if got := len(h.Items); got != benchEventCount {
		t.Errorf("expected %d items retained, got %d", benchEventCount, got)
	}

	h.Status(context.Background(), notDoneStatus())
	if got := calls.Load(); got != 0 {
		t.Errorf("DoneCallback fired %d times on non-done status; expected 0", got)
	}

	h.Status(context.Background(), doneStatus())
	if got := calls.Load(); got != 1 {
		t.Errorf("DoneCallback fired %d times on done status; expected 1", got)
	}
}

func TestWaitForAllQueriesStoreEverythingHandler(t *testing.T) {
	var calls atomic.Int32
	h := &WaitForAllQueriesStoreEverythingHandler{DoneCallback: func() { calls.Add(1) }}

	feedItemsAndEdges(h, benchEventCount)

	if got := len(h.Items); got != benchEventCount {
		t.Errorf("expected %d items retained, got %d", benchEventCount, got)
	}
	if got := len(h.Edges); got != benchEventCount {
		t.Errorf("expected %d edges retained, got %d", benchEventCount, got)
	}

	h.Status(context.Background(), notDoneStatus())
	if got := calls.Load(); got != 0 {
		t.Errorf("DoneCallback fired %d times on non-done status; expected 0", got)
	}

	h.Status(context.Background(), doneStatus())
	if got := calls.Load(); got != 1 {
		t.Errorf("DoneCallback fired %d times on done status; expected 1", got)
	}
}

// TestStoreEverythingHandler asserts the pre-existing handler still works
// post-rename. It's not new code, but we test it here to keep the four wait
// variants symmetric and guard against accidental future drift.
func TestStoreEverythingHandler(t *testing.T) {
	h := &StoreEverythingHandler{}
	feedItemsAndEdges(h, benchEventCount)

	if got := len(h.Items); got != benchEventCount {
		t.Errorf("expected %d items retained, got %d", benchEventCount, got)
	}
	if got := len(h.Edges); got != benchEventCount {
		t.Errorf("expected %d edges retained, got %d", benchEventCount, got)
	}
}
