package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/overmindtech/cli/sdp-go"
)

func TestToNatsOptions(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		o := NATSOptions{}

		expectedOptions, err := optionsToStruct([]nats.Option{
			nats.Timeout(ConnectionTimeoutDefault),
			nats.MaxReconnects(MaxReconnectsDefault),
			nats.ReconnectWait(ReconnectWaitDefault),
			nats.ReconnectJitter(ReconnectJitterDefault, ReconnectJitterDefault),
			nats.ConnectHandler(ConnectHandlerDefault),
			nats.DisconnectErrHandler(DisconnectErrHandlerDefault),
			nats.ReconnectHandler(ReconnectHandlerDefault),
			nats.ClosedHandler(ClosedHandlerDefault),
			nats.LameDuckModeHandler(LameDuckModeHandlerDefault),
			nats.ErrorHandler(ErrorHandlerDefault),
		})
		if err != nil {
			t.Fatal(err)
		}

		server, options := o.ToNatsOptions()

		if server != "" {
			t.Error("Expected server to be empty")
		}

		actualOptions, err := optionsToStruct(options)
		if err != nil {
			t.Fatal(err)
		}

		if expectedOptions.MaxReconnect != actualOptions.MaxReconnect {
			t.Errorf("Expected MaxReconnect to be %v, got %v", expectedOptions.MaxReconnect, actualOptions.MaxReconnect)
		}

		if expectedOptions.Timeout != actualOptions.Timeout {
			t.Errorf("Expected ConnectionTimeout to be %v, got %v", expectedOptions.Timeout, actualOptions.Timeout)
		}

		if expectedOptions.ReconnectWait != actualOptions.ReconnectWait {
			t.Errorf("Expected ReconnectWait to be %v, got %v", expectedOptions.ReconnectWait, actualOptions.ReconnectWait)
		}

		if expectedOptions.ReconnectJitter != actualOptions.ReconnectJitter {
			t.Errorf("Expected ReconnectJitter to be %v, got %v", expectedOptions.ReconnectJitter, actualOptions.ReconnectJitter)
		}

		// TokenClient
		if expectedOptions.UserJWT != nil || expectedOptions.SignatureCB != nil {
			t.Error("Expected TokenClient to be nil")
		}

		if actualOptions.DisconnectedErrCB == nil {
			t.Error("Expected DisconnectedErrCB to be non-nil")
		}

		if actualOptions.ReconnectedCB == nil {
			t.Error("Expected ReconnectedCB to be non-nil")
		}

		if actualOptions.ClosedCB == nil {
			t.Error("Expected ClosedCB to be non-nil")
		}

		if actualOptions.LameDuckModeHandler == nil {
			t.Error("Expected LameDuckModeHandler to be non-nil")
		}

		if actualOptions.AsyncErrorCB == nil {
			t.Error("Expected AsyncErrorCB to be non-nil")
		}
	})

	t.Run("with non-defaults", func(t *testing.T) {
		var connectHandlerUsed bool
		var disconnectErrHandlerUsed bool
		var reconnectHandlerUsed bool
		var closedHandlerUsed bool
		var lameDuckModeHandlerUsed bool
		var errorHandlerUsed bool

		o := NATSOptions{
			Servers:              []string{"one", "two"},
			ConnectionName:       "foo",
			MaxReconnects:        999,
			ReconnectWait:        999,
			ReconnectJitter:      999,
			ConnectHandler:       func(c *nats.Conn) { connectHandlerUsed = true },
			DisconnectErrHandler: func(c *nats.Conn, err error) { disconnectErrHandlerUsed = true },
			ReconnectHandler:     func(c *nats.Conn) { reconnectHandlerUsed = true },
			ClosedHandler:        func(c *nats.Conn) { closedHandlerUsed = true },
			LameDuckModeHandler:  func(c *nats.Conn) { lameDuckModeHandlerUsed = true },
			ErrorHandler:         func(c *nats.Conn, s *nats.Subscription, err error) { errorHandlerUsed = true },
		}

		expectedOptions, err := optionsToStruct([]nats.Option{
			nats.Name("foo"),
			nats.MaxReconnects(999),
			nats.ReconnectWait(999),
			nats.ReconnectJitter(999, 999),
			nats.DisconnectErrHandler(nil),
			nats.ReconnectHandler(nil),
			nats.ClosedHandler(nil),
			nats.LameDuckModeHandler(nil),
			nats.ErrorHandler(nil),
		})
		if err != nil {
			t.Fatal(err)
		}

		server, options := o.ToNatsOptions()

		if server != "one,two" {
			t.Errorf("Expected server to be one,two got %v", server)
		}

		actualOptions, err := optionsToStruct(options)
		if err != nil {
			t.Fatal(err)
		}

		if expectedOptions.MaxReconnect != actualOptions.MaxReconnect {
			t.Errorf("Expected MaxReconnect to be %v, got %v", expectedOptions.MaxReconnect, actualOptions.MaxReconnect)
		}

		if expectedOptions.ReconnectWait != actualOptions.ReconnectWait {
			t.Errorf("Expected ReconnectWait to be %v, got %v", expectedOptions.ReconnectWait, actualOptions.ReconnectWait)
		}

		if expectedOptions.ReconnectJitter != actualOptions.ReconnectJitter {
			t.Errorf("Expected ReconnectJitter to be %v, got %v", expectedOptions.ReconnectJitter, actualOptions.ReconnectJitter)
		}

		if actualOptions.DisconnectedErrCB != nil {
			actualOptions.DisconnectedErrCB(nil, nil)
			if !disconnectErrHandlerUsed {
				t.Error("DisconnectErrHandler not used")
			}
		} else {
			t.Error("Expected DisconnectedErrCB to non-nil")
		}

		if actualOptions.ConnectedCB != nil {
			actualOptions.ConnectedCB(nil)
			if !connectHandlerUsed {
				t.Error("ConnectHandler not used")
			}
		} else {
			t.Error("Expected ConnectedCB to non-nil")
		}

		if actualOptions.ReconnectedCB != nil {
			actualOptions.ReconnectedCB(nil)
			if !reconnectHandlerUsed {
				t.Error("ReconnectHandler not used")
			}
		} else {
			t.Error("Expected ReconnectedCB to non-nil")
		}

		if actualOptions.ClosedCB != nil {
			actualOptions.ClosedCB(nil)
			if !closedHandlerUsed {
				t.Error("ClosedHandler not used")
			}
		} else {
			t.Error("Expected ClosedCB to non-nil")
		}

		if actualOptions.LameDuckModeHandler != nil {
			actualOptions.LameDuckModeHandler(nil)
			if !lameDuckModeHandlerUsed {
				t.Error("LameDuckModeHandler not used")
			}
		} else {
			t.Error("Expected LameDuckModeHandler to non-nil")
		}

		if actualOptions.AsyncErrorCB != nil {
			actualOptions.AsyncErrorCB(nil, nil, nil)
			if !errorHandlerUsed {
				t.Error("ErrorHandler not used")
			}
		} else {
			t.Error("Expected AsyncErrorCB to non-nil")
		}
	})
}

