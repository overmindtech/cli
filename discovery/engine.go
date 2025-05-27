package discovery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/overmindtech/cli/auth"
	"github.com/overmindtech/cli/sdp-go"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	DefaultMaxRequestTimeout       = 5 * time.Minute
	DefaultConnectionWatchInterval = 3 * time.Second
)

// The client that will be used to send heartbeats. This will usually be an
// `sdpconnect.ManagementServiceClient`
type HeartbeatClient interface {
	SubmitSourceHeartbeat(context.Context, *connect.Request[sdp.SubmitSourceHeartbeatRequest]) (*connect.Response[sdp.SubmitSourceHeartbeatResponse], error)
}

type HeartbeatOptions struct {
	// The client that will be used to send heartbeats
	ManagementClient HeartbeatClient

	// The function that should be run to check if the adapter is healthy. It
	// will be executed each time a heartbeat is sent and should return an error
	// if the adapter is unhealthy.
	HealthCheck func(context.Context) error

	// How frequently to send a heartbeat
	Frequency time.Duration
}

// EngineConfig is the configuration for the engine
// it is used to configure the engine before starting it
type EngineConfig struct {
	EngineType   string    // The type of the engine, e.g. "aws" or "kubernetes"
	Version      string    // The version of the adapter that should be reported in the heartbeat
	SourceName   string    // normally follows the format of "type-hostname", e.g. "stdlib-source"
	SourceUUID   uuid.UUID // The UUID of the source, is this is blank it will be auto-generated. This is used in heartbeats and shouldn't be supplied usually"
	App          string    // "https://app.overmind.tech", "The URL of the Overmind app to use"
	APIServerURL string    // The URL of the Overmind API server to uses for the heartbeat, this is calculated

	// The 'ovm_*' API key to use to authenticate to the Overmind API.
	// This and 'SourceAccessToken' are mutually exclusive
	ApiKey string // The API key to use to authenticate to the Overmind API"
	// Static token passed to the source to authenticate.
	SourceAccessToken     string // The access token to use to authenticate to the source
	SourceAccessTokenType string // The type of token to use to authenticate the source for managed sources

	// NATS options
	NATSOptions           *auth.NATSOptions // Options for connecting to NATS
	NATSConnectionTimeout int               // The timeout for connecting to NATS
	NATSQueueName         string            // The name of the queue to use when subscribing
	Unauthenticated       bool              // Whether the source is unauthenticated

	// The options for the heartbeat. If this is nil the engine won't send
	// it is not used if we are nats only or unauthenticated. this will only happen if we are running in a test environment
	HeartbeatOptions *HeartbeatOptions

	// Whether this adapter is managed by Overmind. This is initially used for
	// reporting so that you can tell the difference between managed adapters and
	// ones you're running locally
	OvermindManagedSource sdp.SourceManaged
	MaxParallelExecutions int // 2_000, Max number of requests to run in parallel
}

