package discovery

import (
	"context"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
)

type testHeartbeatClient struct {
	// Requests will be sent to this channel
	Requests chan *connect.Request[sdp.SubmitSourceHeartbeatRequest]
	// Responses should be sent here
	Responses chan *connect.Response[sdp.SubmitSourceHeartbeatResponse]
}

func (t testHeartbeatClient) SubmitSourceHeartbeat(ctx context.Context, req *connect.Request[sdp.SubmitSourceHeartbeatRequest]) (*connect.Response[sdp.SubmitSourceHeartbeatResponse], error) {
	t.Requests <- req
	return <-t.Responses, nil
}

func TestHeartbeats(t *testing.T) {
	name := t.Name()
	u := uuid.New()
	version := "v0.0.0-test"
	engineType := "aws"

	requests := make(chan *connect.Request[sdp.SubmitSourceHeartbeatRequest], 1)
	responses := make(chan *connect.Response[sdp.SubmitSourceHeartbeatResponse], 1)

	heartbeatOptions := HeartbeatOptions{
		ManagementClient: testHeartbeatClient{
			Requests:  requests,
			Responses: responses,
		},
	}
	ec := EngineConfig{
		SourceName:       name,
		SourceUUID:       u,
		Version:          version,
		EngineType:       engineType,
		HeartbeatOptions: &heartbeatOptions,
	}
	e, _ := NewEngine(&ec)

	if err := e.AddAdapters(
		&TestAdapter{
			ReturnScopes: []string{"test"},
			ReturnType:   "test-type",
			ReturnName:   "test-name",
		},
		&TestAdapter{
			ReturnScopes: []string{"test"},
			ReturnType:   "test-type2",
			ReturnName:   "test-name2",
		},
	); err != nil {
		t.Fatalf("unexpected error adding adapters: %v", err)
	}

	t.Run("sendHeartbeat when healthy", func(t *testing.T) {
		ec.HeartbeatOptions.ReadinessCheck = func(_ context.Context) error {
			return nil
		}
		responses <- &connect.Response[sdp.SubmitSourceHeartbeatResponse]{
			Msg: &sdp.SubmitSourceHeartbeatResponse{},
		}

		err := e.SendHeartbeat(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		req := <-requests

		if reqUUID, err := uuid.FromBytes(req.Msg.GetUUID()); err == nil {
			if reqUUID != u {
				t.Errorf("expected uuid %v, got %v", u, reqUUID)
			}
		} else {
			t.Errorf("error parsing uuid: %v", err)
		}

		if req.Msg.GetVersion() != version {
			t.Errorf("expected version %v, got %v", version, req.Msg.GetVersion())
		}

		if req.Msg.GetName() != name {
			t.Errorf("expected name %v, got %v", name, req.Msg.GetName())
		}

		if req.Msg.GetType() != engineType {
			t.Errorf("expected type %v, got %v", engineType, req.Msg.GetType())
		}

		if req.Msg.GetManaged() != sdp.SourceManaged_LOCAL {
			t.Errorf("expected managed %v, got %v", sdp.SourceManaged_LOCAL, req.Msg.GetManaged())
		}

		if req.Msg.GetError() != "" {
			t.Errorf("expected no error, got %v", req.Msg.GetError())
		}

		reqAvailableScopes := req.Msg.GetAvailableScopes()

		if len(reqAvailableScopes) != 1 {
			t.Errorf("expected 1 scope, got %v", len(reqAvailableScopes))
		}

		if !slices.Contains(reqAvailableScopes, "test") {
			t.Errorf("expected scope 'test' to be present in the response")
		}

		reqAdapterMetadata := req.Msg.GetAdapterMetadata()

		if len(reqAdapterMetadata) != 2 {
			t.Errorf("expected 2 adapter metadata, got %v", len(reqAdapterMetadata))
		}
	})

	t.Run("sendHeartbeat when unhealthy", func(t *testing.T) {
		e.EngineConfig.HeartbeatOptions.ReadinessCheck = func(_ context.Context) error {
			return ErrNoHealthcheckDefined
		}

		responses <- &connect.Response[sdp.SubmitSourceHeartbeatResponse]{
			Msg: &sdp.SubmitSourceHeartbeatResponse{},
		}

		err := e.SendHeartbeat(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		req := <-requests

		// Error message is no longer wrapped (wrapping removed to avoid double-prefixing)
		expectedError := "no healthcheck defined"
		if req.Msg.GetError() != expectedError {
			t.Errorf("expected error %q, got %q", expectedError, req.Msg.GetError())
		}
	})

	t.Run("startSendingHeartbeats", func(t *testing.T) {
		e.EngineConfig.HeartbeatOptions.Frequency = time.Millisecond * 250
		e.EngineConfig.HeartbeatOptions.ReadinessCheck = func(_ context.Context) error {
			return nil
		}

		ctx, cancel := context.WithCancel(context.Background())

		start := time.Now()

		responses <- &connect.Response[sdp.SubmitSourceHeartbeatResponse]{
			Msg: &sdp.SubmitSourceHeartbeatResponse{},
		}
		e.StartSendingHeartbeats(ctx)

		// Get the initial heartbeat
		<-requests

		// Get two
		responses <- &connect.Response[sdp.SubmitSourceHeartbeatResponse]{
			Msg: &sdp.SubmitSourceHeartbeatResponse{},
		}
		<-requests

		cancel()

		// Make sure that took the expected amount of time
		if elapsed := time.Since(start); elapsed < time.Millisecond*250 {
			t.Errorf("expected to take at least 500ms, took %v", elapsed)
		}

		if elapsed := time.Since(start); elapsed > time.Millisecond*500 {
			t.Errorf("expected to take at most 750ms, took %v", elapsed)
		}
	})
}

// TestSendHeartbeatNilManagementClient ensures unauthenticated/local dev mode
// (HeartbeatOptions set by SetReadinessCheck but ManagementClient nil) does not error.
func TestSendHeartbeatNilManagementClient(t *testing.T) {
	ec := EngineConfig{
		SourceName: t.Name(),
		SourceUUID: uuid.New(),
		Version:    "v0.0.0-test",
		EngineType: "aws",
		HeartbeatOptions: &HeartbeatOptions{
			ManagementClient: nil, // e.g. ALLOW_UNAUTHENTICATED - no API to send to
			Frequency:        time.Second * 30,
		},
	}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	err = e.SendHeartbeat(context.Background(), nil)
	if err != nil {
		t.Errorf("SendHeartbeat with nil ManagementClient should be no-op, got: %v", err)
	}
}
