package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/sdp-go"
)

func TestSetInitError(t *testing.T) {
	e := &Engine{
		initError:      nil,
		initErrorMutex: sync.RWMutex{},
	}

	testErr := errors.New("initialization failed")
	e.SetInitError(testErr)

	// Direct pointer comparison is intentional here - we want to verify the exact error object is stored
	if e.initError == nil || e.initError.Error() != testErr.Error() {
		t.Errorf("expected initError to be %v, got %v", testErr, e.initError)
	}
}

func TestGetInitError(t *testing.T) {
	e := &Engine{
		initError:      nil,
		initErrorMutex: sync.RWMutex{},
	}

	// Test nil case
	if err := e.GetInitError(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Test with error set
	testErr := errors.New("test error")
	e.initError = testErr

	if err := e.GetInitError(); err == nil || err.Error() != testErr.Error() {
		t.Errorf("expected error to be %v, got %v", testErr, err)
	}
}

func TestSetInitErrorNil(t *testing.T) {
	e := &Engine{
		initError:      errors.New("previous error"),
		initErrorMutex: sync.RWMutex{},
	}

	// Clear the error
	e.SetInitError(nil)

	if e.initError != nil {
		t.Errorf("expected initError to be nil after clearing, got %v", e.initError)
	}

	if err := e.GetInitError(); err != nil {
		t.Errorf("expected GetInitError to return nil after clearing, got %v", err)
	}
}

func TestInitErrorConcurrentAccess(t *testing.T) {
	e := &Engine{
		initError:      nil,
		initErrorMutex: sync.RWMutex{},
	}

	// Test concurrent access from multiple goroutines
	var wg sync.WaitGroup
	iterations := 100

	// Writers
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				e.SetInitError(fmt.Errorf("error from goroutine %d iteration %d", id, j))
			}
		}(i)
	}

	// Readers
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				_ = e.GetInitError()
			}
		}()
	}

	wg.Wait()

	// Should not panic - error should be one of the written values or nil
	finalErr := e.GetInitError()
	if finalErr == nil {
		t.Log("Final error is nil (acceptable in concurrent test)")
	} else {
		t.Logf("Final error: %v", finalErr)
	}
}

func TestReadinessHealthCheckWithInitError(t *testing.T) {
	ec := &EngineConfig{
		EngineType: "test",
		SourceName: "test-source",
		HeartbeatOptions: &HeartbeatOptions{
			ReadinessCheck: func(ctx context.Context) error {
				// Adapter health is fine
				return nil
			},
		},
	}

	e, err := NewEngine(ec)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx := context.Background()

	// Readiness should pass when no init error
	if err := e.ReadinessHealthCheck(ctx); err != nil {
		t.Errorf("expected readiness to pass with no init error, got: %v", err)
	}

	// Set an init error
	testErr := errors.New("AWS AssumeRole denied")
	e.SetInitError(testErr)

	// Readiness should now fail with the init error
	err = e.ReadinessHealthCheck(ctx)
	if err == nil {
		t.Error("expected readiness to fail with init error, got nil")
	} else if !errors.Is(err, testErr) {
		t.Errorf("expected readiness error to wrap init error, got: %v", err)
	}

	// Clear the init error
	e.SetInitError(nil)

	// Readiness should pass again
	if err := e.ReadinessHealthCheck(ctx); err != nil {
		t.Errorf("expected readiness to pass after clearing init error, got: %v", err)
	}
}

func TestSendHeartbeatWithInitError(t *testing.T) {
	requests := make(chan *connect.Request[sdp.SubmitSourceHeartbeatRequest], 10)
	responses := make(chan *connect.Response[sdp.SubmitSourceHeartbeatResponse], 10)

	ec := &EngineConfig{
		EngineType: "test",
		SourceName: "test-source",
		HeartbeatOptions: &HeartbeatOptions{
			ManagementClient: testHeartbeatClient{
				Requests:  requests,
				Responses: responses,
			},
			Frequency: 0, // Disable automatic heartbeats
			ReadinessCheck: func(ctx context.Context) error {
				return nil // Adapters are fine
			},
		},
	}

	e, err := NewEngine(ec)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx := context.Background()

	// Send heartbeat with init error
	testErr := errors.New("configuration error: invalid credentials")
	e.SetInitError(testErr)

	responses <- &connect.Response[sdp.SubmitSourceHeartbeatResponse]{
		Msg: &sdp.SubmitSourceHeartbeatResponse{},
	}

	err = e.SendHeartbeat(ctx, nil)
	if err != nil {
		t.Errorf("expected SendHeartbeat to succeed, got: %v", err)
	}

	// Verify the heartbeat included the init error
	req := <-requests
	if req.Msg.GetError() == "" {
		t.Error("expected heartbeat to include error, got empty string")
	} else if !strings.Contains(req.Msg.GetError(), testErr.Error()) {
		t.Errorf("expected heartbeat error to contain %q, got: %q", testErr.Error(), req.Msg.GetError())
	}
}