func TestNATSConnect(t *testing.T) {
	t.Run("with a bad URL", func(t *testing.T) {
		o := NATSOptions{
			Servers:    []string{"nats://badname.dontresolve.com"},
			NumRetries: 5,
			RetryDelay: 100 * time.Millisecond,
		}

		start := time.Now()

		_, err := o.Connect()

		// Just sanity check the duration here, it should not be less than
		// NumRetries * RetryDelay and it should be more than... Some larger
		// number of seconds. This is very much dependant on how long it takes
		// to not resolve the name
		if time.Since(start) < 5*100*time.Millisecond {
			t.Errorf("Reconnecting didn't take long enough, expected >0.5s got: %v", time.Since(start).String())
		}

		if time.Since(start) > 3*time.Second {
			t.Errorf("Reconnecting took too long, expected <3s got: %v", time.Since(start).String())
		}

		var maxRetriesError MaxRetriesError
		if !errors.As(err, &maxRetriesError) {
			t.Errorf("Unknown error type %T: %v", err, err)
		}
	})

	t.Run("with a bad URL, but a good token", func(t *testing.T) {
		tk := GetTestOAuthTokenClient(t)

		startToken, err := tk.GetJWT()
		if err != nil {
			t.Fatal(err)
		}

		o := NATSOptions{
			Servers:     []string{"nats://badname.dontresolve.com"},
			TokenClient: tk,
			NumRetries:  3,
			RetryDelay:  100 * time.Millisecond,
		}

		_, err = o.Connect()

		var maxRetriesError MaxRetriesError
		if errors.As(err, &maxRetriesError) {
			// Make sure we have only got one token, not three
			currentToken, err := o.TokenClient.GetJWT()
			if err != nil {
				t.Fatal(err)
			}

			if currentToken != startToken {
				t.Error("Tokens have changed")
			}
		} else {
			t.Errorf("Unknown error type %T", err)
		}
	})

	t.Run("with a good URL", func(t *testing.T) {
		o := NATSOptions{
			Servers: []string{
				"nats://nats:4222",
				"nats://localhost:4222",
			},
			NumRetries: 3,
			RetryDelay: 100 * time.Millisecond,
		}

		conn, err := o.Connect()
		if err != nil {
			t.Fatal(err)
		}

		ValidateNATSConnection(t, conn)
	})

	t.Run("with a good URL but no retries", func(t *testing.T) {
		o := NATSOptions{
			Servers: []string{
				"nats://nats:4222",
				"nats://localhost:4222",
			},
		}

		conn, err := o.Connect()
		if err != nil {
			t.Fatal(err)
		}

		ValidateNATSConnection(t, conn)
	})

	t.Run("with a good URL and infinite retries", func(t *testing.T) {
		o := NATSOptions{
			Servers: []string{
				"nats://nats:4222",
				"nats://localhost:4222",
			},
			NumRetries: -1,
			RetryDelay: 100 * time.Millisecond,
		}

		conn, err := o.Connect()
		if err != nil {
			t.Error(err)
		}

		ValidateNATSConnection(t, conn)
	})
}

