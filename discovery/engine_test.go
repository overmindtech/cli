package discovery

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/overmindtech/cli/auth"
	"github.com/overmindtech/cli/sdp-go"
	"golang.org/x/oauth2"
)

func newEngine(t *testing.T, name string, no *auth.NATSOptions, eConn sdp.EncodedConnection, adapters ...Adapter) *Engine {
	t.Helper()

	if no != nil && eConn != nil {
		t.Fatal("Cannot provide both NATSOptions and EncodedConnection")
	}

	ec := EngineConfig{
		MaxParallelExecutions: 10,
		SourceName:            name,
		NATSQueueName:         "test",
	}
	if no != nil {
		ec.NATSOptions = no
	} else if eConn == nil {
		ec.NATSOptions = &auth.NATSOptions{
			NumRetries:        5,
			RetryDelay:        time.Second,
			Servers:           NatsTestURLs,
			ConnectionName:    "test-connection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     5,
			TokenClient:       GetTestOAuthTokenClient(t, "org_hdeUXbB55sMMvJLa"),
		}
	}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("Error initializing Engine: %v", err)
	}

	if eConn != nil {
		e.natsConnection = eConn
	}

	if err := e.AddAdapters(adapters...); err != nil {
		t.Fatalf("Error adding adapters: %v", err)
	}

	return e
}

func newStartedEngine(t *testing.T, name string, no *auth.NATSOptions, eConn sdp.EncodedConnection, adapters ...Adapter) *Engine {
	t.Helper()

	e := newEngine(t, name, no, eConn, adapters...)

	err := e.Start()
	if err != nil {
		t.Fatalf("Error starting Engine: %v", err)
	}

	t.Cleanup(func() {
		err = e.Stop()
		if err != nil {
			t.Errorf("Error stopping Engine: %v", err)
		}
	})

	return e
}

func TestTrackQuery(t *testing.T) {
	t.Run("With normal query", func(t *testing.T) {
		t.Parallel()

		e := newStartedEngine(t, "TestTrackQuery_normal", nil, nil)

		u := uuid.New()

		qt := QueryTracker{
			Engine: e,
			Query: &sdp.Query{
				Type:   "person",
				Method: sdp.QueryMethod_LIST,
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth: 10,
				},
				UUID: u[:],
			},
		}

		e.TrackQuery(u, &qt)

		if got, err := e.GetTrackedQuery(u); err == nil {
			if got != &qt {
				t.Errorf("Got mismatched QueryTracker objects %v and %v", got, &qt)
			}
		} else {
			t.Error(err)
		}
	})

	t.Run("With many queries", func(t *testing.T) {
		t.Parallel()

		e := newStartedEngine(t, "TestTrackQuery_many", nil, nil)

		var wg sync.WaitGroup

		for i := range 1000 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				u := uuid.New()

				qt := QueryTracker{
					Engine: e,
					Query: &sdp.Query{
						Type:   "person",
						Query:  fmt.Sprintf("person-%v", i),
						Method: sdp.QueryMethod_GET,
						RecursionBehaviour: &sdp.Query_RecursionBehaviour{
							LinkDepth: 10,
						},
						UUID: u[:],
					},
				}

				e.TrackQuery(u, &qt)
			}(i)
		}

		wg.Wait()

		if len(e.trackedQueries) != 1000 {
			t.Errorf("Expected 1000 tracked queries, got %v", len(e.trackedQueries))
		}
	})
}

func TestDeleteTrackedQuery(t *testing.T) {
	t.Parallel()
	e := newStartedEngine(t, "TestDeleteTrackedQuery", nil, nil)

	var wg sync.WaitGroup

	// Add and delete many query in parallel
	for i := 1; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u := uuid.New()

			qt := QueryTracker{
				Engine: e,
				Query: &sdp.Query{
					Type:   "person",
					Query:  fmt.Sprintf("person-%v", i),
					Method: sdp.QueryMethod_GET,
					RecursionBehaviour: &sdp.Query_RecursionBehaviour{
						LinkDepth: 10,
					},
					UUID: u[:],
				},
			}

			e.TrackQuery(u, &qt)
			wg.Add(1)
			go func(u uuid.UUID) {
				defer wg.Done()
				e.DeleteTrackedQuery(u)
			}(u)
		}(i)
	}

	wg.Wait()

	if len(e.trackedQueries) != 0 {
		t.Errorf("Expected 0 tracked queries, got %v", len(e.trackedQueries))
	}
}