// Engine is the main discovery engine. This is where all of the Adapters and
// adapters are stored and is responsible for calling out to the right adapters to
// discover everything
//
// Note that an engine that does not have a connected NATS connection will
// simply not communicate over NATS
type Engine struct {
	EngineConfig *EngineConfig
	// The maximum request timeout. Defaults to `DefaultMaxRequestTimeout` if
	// set to zero. If a client does not send a timeout, it will default to this
	// value. Requests with timeouts larger than this value will have their
	// timeouts overridden
	MaxRequestTimeout time.Duration

	// How often to check for closed connections and try to recover
	ConnectionWatchInterval time.Duration
	connectionWatcher       NATSWatcher

	// The configuration for the heartbeat for this engine. If this is nil the
	// engine won't send heartbeats when started

	// Internal throttle used to limit MaxParallelExecutions. This reads
	// MaxParallelExecutions and is populated when the engine is started. This
	// pool is only used for LIST requests. Since GET requests can be blocked by
	// LIST requests, they need to be handled in a different pool to avoid
	// deadlocking.
	listExecutionPool *pool.Pool

	// Internal throttle used to limit MaxParallelExecutions. This reads
	// MaxParallelExecutions and is populated when the engine is started. This
	// pool is only used for GET and SEARCH requests. Since GET requests can be
	// blocked by LIST requests, they need to be handled in a different pool to
	// avoid deadlocking.
	getExecutionPool *pool.Pool

	// The NATS connection
	natsConnection      sdp.EncodedConnection
	natsConnectionMutex sync.Mutex

	// All Adapters managed by this Engine
	sh *AdapterHost

	// handle log requests with this adapter
	logAdapter   LogAdapter
	logAdapterMu sync.RWMutex

	// GetListMutex used for locking out Get queries when there's a List happening
	gfm GetListMutex

	// trackedQueries is used for storing queries that have a UUID so they can
	// be cancelled if required
	trackedQueries      map[uuid.UUID]*QueryTracker
	trackedQueriesMutex sync.RWMutex

	// Prevents the engine being restarted many times in parallel
	restartMutex sync.Mutex

	// Context to background jobs like cache purging and heartbeats. These will
	// stop when the context is cancelled
	backgroundJobContext context.Context
	backgroundJobCancel  context.CancelFunc
	heartbeatCancel      context.CancelFunc
}

func NewEngine(engineConfig *EngineConfig) (*Engine, error) {
	sh := NewAdapterHost()
	return &Engine{
		EngineConfig:            engineConfig,
		MaxRequestTimeout:       DefaultMaxRequestTimeout,
		ConnectionWatchInterval: DefaultConnectionWatchInterval,
		sh:                      sh,
		trackedQueries:          make(map[uuid.UUID]*QueryTracker),
	}, nil
}

// TrackQuery Stores a QueryTracker in the engine so that it can be looked
// up later and cancelled if required. The UUID should be supplied as part of
// the query itself
func (e *Engine) TrackQuery(uuid uuid.UUID, qt *QueryTracker) {
	e.trackedQueriesMutex.Lock()
	defer e.trackedQueriesMutex.Unlock()
	e.trackedQueries[uuid] = qt
}

// GetTrackedQuery Returns the QueryTracker object for a given UUID. This
// tracker can then be used to cancel the query
func (e *Engine) GetTrackedQuery(uuid uuid.UUID) (*QueryTracker, error) {
	e.trackedQueriesMutex.RLock()
	defer e.trackedQueriesMutex.RUnlock()

	if qt, ok := e.trackedQueries[uuid]; ok {
		return qt, nil
	} else {
		return nil, fmt.Errorf("tracker with UUID %x not found", uuid)
	}
}

// DeleteTrackedQuery Deletes a query from tracking
func (e *Engine) DeleteTrackedQuery(uuid [16]byte) {
	e.trackedQueriesMutex.Lock()
	defer e.trackedQueriesMutex.Unlock()
	delete(e.trackedQueries, uuid)
}

// AddAdapters Adds an adapter to this engine
func (e *Engine) AddAdapters(adapters ...Adapter) error {
	return e.sh.AddAdapters(adapters...)
}

