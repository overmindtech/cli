package discovery

import (
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

type TestConnection struct {
	ReturnStatus nats.Status
	ReturnStats  nats.Statistics
	ReturnError  error
	Mutex        sync.Mutex
}

func (t *TestConnection) Status() nats.Status {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.ReturnStatus
}

func (t *TestConnection) Stats() nats.Statistics {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.ReturnStats
}

func (t *TestConnection) LastError() error {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.ReturnError
}

func TestNATSWatcher(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTING,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	fail := make(chan bool)

	w := NATSWatcher{
		Connection: &c,
		FailureHandler: func() {
			fail <- true
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	time.Sleep(interval * 2)

	c.Mutex.Lock()
	c.ReturnStatus = nats.CONNECTED
	c.Mutex.Unlock()

	time.Sleep(interval * 2)

	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	time.Sleep(interval * 2)

	c.Mutex.Lock()
	c.ReturnStatus = nats.CONNECTED
	c.Mutex.Unlock()

	time.Sleep(interval * 2)

	c.Mutex.Lock()
	c.ReturnStatus = nats.CLOSED
	c.Mutex.Unlock()

	select {
	case <-time.After(interval * 2):
		t.Errorf("FailureHandler not called in %v", (interval * 2).String())
	case <-fail:
		// The fail handler has been called!
		t.Log("Fail handler called successfully 🥳")
	}
}

func TestFailureHandler(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTING,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	var w *NATSWatcher
	done := make(chan bool, 1024)

	w = &NATSWatcher{
		Connection: &c,
		FailureHandler: func() {
			go w.Stop()
			done <- true
		},
	}

	interval := 100 * time.Millisecond

	w.Start(interval)

	time.Sleep(interval * 2)

	c.Mutex.Lock()
	c.ReturnStatus = nats.CLOSED
	c.Mutex.Unlock()

	time.Sleep(interval * 2)

	select {
	case <-time.After(interval * 2):
		t.Errorf("FailureHandler not completed in %v", (interval * 2).String())
	case <-done:
		if len(done) != 0 {
			t.Errorf("Handler was called more than once")
		}
		// The fail handler has been called!
		t.Log("Fail handler called successfully 🥳")
	}
}

func TestReconnectionTimeout(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	fail := make(chan bool)

	w := NATSWatcher{
		Connection: &c,
		// Set a short timeout for testing
		ReconnectionTimeout: 100 * time.Millisecond,
		FailureHandler: func() {
			fail <- true
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	// Start connected
	time.Sleep(interval * 2)

	// Transition to RECONNECTING state
	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	// Wait for the timeout to trigger (100ms + some buffer)
	select {
	case <-time.After(200 * time.Millisecond):
		t.Error("FailureHandler not called after reconnection timeout")
	case <-fail:
		t.Log("Fail handler called successfully after reconnection timeout 🥳")
	}

	w.Stop()
}

func TestReconnectionTimeoutNotTriggeredWhenConnected(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	fail := make(chan bool)

	w := NATSWatcher{
		Connection: &c,
		// Set a short timeout for testing
		ReconnectionTimeout: 50 * time.Millisecond,
		FailureHandler: func() {
			fail <- true
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	// Briefly go to RECONNECTING state
	time.Sleep(interval * 2)
	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	// But reconnect before timeout
	time.Sleep(20 * time.Millisecond)
	c.Mutex.Lock()
	c.ReturnStatus = nats.CONNECTED
	c.Mutex.Unlock()

	// Wait longer than the timeout to ensure it doesn't trigger
	select {
	case <-time.After(100 * time.Millisecond):
		t.Log("Timeout not triggered as expected when connection recovered 🥳")
	case <-fail:
		t.Error("FailureHandler should not be called when connection recovers before timeout")
	}

	w.Stop()
}

func TestReconnectionTimeoutDisabled(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	fail := make(chan bool)

	w := NATSWatcher{
		Connection: &c,
		// No timeout set (0 means disabled)
		ReconnectionTimeout: 0,
		FailureHandler: func() {
			fail <- true
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	// Transition to RECONNECTING state
	time.Sleep(interval * 2)
	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	// Wait for a while - should not trigger failure handler
	select {
	case <-time.After(100 * time.Millisecond):
		t.Log("Timeout correctly disabled, failure handler not called 🥳")
	case <-fail:
		t.Error("FailureHandler should not be called when timeout is disabled")
	}

	w.Stop()
}

func TestFailureHandlerNotCalledRepeatedly(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	failCount := 0
	var mu sync.Mutex

	w := NATSWatcher{
		Connection: &c,
		// Set a short timeout for testing
		ReconnectionTimeout: 50 * time.Millisecond,
		FailureHandler: func() {
			mu.Lock()
			failCount++
			mu.Unlock()
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	// Transition to RECONNECTING state
	time.Sleep(interval * 2)
	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	// Wait for timeout to trigger (50ms timeout + buffer)
	time.Sleep(80 * time.Millisecond)

	// Give it more time to ensure handler isn't called again
	time.Sleep(50 * time.Millisecond)

	w.Stop()

	mu.Lock()
	count := failCount
	mu.Unlock()

	if count != 1 {
		t.Errorf("FailureHandler should be called exactly once, but was called %d times", count)
	} else {
		t.Log("Failure handler called exactly once as expected 🥳")
	}
}

func TestStartWithNilConnection(t *testing.T) {
	w := NATSWatcher{
		Connection: nil,
		FailureHandler: func() {
			t.Error("FailureHandler should not be called when connection is nil")
		},
	}

	// Should not panic and should return early
	w.Start(10 * time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	// If we get here without panicking, the test passes
	t.Log("Start with nil connection handled gracefully 🥳")
}

func TestStartWithNilWatcher(t *testing.T) {
	var w *NATSWatcher

	// Should not panic
	w.Start(10 * time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	// If we get here without panicking, the test passes
	t.Log("Start with nil watcher handled gracefully 🥳")
}

func TestReconnectionTimeoutWithConnectingState(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	fail := make(chan bool)

	w := NATSWatcher{
		Connection: &c,
		// Set a short timeout for testing
		ReconnectionTimeout: 100 * time.Millisecond,
		FailureHandler: func() {
			fail <- true
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	// Start connected
	time.Sleep(interval * 2)

	// Transition to CONNECTING state (not just RECONNECTING)
	c.Mutex.Lock()
	c.ReturnStatus = nats.CONNECTING
	c.Mutex.Unlock()

	// Wait for the timeout to trigger (100ms + some buffer)
	select {
	case <-time.After(200 * time.Millisecond):
		t.Error("FailureHandler not called after reconnection timeout with CONNECTING state")
	case <-fail:
		t.Log("Fail handler called successfully after reconnection timeout with CONNECTING state 🥳")
	}

	w.Stop()
}

func TestMultipleDisconnectionCycles(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	failCount := 0
	var mu sync.Mutex

	w := NATSWatcher{
		Connection: &c,
		// Set a short timeout for testing
		ReconnectionTimeout: 50 * time.Millisecond,
		FailureHandler: func() {
			mu.Lock()
			failCount++
			mu.Unlock()
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	// First disconnection cycle
	time.Sleep(interval * 2)
	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	// Wait for timeout to trigger
	time.Sleep(80 * time.Millisecond)

	// Reconnect
	c.Mutex.Lock()
	c.ReturnStatus = nats.CONNECTED
	c.Mutex.Unlock()
	time.Sleep(interval * 2)

	// Second disconnection cycle - should reset and allow handler to be called again
	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	// Wait for timeout to trigger again
	time.Sleep(80 * time.Millisecond)

	w.Stop()

	mu.Lock()
	count := failCount
	mu.Unlock()

	if count != 2 {
		t.Errorf("FailureHandler should be called twice (once per disconnection cycle), but was called %d times", count)
	} else {
		t.Log("Failure handler called correctly for multiple disconnection cycles 🥳")
	}
}

func TestStopBeforeStart(t *testing.T) {
	w := NATSWatcher{
		Connection: &TestConnection{
			ReturnStatus: nats.CONNECTED,
		},
	}

	// Should not panic if Stop is called before Start
	w.Stop()
	t.Log("Stop before Start handled gracefully 🥳")
}

func TestStopMultipleTimes(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	w := NATSWatcher{
		Connection:     &c,
		FailureHandler: func() {},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)
	time.Sleep(interval * 2)

	// Stop multiple times should not panic
	w.Stop()
	w.Stop()
	w.Stop()

	t.Log("Multiple Stop calls handled gracefully 🥳")
}

func TestHandlerResetAfterReconnection(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	failCount := 0
	var mu sync.Mutex

	w := NATSWatcher{
		Connection: &c,
		// Set a short timeout for testing
		ReconnectionTimeout: 50 * time.Millisecond,
		FailureHandler: func() {
			mu.Lock()
			failCount++
			mu.Unlock()
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	// First disconnection - trigger timeout
	time.Sleep(interval * 2)
	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	// Wait for timeout
	time.Sleep(80 * time.Millisecond)

	// Reconnect - this should reset the tracking
	c.Mutex.Lock()
	c.ReturnStatus = nats.CONNECTED
	c.Mutex.Unlock()
	time.Sleep(interval * 2)

	// Disconnect again - should be able to trigger handler again
	c.Mutex.Lock()
	c.ReturnStatus = nats.RECONNECTING
	c.Mutex.Unlock()

	// Wait for timeout again
	time.Sleep(80 * time.Millisecond)

	w.Stop()

	mu.Lock()
	count := failCount
	mu.Unlock()

	if count != 2 {
		t.Errorf("FailureHandler should be called twice after reconnection reset, but was called %d times", count)
	} else {
		t.Log("Handler reset correctly after reconnection 🥳")
	}
}

func TestCLOSEDStatusTriggersImmediately(t *testing.T) {
	c := TestConnection{
		ReturnStatus: nats.CONNECTED,
		ReturnStats:  nats.Statistics{},
		ReturnError:  nil,
	}

	fail := make(chan bool)

	w := NATSWatcher{
		Connection: &c,
		// Even with timeout set, CLOSED should trigger immediately
		ReconnectionTimeout: 100 * time.Millisecond,
		FailureHandler: func() {
			fail <- true
		},
	}

	interval := 10 * time.Millisecond

	w.Start(interval)

	// Start connected
	time.Sleep(interval * 2)

	// Transition directly to CLOSED (should trigger immediately, not wait for timeout)
	c.Mutex.Lock()
	c.ReturnStatus = nats.CLOSED
	c.Mutex.Unlock()

	// Should trigger much faster than the timeout
	select {
	case <-time.After(50 * time.Millisecond):
		t.Error("FailureHandler not called immediately for CLOSED status")
	case <-fail:
		t.Log("Fail handler called immediately for CLOSED status 🥳")
	}

	w.Stop()
}