func TestNats(t *testing.T) {
	SkipWithoutNats(t)

	ec := EngineConfig{
		MaxParallelExecutions: 10,
		SourceName:            "nats-test",
		NATSOptions: &auth.NATSOptions{
			NumRetries:        5,
			RetryDelay:        time.Second,
			Servers:           NatsTestURLs,
			ConnectionName:    "test-connection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     5,
		},
		NATSQueueName: "test",
	}

	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("Error initializing Engine: %v", err)
	}

	adapter := TestAdapter{}

	err = e.AddAdapters(
		&adapter,
		&TestAdapter{
			ReturnScopes: []string{
				sdp.WILDCARD,
			},
			ReturnName: "test-adapter",
			ReturnType: "test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Starting", func(t *testing.T) {
		err := e.Start()
		if err != nil {
			t.Error(err)
		}

		if e.natsConnection.Underlying().NumSubscriptions() != 4 {
			t.Errorf("Expected engine to have 4 subscriptions, got %v", e.natsConnection.Underlying().NumSubscriptions())
		}
	})

	t.Run("Restarting", func(t *testing.T) {
		err := e.Stop()
		if err != nil {
			t.Error(err)
		}

		err = e.Start()
		if err != nil {
			t.Error(err)
		}

		if e.natsConnection.Underlying().NumSubscriptions() != 4 {
			t.Errorf("Expected engine to have 4 subscriptions, got %v", e.natsConnection.Underlying().NumSubscriptions())
		}
	})

	t.Run("Handling a basic query", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
			e.ClearCache()
		})

		query := &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "basic",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 0,
			},
			Scope: "test",
		}

		_, _, _, err := sdp.RunSourceQuerySync(context.Background(), query, sdp.DefaultStartTimeout, e.natsConnection)
		if err != nil {
			t.Error(err)
		}

		if len(adapter.GetCalls) != 1 {
			t.Errorf("expected 1 get call, got %v: %v", len(adapter.GetCalls), adapter.GetCalls)
		}
	})

	t.Run("stopping", func(t *testing.T) {
		err := e.Stop()
		if err != nil {
			t.Error(err)
		}
	})
}

func TestNatsCancel(t *testing.T) {
	SkipWithoutNats(t)

	ec := EngineConfig{
		MaxParallelExecutions: 1,
		SourceName:            "nats-test",
		NATSOptions: &auth.NATSOptions{
			NumRetries:        5,
			RetryDelay:        time.Second,
			Servers:           NatsTestURLs,
			ConnectionName:    "test-connection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     5,
		},
		NATSQueueName: "test",
	}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("Error initializing Engine: %v", err)
	}

	adapter := SpeedTestAdapter{
		QueryDelay:   2 * time.Second,
		ReturnType:   "person",
		ReturnScopes: []string{"test"},
	}

	if err := e.AddAdapters(&adapter); err != nil {
		t.Fatalf("Error adding adapters: %v", err)
	}

	t.Run("Starting", func(t *testing.T) {
		err := e.Start()
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Cancelling queries", func(t *testing.T) {
		conn := e.natsConnection
		u := uuid.New()

		query := &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "foo",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 100,
			},
			Scope: "*",
			UUID:  u[:],
		}

		responses := make(chan *sdp.QueryResponse, 1000)
		progress, err := sdp.RunSourceQuery(t.Context(), query, sdp.DefaultStartTimeout, conn, responses)
		if err != nil {
			t.Error(err)
		}

		time.Sleep(250 * time.Millisecond)

		err = conn.Publish(context.Background(), "cancel.all", &sdp.CancelQuery{
			UUID: u[:],
		})
		if err != nil {
			t.Error(err)
		}

		// Read and discard all items and errors until they are closed
		for range responses {
		}

		time.Sleep(250 * time.Millisecond)

		if progress.Progress().Cancelled != 1 {
			t.Errorf("Expected query to be cancelled, got\n%v", progress.String())
		}
	})

	t.Run("stopping", func(t *testing.T) {
		err := e.Stop()
		if err != nil {
			t.Error(err)
		}
	})
}