// Connect Connects to NATS
func (e *Engine) connect() error {
	if e.EngineConfig.NATSOptions != nil {
		encodedConnection, err := e.EngineConfig.NATSOptions.Connect()
		if err != nil {
			return fmt.Errorf("error connecting to NATS '%+v' : %w", e.EngineConfig.NATSOptions.Servers, err)
		}

		e.natsConnectionMutex.Lock()
		e.natsConnection = encodedConnection
		e.natsConnectionMutex.Unlock()

		// TODO: this could be replaced by setting the various callbacks on the
		// natsConnection and waiting for notification from the underlying
		// connection.
		e.connectionWatcher = NATSWatcher{
			Connection: e.natsConnection,
			FailureHandler: func() {
				go func() {
					if err := e.disconnect(); err != nil {
						log.Error(err)
					}

					if err := e.connect(); err != nil {
						log.Error(err)
					}
				}()
			},
		}
		e.connectionWatcher.Start(e.ConnectionWatchInterval)

		// Wait for the connection to be completed
		err = e.natsConnection.Underlying().FlushTimeout(10 * time.Minute)
		if err != nil {
			return fmt.Errorf("error flushing NATS connection: %w", err)
		}

		log.WithFields(log.Fields{
			"ServerID": e.natsConnection.Underlying().ConnectedServerId(),
			"URL:":     e.natsConnection.Underlying().ConnectedUrl(),
		}).Info("NATS connected")
	}

	if e.natsConnection == nil {
		return errors.New("no NATSOptions struct and no natsConnection provided")
	}

	// Since the underlying query processing logic creates its own spans
	// when it has some real work to do, we are not passing a name to these
	// query handlers so that we don't get spans that are completely empty
	err := e.subscribe("request.all", sdp.NewAsyncRawQueryHandler("", func(ctx context.Context, _ *nats.Msg, i *sdp.Query) {
		e.HandleQuery(ctx, i)
	}))
	if err != nil {
		return fmt.Errorf("error subscribing to request.all: %w", err)
	}

	err = e.subscribe("request.scope.>", sdp.NewAsyncRawQueryHandler("", func(ctx context.Context, m *nats.Msg, i *sdp.Query) {
		e.HandleQuery(ctx, i)
	}))
	if err != nil {
		return fmt.Errorf("error subscribing to request.scope.>: %w", err)
	}

	err = e.subscribe("cancel.all", sdp.NewAsyncRawCancelQueryHandler("CancelQueryHandler", func(ctx context.Context, m *nats.Msg, i *sdp.CancelQuery) {
		e.HandleCancelQuery(ctx, i)
	}))
	if err != nil {
		return fmt.Errorf("error subscribing to cancel.all: %w", err)
	}

	err = e.subscribe("cancel.scope.>", sdp.NewAsyncRawCancelQueryHandler("WildcardCancelQueryHandler", func(ctx context.Context, m *nats.Msg, i *sdp.CancelQuery) {
		e.HandleCancelQuery(ctx, i)
	}))
	if err != nil {
		return fmt.Errorf("error subscribing to cancel.scope.>: %w", err)
	}

	if e.logAdapter != nil {
		for _, scope := range e.logAdapter.Scopes() {
			subj := fmt.Sprintf("logs.scope.%v", scope)
			err = e.subscribe(subj, sdp.NewAsyncRawNATSGetLogRecordsRequestHandler("WildcardCancelQueryHandler", func(ctx context.Context, m *nats.Msg, i *sdp.NATSGetLogRecordsRequest) {
				replyTo := m.Header.Get("reply-to")
				e.HandleLogRecordsRequest(ctx, replyTo, i)
			}))
			if err != nil {
				return fmt.Errorf("error subscribing to %v: %w", subj, err)
			}
		}
	}

	return nil
}

// disconnect Disconnects the engine from the NATS network
func (e *Engine) disconnect() error {
	e.connectionWatcher.Stop()

	e.natsConnectionMutex.Lock()
	defer e.natsConnectionMutex.Unlock()

	if e.natsConnection == nil {
		return nil
	}

	e.natsConnection.Close()
	e.natsConnection.Drop()

	return nil
}

// Start performs all of the initialisation steps required for the engine to
// work. Note that this creates NATS subscriptions for all available adapters so
// modifying the Adapters value after an engine has been started will not have
// any effect until the engine is restarted
func (e *Engine) Start() error {
	e.listExecutionPool = pool.New().WithMaxGoroutines(e.EngineConfig.MaxParallelExecutions)
	e.getExecutionPool = pool.New().WithMaxGoroutines(e.EngineConfig.MaxParallelExecutions)

	e.backgroundJobContext, e.backgroundJobCancel = context.WithCancel(context.Background())

	// Decide your own UUID if not provided
	if e.EngineConfig.SourceUUID == uuid.Nil {
		e.EngineConfig.SourceUUID = uuid.New()
	}

	err := e.connect()
	if err != nil {
		return e.SendHeartbeat(e.backgroundJobContext, err)		
	}

	// Start background jobs
	e.sh.StartPurger(e.backgroundJobContext)
	e.StartSendingHeartbeats(e.backgroundJobContext)
	return nil
}

