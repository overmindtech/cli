package sdpconnect

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/tracing"
)

// KeepaliveClient is the minimal slice from ManagementServiceClient to be able
// to run the KeepaliveSourcesInterceptor. This slicing is required to be able
// to directly reuse the service handler without running into issues where the
// Handler and Client methods diverge (e.g. when using `stream`)
type KeepaliveClient interface {
	KeepaliveSources(context.Context, *connect.Request[sdp.KeepaliveSourcesRequest]) (*connect.Response[sdp.KeepaliveSourcesResponse], error)
}

// Create a new interceptor that will ensure that sources are alive on all
// requests. This interceptor will call `KeepaliveSources` on the management
// service to ensure that the sources are alive. This will be done in a
// goroutine so that the request is not blocked. If the management service is
// not set, then this interceptor will do nothing.
//
// For services that actually require the sources to be alive, they can use the
// WaitForSources function to wait for the sources to be ready. This function
// will block until the sources are ready.
func NewKeepaliveSourcesInterceptor(managementClient KeepaliveClient) connect.Interceptor {
	return &KeepaliveSourcesInterceptor{
		keepalive:  managementClient,
		lastCalled: make(map[string]time.Time),
	}
}

// WaitForSources will wait for the sources to be ready after they have been
// woken up by the `KeepaliveSourcesInterceptor`. If this context was created
// without the interceptor, then this function will return immediately. If the
// waking of the sources returns an error it will be returned via this function
func WaitForSources(ctx context.Context) error {
	// Check the context key
	if readyFunc := ctx.Value(keepaliveSourcesReadyContextKey{}); readyFunc != nil {
		// Call the function
		return readyFunc.(waitForSourcesFunc)()
	} else {
		// Return immediately
		return nil
	}
}

type KeepaliveSourcesInterceptor struct {
	// Map of when the sources were last kept alive for each account, and the
	// time that the call was made
	lastCalled map[string]time.Time
	m          sync.RWMutex

	keepalive KeepaliveClient
}

// keepaliveSourcesReadyContextKey is the context key used to determine if the
// keepalive sources interceptor has run and the sources are ready
type keepaliveSourcesReadyContextKey struct{}

// A func that waits for the sources to be ready
type waitForSourcesFunc func() error

// Returns whether or not the keepalive should actually be called for this
// request. This is based on a cache to ensure that we aren't spamming the
// endpoint when we don't need to
func (i *KeepaliveSourcesInterceptor) shouldCallKeepalive(ctx context.Context) bool {
	// Extract the account name from the context

	accountName, ok := ctx.Value(sdp.AccountNameContextKey{}).(string)
	if !ok || accountName == "" {
		return false
	}

	i.m.RLock()
	lastCalled, exists := i.lastCalled[accountName]
	i.m.RUnlock()

	if !exists {
		return true
	}

	// If the last called time is more then 10 minutes ago, then we should
	// call the endpoint again
	return time.Since(lastCalled) > 10*time.Minute
}

// Update the last called time for the account in the context
func (i *KeepaliveSourcesInterceptor) updateLastCalled(ctx context.Context) {
	// Extract the account name from the context
	accountName, ok := ctx.Value(sdp.AccountNameContextKey{}).(string)
	if !ok || accountName == "" {
		return
	}

	i.m.Lock()
	defer i.m.Unlock()

	i.lastCalled[accountName] = time.Now()
}

func (i *KeepaliveSourcesInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return connect.UnaryFunc(func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
		// Wake the sources
		ctx = i.wakeSources(ctx)

		return next(ctx, ar)
	})
}

func (i *KeepaliveSourcesInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return connect.StreamingClientFunc(func(ctx context.Context, s connect.Spec) connect.StreamingClientConn {
		// Wake the sources
		ctx = i.wakeSources(ctx)

		return next(ctx, s)
	})
}

func (i *KeepaliveSourcesInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return connect.StreamingHandlerFunc(func(ctx context.Context, shc connect.StreamingHandlerConn) error {
		// Wake the sources
		ctx = i.wakeSources(ctx)

		return next(ctx, shc)
	})
}

// Actually does the work of waking the sources and attaching the channel to the
// context. Returns a new context that has the channel attached to it
func (i *KeepaliveSourcesInterceptor) wakeSources(ctx context.Context) context.Context {
	if i.keepalive == nil {
		return ctx
	}

	// If the function has already been set, then we don't need to do
	// anything since the middleware has already run
	if readyFunc := ctx.Value(keepaliveSourcesReadyContextKey{}); readyFunc != nil {
		return ctx
	}

	// Check that we haven't already called the endpoint recently
	if !i.shouldCallKeepalive(ctx) {
		return ctx
	}

	// Create a buffered channel so that if the value is never used, the
	// goroutine that keeps the sources awake can close. This will be
	// garbage collected when there are no longer any references to it,
	// which will happen once the context is garbage collected after the
	// request is fully completed
	sourcesReady := make(chan error, 1)

	// Attach a function to the context that will wait for the sources to be
	// ready
	ctx = context.WithValue(ctx, keepaliveSourcesReadyContextKey{}, waitForSourcesFunc(func() error {
		return <-sourcesReady
	}))

	// Make the request in another goroutine so that we don't block the
	// request
	go func() {
		defer tracing.LogRecoverToReturn(ctx, "KeepaliveSourcesInterceptor.wakeSources")
		defer close(sourcesReady)

		// Make the request to keep the source awake
		_, err := i.keepalive.KeepaliveSources(ctx, &connect.Request[sdp.KeepaliveSourcesRequest]{
			Msg: &sdp.KeepaliveSourcesRequest{
				WaitForHealthy: true,
			},
		})

		i.updateLastCalled(ctx)

		// Send the error to the channel
		sourcesReady <- err
	}()

	return ctx
}