func TestTokenRefresh(t *testing.T) {
	tk := GetTestOAuthTokenClient(t)

	// Get a token
	token, err := tk.GetJWT()
	if err != nil {
		t.Fatal(err)
	}

	// Artificially set the expiry and replace the token
	claims, err := jwt.DecodeUserClaims(token)
	if err != nil {
		t.Fatal(err)
	}

	pair, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatal(err)
	}

	claims.Expires = time.Now().Add(-10 * time.Second).Unix()
	tk.jwt, err = claims.Encode(pair)
	expiredToken := tk.jwt

	if err != nil {
		t.Error(err)
	}

	// Get the token again
	newToken, err := tk.GetJWT()
	if err != nil {
		t.Error(err)
	}

	if expiredToken == newToken {
		t.Error("token is unchanged")
	}
}

func ValidateNATSConnection(t *testing.T, ec sdp.EncodedConnection) {
	t.Helper()
	done := make(chan struct{})

	sub, err := ec.Subscribe("test", sdp.NewQueryResponseHandler("test", func(ctx context.Context, qr *sdp.QueryResponse) {
		rt, ok := qr.GetResponseType().(*sdp.QueryResponse_Response)
		if !ok {
			t.Errorf("Received unexpected message: %v", qr)
		}

		if rt.Response.GetResponder() == "test" {
			done <- struct{}{}
		}
	}))
	if err != nil {
		t.Error(err)
	}

	ru := uuid.New()
	err = ec.Publish(context.Background(), "test", sdp.NewQueryResponseFromResponse(&sdp.Response{
		Responder:     "test",
		ResponderUUID: ru[:],
		State:         sdp.ResponderState_COMPLETE,
	}))
	if err != nil {
		t.Error(err)
	}

	// Wait for the message to come back
	select {
	case <-done:
		// Good
	case <-time.After(500 * time.Millisecond):
		t.Error("Didn't get message after 500ms")
	}

	err = sub.Unsubscribe()
	if err != nil {
		t.Error(err)
	}
}

func optionsToStruct(options []nats.Option) (nats.Options, error) {
	var o nats.Options
	var err error

	for _, option := range options {
		err = option(&o)
		if err != nil {
			return o, err
		}
	}

	return o, nil
}