// subscribe Subscribes to a subject using the current NATS connection.
// Remember to use sdp's genhandler to get a nats.MsgHandler with otel propagation and protobuf marshaling
func (e *Engine) subscribe(subject string, handler nats.MsgHandler) error {
	var err error

	e.natsConnectionMutex.Lock()
	defer e.natsConnectionMutex.Unlock()

	if e.natsConnection.Underlying() == nil {
		return errors.New("cannot subscribe. NATS connection is nil")
	}

	log.WithFields(log.Fields{
		"queueName":  e.EngineConfig.NATSQueueName,
		"subject":    subject,
		"engineName": e.EngineConfig.SourceName,
	}).Debug("creating NATS subscription")

	if e.EngineConfig.NATSQueueName == "" {
		_, err = e.natsConnection.Subscribe(subject, handler)
	} else {
		_, err = e.natsConnection.QueueSubscribe(subject, e.EngineConfig.NATSQueueName, handler)
	}
	if err != nil {
		return fmt.Errorf("error subscribing to NATS: %w", err)
	}

	return nil
}

// Stop Stops the engine running and disconnects from NATS
func (e *Engine) Stop() error {
	err := e.disconnect()
	if err != nil {
		return err
	}

	// Stop purging and clear the cache
	if e.backgroundJobCancel != nil {
		e.backgroundJobCancel()
	}
	if e.heartbeatCancel != nil {
		e.heartbeatCancel()
	}

	e.sh.ClearCaches()

	return nil
}

// Restart Restarts the engine. If called in parallel, subsequent calls are
// ignored until the restart is completed
func (e *Engine) Restart() error {
	e.restartMutex.Lock()
	defer e.restartMutex.Unlock()

	err := e.Stop()
	if err != nil {
		return fmt.Errorf("Restart.Stop: %w", err)
	}

	err = e.Start()
	return fmt.Errorf("Restart.Start: %w", err)
}

// IsNATSConnected returns whether the engine is connected to NATS
func (e *Engine) IsNATSConnected() bool {
	e.natsConnectionMutex.Lock()
	defer e.natsConnectionMutex.Unlock()

	if e.natsConnection == nil {
		return false
	}

	if conn := e.natsConnection.Underlying(); conn != nil {
		return conn.IsConnected()
	}

	return false
}

// HealthCheck returns an error if the Engine is not healthy. Call this inside
// an opentelemetry span to capture default metrics from the engine.
func (e *Engine) HealthCheck(ctx context.Context) error {
	span := trace.SpanFromContext(ctx)

	natsConnected := e.IsNATSConnected()

	span.SetAttributes(
		attribute.String("ovm.engine.name", e.EngineConfig.SourceName),
		attribute.Bool("ovm.nats.connected", natsConnected),
		attribute.Int("ovm.discovery.listExecutionPoolCount", int(listExecutionPoolCount.Load())),
		attribute.Int("ovm.discovery.getExecutionPoolCount", int(getExecutionPoolCount.Load())),
	)

	if e.natsConnection.Underlying() != nil {
		u := e.natsConnection.Underlying()
		span.SetAttributes(
			attribute.String("ovm.nats.serverId", u.ConnectedServerId()),
			attribute.String("ovm.nats.url", u.ConnectedUrl()),
			attribute.Int64("ovm.nats.reconnects", int64(u.Reconnects)), //nolint:gosec // Reconnects is always a small positive number
		)
	}

	if !natsConnected {
		return errors.New("NATS connection is not connected")
	}

	return nil
}