func TestNatsConnections(t *testing.T) {
	t.Run("with a bad hostname", func(t *testing.T) {
		ec := EngineConfig{
			MaxParallelExecutions: 1,
			SourceName:            "nats-test",
			NATSOptions: &auth.NATSOptions{
				Servers:           []string{"nats://bad.server"},
				ConnectionName:    "test-disconnection",
				ConnectionTimeout: time.Second,
				MaxReconnects:     1,
			},
			NATSQueueName: "test",
		}
		e, err := NewEngine(&ec)
		if err != nil {
			t.Fatalf("Error initializing Engine: %v", err)
		}

		err = e.Start()

		if err == nil {
			t.Error("expected error but got nil")
		}
	})

	t.Run("with a server that disconnects", func(t *testing.T) {
		// We are running a custom server here so that we can control its lifecycle
		opts := test.DefaultTestOptions
		// Need to change this to avoid port clashes in github actions
		opts.Port = 4111
		s := test.RunServer(&opts)

		if !s.ReadyForConnections(10 * time.Second) {
			t.Fatal("Could not start goroutine NATS server")
		}

		t.Cleanup(func() {
			if s != nil {
				s.Shutdown()
			}
		})

		ec := EngineConfig{
			MaxParallelExecutions: 1,
			SourceName:            "nats-test",
			NATSOptions: &auth.NATSOptions{
				NumRetries:        5,
				RetryDelay:        time.Second,
				Servers:           []string{"127.0.0.1:4111"},
				ConnectionName:    "test-disconnection",
				ConnectionTimeout: time.Second,
				MaxReconnects:     10,
				ReconnectWait:     time.Second,
				ReconnectJitter:   time.Second,
			},
			NATSQueueName: "test",
		}
		e, err := NewEngine(&ec)
		if err != nil {
			t.Fatalf("Error initializing Engine: %v", err)
		}

		err = e.Start()
		if err != nil {
			t.Fatal(err)
		}

		t.Log("Stopping NATS server")
		s.Shutdown()

		for i := range 21 {
			if i == 20 {
				t.Errorf("Engine did not report a NATS disconnect after %v tries", i)
			}

			if !e.IsNATSConnected() {
				break
			}

			time.Sleep(time.Second)
		}

		// Reset the server
		s = test.RunServer(&opts)

		// Wait for the server to start
		s.ReadyForConnections(10 * time.Second)

		// Wait 2 more seconds for a reconnect
		time.Sleep(2 * time.Second)

		for range 21 {
			if e.IsNATSConnected() {
				return
			}

			time.Sleep(time.Second)
		}

		t.Error("Engine should have reconnected but hasn't")
	})

	t.Run("with a server that takes a while to start", func(t *testing.T) {
		// We are running a custom server here so that we can control its lifecycle
		opts := test.DefaultTestOptions
		// Need to change this to avoid port clashes in github actions
		opts.Port = 4112

		ec := EngineConfig{
			MaxParallelExecutions: 1,
			SourceName:            "nats-test",
			NATSOptions: &auth.NATSOptions{
				NumRetries:        10,
				RetryDelay:        time.Second,
				Servers:           []string{"127.0.0.1:4112"},
				ConnectionName:    "test-disconnection",
				ConnectionTimeout: time.Second,
				MaxReconnects:     10,
				ReconnectWait:     time.Second,
				ReconnectJitter:   time.Second,
			},
			NATSQueueName: "test",
		}
		e, err := NewEngine(&ec)
		if err != nil {
			t.Fatalf("Error initializing Engine: %v", err)
		}

		var s *server.Server

		go func() {
			// Start the server after a delay
			time.Sleep(2 * time.Second)

			// We are running a custom server here so that we can control its lifecycle
			s = test.RunServer(&opts)

			t.Cleanup(func() {
				if s != nil {
					s.Shutdown()
				}
			})
		}()

		err = e.Start()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestNATSFailureRestart(t *testing.T) {
	restartTestOption := test.DefaultTestOptions
	restartTestOption.Port = 4113

	// We are running a custom server here so that we can control its lifecycle
	s := test.RunServer(&restartTestOption)

	if !s.ReadyForConnections(10 * time.Second) {
		t.Fatal("Could not start goroutine NATS server")
	}

	ec := EngineConfig{
		MaxParallelExecutions: 1,
		SourceName:            "nats-test",
		NATSOptions: &auth.NATSOptions{
			NumRetries:        10,
			RetryDelay:        time.Second,
			Servers:           []string{"127.0.0.1:4113"},
			ConnectionName:    "test-disconnection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     10,
			ReconnectWait:     100 * time.Millisecond,
			ReconnectJitter:   10 * time.Millisecond,
		},
		NATSQueueName: "test",
	}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("Error initializing Engine: %v", err)
	}

	e.ConnectionWatchInterval = 1 * time.Second

	// Connect successfully
	err = e.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = e.Stop()
		if err != nil {
			t.Fatal(err)
		}
	})

	// Lose the connection
	t.Log("Stopping NATS server")
	s.Shutdown()
	s.WaitForShutdown()

	// The watcher should keep watching while the nats connection is
	// RECONNECTING, once it's CLOSED however it won't keep trying to connect so
	// we want to make sure that the watcher detects this and kills the whole
	// thing
	time.Sleep(2 * time.Second)

	s = test.RunServer(&restartTestOption)
	if !s.ReadyForConnections(10 * time.Second) {
		t.Fatal("Could not start goroutine NATS server a second time")
	}

	t.Cleanup(func() {
		s.Shutdown()
	})

	time.Sleep(3 * time.Second)

	if !e.IsNATSConnected() {
		t.Error("NATS didn't manage to reconnect")
	}
}