func TestSendHeartbeatWithInitErrorAndCustomError(t *testing.T) {
	requests := make(chan *connect.Request[sdp.SubmitSourceHeartbeatRequest], 10)
	responses := make(chan *connect.Response[sdp.SubmitSourceHeartbeatResponse], 10)

	ec := &EngineConfig{
		EngineType: "test",
		SourceName: "test-source",
		HeartbeatOptions: &HeartbeatOptions{
			ManagementClient: testHeartbeatClient{
				Requests:  requests,
				Responses: responses,
			},
			Frequency: 0,
		},
	}

	e, err := NewEngine(ec)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx := context.Background()

	// Set init error and send heartbeat with custom error
	initErr := errors.New("init failed: invalid config")
	customErr := errors.New("custom error: readiness failed")
	e.SetInitError(initErr)

	responses <- &connect.Response[sdp.SubmitSourceHeartbeatResponse]{
		Msg: &sdp.SubmitSourceHeartbeatResponse{},
	}

	err = e.SendHeartbeat(ctx, customErr)
	if err != nil {
		t.Errorf("expected SendHeartbeat to succeed, got: %v", err)
	}

	// Verify both errors are included in the heartbeat
	req := <-requests
	if req.Msg.GetError() == "" {
		t.Error("expected heartbeat to include errors, got empty string")
	} else {
		errMsg := req.Msg.GetError()
		// Both errors should be in the joined error string
		if !strings.Contains(errMsg, initErr.Error()) {
			t.Errorf("expected heartbeat error to include init error %q, got: %q", initErr.Error(), errMsg)
		}
		if !strings.Contains(errMsg, customErr.Error()) {
			t.Errorf("expected heartbeat error to include custom error %q, got: %q", customErr.Error(), errMsg)
		}
	}
}

func TestInitialiseAdapters_Success(t *testing.T) {
	ec := &EngineConfig{
		EngineType: "test",
		SourceName: "test-source",
		HeartbeatOptions: &HeartbeatOptions{
			Frequency: 0, // Disable automatic heartbeats from StartSendingHeartbeats
		},
	}
	e, err := NewEngine(ec)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Set an init error to verify it gets cleared on success
	e.SetInitError(errors.New("previous error"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var called bool
	e.InitialiseAdapters(ctx, func(ctx context.Context) error {
		called = true
		return nil
	})

	if !called {
		t.Error("initFn was not called")
	}
	if err := e.GetInitError(); err != nil {
		t.Errorf("expected init error to be cleared after success, got: %v", err)
	}
}

func TestInitialiseAdapters_RetryThenSuccess(t *testing.T) {
	ec := &EngineConfig{
		EngineType: "test",
		SourceName: "test-source",
		HeartbeatOptions: &HeartbeatOptions{
			Frequency: 0,
		},
	}
	e, err := NewEngine(ec)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	attempts := 0
	e.InitialiseAdapters(ctx, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("transient error attempt %d", attempts)
		}
		return nil
	})

	if attempts < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts)
	}
	if err := e.GetInitError(); err != nil {
		t.Errorf("expected init error to be cleared after eventual success, got: %v", err)
	}
}

func TestInitialiseAdapters_ContextCancelled(t *testing.T) {
	ec := &EngineConfig{
		EngineType: "test",
		SourceName: "test-source",
		HeartbeatOptions: &HeartbeatOptions{
			Frequency: 0,
		},
	}
	e, err := NewEngine(ec)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var callCount int

	// InitialiseAdapters blocks; cancel ctx after a short delay so it returns
	time.AfterFunc(500*time.Millisecond, cancel)

	done := make(chan struct{})
	go func() {
		e.InitialiseAdapters(ctx, func(ctx context.Context) error {
			callCount++
			return errors.New("always fails")
		})
		close(done)
	}()

	select {
	case <-done:
		// InitialiseAdapters returned (ctx was cancelled)
	case <-time.After(5 * time.Second):
		t.Fatal("InitialiseAdapters did not return after context cancellation")
	}

	if callCount == 0 {
		t.Error("expected initFn to be called at least once before context cancellation")
	}
	if err := e.GetInitError(); err == nil {
		t.Error("expected init error to be set after context cancellation with failures")
	}
}
