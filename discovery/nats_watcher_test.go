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