func TestNatsAuth(t *testing.T) {
	SkipWithoutNatsAuth(t)

	ec := EngineConfig{
		MaxParallelExecutions: 1,
		SourceName:            "nats-test",
		NATSOptions: &auth.NATSOptions{
			NumRetries:        5,
			RetryDelay:        time.Second,
			Servers:           NatsTestURLs,
			ConnectionName:    "test-connection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     5,
			TokenClient:       GetTestOAuthTokenClient(t, "org_hdeUXbB55sMMvJLa"),
		},
		NATSQueueName: "test",
	}
	e, err := NewEngine(&ec)
	if err != nil {
		t.Fatalf("Error initializing Engine: %v", err)
	}

	adapter := TestAdapter{}
	if err := e.AddAdapters(
		&adapter,
		&TestAdapter{
			ReturnScopes: []string{
				sdp.WILDCARD,
			},
			ReturnType: "test",
			ReturnName: "test-adapter",
		},
	); err != nil {
		t.Fatalf("Error adding adapters: %v", err)
	}

	t.Run("Starting", func(t *testing.T) {
		err := e.Start()
		if err != nil {
			t.Fatal(err)
		}

		if e.natsConnection.Underlying().NumSubscriptions() != 4 {
			t.Errorf("Expected engine to have 4 subscriptions, got %v", e.natsConnection.Underlying().NumSubscriptions())
		}
	})

	t.Run("Handling a basic query", func(t *testing.T) {
		t.Cleanup(func() {
			adapter.ClearCalls()
			e.ClearCache()
		})

		query := &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "basic",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 0,
			},
			Scope: "test",
		}

		_, _, _, err := sdp.RunSourceQuerySync(t.Context(), query, sdp.DefaultStartTimeout, e.natsConnection)
		if err != nil {
			t.Error(err)
		}

		if len(adapter.GetCalls) != 1 {
			t.Errorf("expected 1 get call, got %v: %v", len(adapter.GetCalls), adapter.GetCalls)
		}
	})

	t.Run("stopping", func(t *testing.T) {
		err := e.Stop()
		if err != nil {
			t.Error(err)
		}
	})
}

func TestSetupMaxQueryTimeout(t *testing.T) {
	t.Run("with no value", func(t *testing.T) {
		ec := EngineConfig{}
		e, err := NewEngine(&ec)
		if err != nil {
			t.Fatalf("Error initializing Engine: %v", err)
		}

		if e.MaxRequestTimeout != DefaultMaxRequestTimeout {
			t.Errorf("max request timeout did not default. Got %v expected %v", e.MaxRequestTimeout.String(), DefaultMaxRequestTimeout.String())
		}
	})

	t.Run("with a value", func(t *testing.T) {
		ec := EngineConfig{}
		e, err := NewEngine(&ec)
		if err != nil {
			t.Fatalf("Error initializing Engine: %v", err)
		}
		e.MaxRequestTimeout = 1 * time.Second

		if e.MaxRequestTimeout != 1*time.Second {
			t.Errorf("max request timeout did not take provided value. Got %v expected %v", e.MaxRequestTimeout.String(), (1 * time.Second).String())
		}
	})
}

var (
	testTokenSource   oauth2.TokenSource
	testTokenSourceMu sync.Mutex
)

func GetTestOAuthTokenClient(t *testing.T, account string) auth.TokenClient {
	var domain string
	var clientID string
	var clientSecret string
	var exists bool

	errorFormat := "environment variable %v not found. Set up your test environment first. See: https://github.com/overmindtech/cli/auth0-test-data"

	// Read secrets form the environment
	if domain, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_DOMAIN"); !exists || domain == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_DOMAIN")
		t.Skip("Skipping due to missing environment setup")
	}

	if clientID, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_CLIENT_ID"); !exists || clientID == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_CLIENT_ID")
		t.Skip("Skipping due to missing environment setup")
	}

	if clientSecret, exists = os.LookupEnv("OVERMIND_NTE_ALLPERMS_CLIENT_SECRET"); !exists || clientSecret == "" {
		t.Errorf(errorFormat, "OVERMIND_NTE_ALLPERMS_CLIENT_SECRET")
		t.Skip("Skipping due to missing environment setup")
	}

	exchangeURL, err := GetWorkingTokenExchange()
	if err != nil {
		t.Fatal(err)
	}

	testTokenSourceMu.Lock()
	defer testTokenSourceMu.Unlock()
	if testTokenSource == nil {
		ccc := auth.ClientCredentialsConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		}
		testTokenSource = ccc.TokenSource(
			t.Context(),
			fmt.Sprintf("https://%v/oauth/token", domain),
			os.Getenv("API_SERVER_AUDIENCE"),
		)
	}

	return auth.NewOAuthTokenClient(
		exchangeURL,
		account,
		testTokenSource,
	)
}
