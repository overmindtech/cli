package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

// WatchableConnection Is ususally a *nats.Conn, we are using an interface here
// to allow easier testing
type WatchableConnection interface {
	Status() nats.Status
	Stats() nats.Statistics
	LastError() error
}

type NATSWatcher struct {
	// Connection The NATS connection to watch
	Connection WatchableConnection

	// FailureHandler will be called when the connection has been closed and is
	// no longer trying to reconnect, or when the connection has been in a
	// non-CONNECTED state for longer than ReconnectionTimeout.
	FailureHandler func()

	// ReconnectionTimeout is the maximum duration to wait for a reconnection
	// before triggering the FailureHandler. If set to 0, no timeout is applied
	// and the watcher only triggers on CLOSED status (legacy behavior).
	// Recommended value: 5 minutes.
	ReconnectionTimeout time.Duration

	watcherContext          context.Context
	watcherCancel           context.CancelFunc
	watcherTicker           *time.Ticker
	watchingMutex           sync.Mutex
	disconnectedSince       time.Time
	hasBeenDisconnected     bool
	failureHandlerTriggered bool
}

func (w *NATSWatcher) Start(checkInterval time.Duration) {
	if w == nil || w.Connection == nil {
		return
	}

	w.watcherContext, w.watcherCancel = context.WithCancel(context.Background())
	w.watcherTicker = time.NewTicker(checkInterval)
	w.watchingMutex.Lock()

	go func(ctx context.Context) {
		defer w.watchingMutex.Unlock()
		for {
			select {
			case <-w.watcherTicker.C:
				status := w.Connection.Status()
				if status != nats.CONNECTED {
					// Track when we first became disconnected
					if !w.hasBeenDisconnected {
						w.disconnectedSince = time.Now()
						w.hasBeenDisconnected = true
						w.failureHandlerTriggered = false
					}

					disconnectedDuration := time.Since(w.disconnectedSince)

					log.WithFields(log.Fields{
						"status":               status.String(),
						"inBytes":              w.Connection.Stats().InBytes,
						"outBytes":             w.Connection.Stats().OutBytes,
						"reconnects":           w.Connection.Stats().Reconnects,
						"lastError":            w.Connection.LastError(),
						"disconnectedDuration": disconnectedDuration.String(),
					}).Warn("NATS not connected")

					// Trigger failure handler if connection is CLOSED (won't retry)
					// or if we've been disconnected for too long. Only trigger once
					// per disconnection period to avoid repeated calls while the
					// handler is working on reconnection.
					if !w.failureHandlerTriggered {
						shouldTriggerFailure := false
						if status == nats.CLOSED {
							log.Warn("NATS connection is CLOSED, triggering failure handler")
							shouldTriggerFailure = true
						} else if w.ReconnectionTimeout > 0 && disconnectedDuration > w.ReconnectionTimeout {
							log.WithFields(log.Fields{
								"disconnectedDuration": disconnectedDuration.String(),
								"reconnectionTimeout":  w.ReconnectionTimeout.String(),
							}).Error("NATS connection has been disconnected for too long, triggering failure handler")
							shouldTriggerFailure = true
						}

						if shouldTriggerFailure {
							// Mark that we've triggered the handler for this disconnection
							// period to prevent repeated calls
							w.failureHandlerTriggered = true
							w.FailureHandler()
						}
					}
				} else {
					// Reset the disconnection tracking when we're connected
					w.hasBeenDisconnected = false
					w.failureHandlerTriggered = false
				}
			case <-ctx.Done():
				w.watcherTicker.Stop()

				return
			}
		}
	}(w.watcherContext)
}

func (w *NATSWatcher) Stop() {
	if w.watcherCancel != nil {
		w.watcherCancel()

		// Once we have sent the signal, wait until it's unlocked so we know
		// it's completely stopped
		w.watchingMutex.Lock()
		defer w.watchingMutex.Unlock()

	}
}