// HandleCancelQuery Takes a CancelQuery and cancels that query if it exists
func (e *Engine) HandleCancelQuery(ctx context.Context, cancelQuery *sdp.CancelQuery) {
	span := trace.SpanFromContext(ctx)
	span.SetName("HandleCancelQuery")

	u, err := uuid.FromBytes(cancelQuery.GetUUID())
	if err != nil {
		log.Errorf("Error parsing UUID for cancel query: %v", err)
		return
	}

	rt, err := e.GetTrackedQuery(u)
	if err != nil {
		log.Debugf("Could not find tracked query %v. Possibly it has already finished", u.String())
		return
	}

	if rt != nil && rt.Cancel != nil {
		log.WithFields(log.Fields{
			"UUID": u.String(),
		}).Debug("Cancelling query")
		rt.Cancel()
	}
}

func (e *Engine) HandleLogRecordsRequest(ctx context.Context, replyTo string, request *sdp.NATSGetLogRecordsRequest) {
	span := trace.SpanFromContext(ctx)
	span.SetName("HandleLogRecordsRequest")

	if !strings.HasPrefix(replyTo, "logs.records.") {
		sentry.CaptureException(fmt.Errorf("received log records request with invalid reply-to header: %s", replyTo))
		return
	}

	err := e.natsConnection.Publish(ctx, replyTo, &sdp.NATSGetLogRecordsResponse{
		Content: &sdp.NATSGetLogRecordsResponse_Status{
			Status: &sdp.NATSGetLogRecordsResponseStatus{
				Status: sdp.NATSGetLogRecordsResponseStatus_STARTED,
			},
		},
	})
	if err != nil {
		sentry.CaptureException(fmt.Errorf("error publishing log records STARTED response: %w", err))
		return
	}

	// ensure that we send an error response if the HandleLogRecordsRequestWithErrors call panics
	defer func() {
		if r := recover(); r != nil {
			sentry.CaptureException(fmt.Errorf("panic in log records request handler: %v", r))
			err = e.natsConnection.Publish(ctx, replyTo, &sdp.NATSGetLogRecordsResponse{
				Content: &sdp.NATSGetLogRecordsResponse_Status{
					Status: &sdp.NATSGetLogRecordsResponseStatus{
						Status: sdp.NATSGetLogRecordsResponseStatus_ERRORED,
						Error:  sdp.NewLocalSourceError(connect.CodeInternal, "panic in log records request handler"),
					},
				},
			})
			if err != nil {
				sentry.CaptureException(fmt.Errorf("error publishing log records FINISHED response: %w", err))
				return
			}
		}
	}()

	srcErr := e.HandleLogRecordsRequestWithErrors(ctx, replyTo, request)
	if srcErr != nil {
		err = e.natsConnection.Publish(ctx, replyTo, &sdp.NATSGetLogRecordsResponse{
			Content: &sdp.NATSGetLogRecordsResponse_Status{
				Status: &sdp.NATSGetLogRecordsResponseStatus{
					Status: sdp.NATSGetLogRecordsResponseStatus_ERRORED,
					Error:  srcErr,
				},
			},
		})
		if err != nil {
			sentry.CaptureException(fmt.Errorf("error publishing log records FINISHED response: %w", err))
			return
		}
		return
	}

	err = e.natsConnection.Publish(ctx, replyTo, &sdp.NATSGetLogRecordsResponse{
		Content: &sdp.NATSGetLogRecordsResponse_Status{
			Status: &sdp.NATSGetLogRecordsResponseStatus{
				Status: sdp.NATSGetLogRecordsResponseStatus_FINISHED,
			},
		},
	})
	if err != nil {
		sentry.CaptureException(fmt.Errorf("error publishing log records FINISHED response: %w", err))
		return
	}
}

