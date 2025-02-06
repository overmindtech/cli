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
	// no longer trying to reconnect.
	FailureHandler func()

	watcherContext context.Context
	watcherCancel  context.CancelFunc
	watcherTicker  *time.Ticker
	watchingMutex  sync.Mutex
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
				if w.Connection.Status() != nats.CONNECTED {
					log.WithFields(log.Fields{
						"status":     w.Connection.Status().String(),
						"inBytes":    w.Connection.Stats().InBytes,
						"outBytes":   w.Connection.Stats().OutBytes,
						"reconnects": w.Connection.Stats().Reconnects,
						"lastError":  w.Connection.LastError(),
					}).Warn("NATS not connected")

					if w.Connection.Status() == nats.CLOSED {
						// If the connection is closed this means that it won't
						// try to reconnect.
						w.FailureHandler()
					}
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
