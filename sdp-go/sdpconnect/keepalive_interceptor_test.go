package sdpconnect

import (
	"context"
	"errors"
	"testing"

	connect "connectrpc.com/connect"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/overmindtech/cli/sdp-go"
)

type testManagementServiceClient struct {
	// Error that will be returned
	Error error

	// Counts the number of times it has been called
	callCount int

	UnimplementedManagementServiceHandler
}

func (t *testManagementServiceClient) KeepaliveSources(context.Context, *connect.Request[sdp.KeepaliveSourcesRequest]) (*connect.Response[sdp.KeepaliveSourcesResponse], error) {
	t.callCount++

	return &connect.Response[sdp.KeepaliveSourcesResponse]{
		Msg: &sdp.KeepaliveSourcesResponse{},
	}, t.Error
}

func TestWaitForSources(t *testing.T) {
	t.Run("without the interceptor", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		err := WaitForSources(ctx)

		if err != nil {
			t.Error(err)
		}
	})

	tests := []struct {
		Name           string
		WaitForSources bool
		SourcesError   error
	}{
		{
			Name:           "without calling the wait function",
			WaitForSources: false,
			SourcesError:   nil,
		},
		{
			Name:           "with calling the wait function",
			WaitForSources: true,
			SourcesError:   nil,
		},
		{
			Name:           "when waking the sources fails but we aren't waiting on it",
			WaitForSources: false,
			SourcesError:   errors.New("test error"),
		},
		{
			Name:           "when waking the sources fails but we *are* waiting on it",
			WaitForSources: true,
			SourcesError:   errors.New("test error"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			client := testManagementServiceClient{
				Error: test.SourcesError,
			}
			i := NewKeepaliveSourcesInterceptor(&client)

			called := false

			testFunc := connect.UnaryFunc(func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
				var err error

				if test.WaitForSources {
					err = WaitForSources(ctx)
				}

				called = true
				return nil, err
			})

			testString := "test-account"
			ctx := sdp.OverrideCustomClaims(context.Background(), nil, &testString)

			// Wrap the function
			testFunc = i.WrapUnary(testFunc)

			// Call the function
			_, err := testFunc(ctx, nil)

			if test.SourcesError != nil && test.WaitForSources {
				// If the sources error is not nil and we are waiting for
				// sources, then we expect to see an error here
				if err == nil {
					t.Errorf("Expected error but got nil, despite test.SourcesError=%v", test.SourcesError)
				}

				if client.callCount != 1 {
					t.Errorf("Expected call count to be 1 but got %d", client.callCount)
				}
			} else {
				// Otherwise we shouldn't see an error
				if err != nil {
					t.Error(err)
				}
			}

			if called != true {
				t.Error("Wrapped function was not called")
			}
		})
	}

	t.Run("with caching", func(t *testing.T) {
		ctx := context.Background()

		// Mock the account name
		ctx = sdp.OverrideAuthContext(ctx, &validator.ValidatedClaims{
			CustomClaims: &sdp.CustomClaims{
				Scope:       "test",
				AccountName: "test",
			},
		})

		// Create the interceptor
		client := testManagementServiceClient{}
		i := NewKeepaliveSourcesInterceptor(&client)

		testFunc := connect.UnaryFunc(func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
			_ = WaitForSources(ctx)
			return nil, nil
		})

		// Wrap the function
		testFunc = i.WrapUnary(testFunc)

		// Call wake sources and expect the call count to be 1
		_, err := testFunc(ctx, nil)
		if err != nil {
			t.Error(err)
		}

		if client.callCount != 1 {
			t.Errorf("Expected call count to be 1 but got %d", client.callCount)
		}

		// Call wake sources again and expect the call count to be 1
		_, err = testFunc(ctx, nil)
		if err != nil {
			t.Error(err)
		}

		if client.callCount != 1 {
			t.Errorf("Expected call count to be 1 but got %d", client.callCount)
		}
	})
}