func (e *Engine) HandleLogRecordsRequestWithErrors(ctx context.Context, replyTo string, natsRequest *sdp.NATSGetLogRecordsRequest) *sdp.SourceError {
	if e.logAdapter == nil {
		return sdp.NewLocalSourceError(connect.CodeInvalidArgument, "no logs adapter registered")
	}

	if natsRequest == nil {
		return sdp.NewLocalSourceError(connect.CodeInvalidArgument, "received nil log records request")
	}

	req := natsRequest.GetRequest()
	if req == nil {
		return sdp.NewLocalSourceError(connect.CodeInvalidArgument, "received nil log records request body")
	}

	err := req.Validate()
	if err != nil {
		return sdp.NewLocalSourceError(connect.CodeInvalidArgument, fmt.Sprintf("invalid log records request: %v", err))
	}

	if !slices.Contains(e.logAdapter.Scopes(), req.GetScope()) {
		return sdp.NewLocalSourceError(connect.CodeInvalidArgument, fmt.Sprintf("scope %s is not available", req.GetScope()))
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("ovm.logs.replyTo", replyTo),
		attribute.String("ovm.logs.scope", req.GetScope()),
		attribute.String("ovm.logs.query", req.GetQuery()),
		attribute.String("ovm.logs.from", req.GetFrom().String()),
		attribute.String("ovm.logs.to", req.GetTo().String()),
		attribute.Int("ovm.logs.maxRecords", int(req.GetMaxRecords())),
		attribute.Bool("ovm.logs.startFromOldest", req.GetStartFromOldest()),
	)

	stream := &LogRecordsStreamImpl{
		subject: replyTo,
		stream:  e.natsConnection,
	}
	err = e.logAdapter.Get(ctx, req, stream)

	span.SetAttributes(
		attribute.Int("ovm.logs.numResponses", stream.responses),
		attribute.Int("ovm.logs.numRecords", stream.records),
	)
	srcErr := &sdp.SourceError{}
	if errors.As(err, &srcErr) {
		return srcErr
	}
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		return sdp.NewLocalSourceError(connect.CodeDeadlineExceeded, "log records request deadline exceeded")
	}
	if err != nil {
		return sdp.NewLocalSourceError(connect.CodeInternal, fmt.Sprintf("error handling log records request: %v", err))
	}

	return nil
}

// ClearCache Completely clears the cache
func (e *Engine) ClearCache() {
	e.sh.ClearCaches()
}

// ClearAdapters Deletes all adapters from the engine, allowing new adapters to be
// added using `AddAdapter()`. Note that this requires a restart using
// `Restart()` in order to take effect
func (e *Engine) ClearAdapters() {
	e.sh.ClearAllAdapters()
}

// IsWildcard checks if a string is the wildcard. Use this instead of
// implementing the wildcard check everywhere so that if we need to change the
// wildcard at a later date we can do so here
func IsWildcard(s string) bool {
	return s == sdp.WILDCARD
}

// SetLogAdapter registers a single LogAdapter with the engine.
// Returns an error when there is already a log adapter registered.
func (e *Engine) SetLogAdapter(adapter LogAdapter) error {
	if adapter == nil {
		return errors.New("log adapter cannot be nil")
	}

	e.logAdapterMu.Lock()
	defer e.logAdapterMu.Unlock()

	if e.logAdapter != nil {
		return errors.New("log adapter already registered")
	}

	e.logAdapter = adapter
	return nil
}

// GetAvailableScopesAndMetadata returns the available scopes and adapter metadata
// from all visible adapters. This is useful for heartbeats and other reporting.
func (e *Engine) GetAvailableScopesAndMetadata() ([]string, []*sdp.AdapterMetadata) {
	// Get available types and scopes
	availableScopesMap := map[string]bool{}
	adapterMetadata := []*sdp.AdapterMetadata{}

	for _, adapter := range e.sh.VisibleAdapters() {
		for _, scope := range adapter.Scopes() {
			availableScopesMap[scope] = true
		}
		adapterMetadata = append(adapterMetadata, adapter.Metadata())
	}

	// Extract slices from maps
	availableScopes := []string{}
	for s := range availableScopesMap {
		availableScopes = append(availableScopes, s)
	}

	return availableScopes, adapterMetadata
}
