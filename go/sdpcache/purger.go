package sdpcache

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// MinWaitDefault is the default minimum wait time between purge cycles.
const MinWaitDefault = 5 * time.Second

// PurgeStats holds statistics from a single purge run.
type PurgeStats struct {
	// How many items were timed out of the cache
	NumPurged int
	// How long purging took overall
	TimeTaken time.Duration
	// The expiry time of the next item to expire. If there are no more items in
	// the cache, this will be nil
	NextExpiry *time.Time
}

// purger manages timer-based scheduling for periodic cache purging.
// MemoryCache and boltStore embed this struct to share the scheduling logic;
// storage-specific purge work is injected via the purgeFunc callback.
type purger struct {
	purgeFunc   func(context.Context, time.Time) PurgeStats
	minWaitTime time.Duration
	purgeTimer  *time.Timer
	nextPurge   time.Time
	purgeMutex  sync.Mutex
}

// GetMinWaitTime returns the minimum wait time or the default if not set.
func (p *purger) GetMinWaitTime() time.Duration {
	if p.minWaitTime == 0 {
		return MinWaitDefault
	}
	return p.minWaitTime
}

// StartPurger starts the purge process in the background, it will be cancelled
// when the context is cancelled. The cache will be purged initially, at which
// point the process will sleep until the next time an item expires.
func (p *purger) StartPurger(ctx context.Context) {
	p.purgeMutex.Lock()
	if p.purgeTimer == nil {
		p.purgeTimer = time.NewTimer(0)
		p.purgeMutex.Unlock()
	} else {
		p.purgeMutex.Unlock()
		log.WithContext(ctx).Info("Purger already running")
		return
	}

	go func(ctx context.Context) {
		for {
			select {
			case <-p.purgeTimer.C:
				stats := p.purgeFunc(ctx, time.Now())
				p.setNextPurgeFromStats(stats)
			case <-ctx.Done():
				p.purgeMutex.Lock()
				defer p.purgeMutex.Unlock()

				p.purgeTimer.Stop()
				p.purgeTimer = nil
				return
			}
		}
	}(ctx)
}

// setNextPurgeFromStats sets when the next purge should run based on the stats
// of the previous purge.
func (p *purger) setNextPurgeFromStats(stats PurgeStats) {
	p.purgeMutex.Lock()
	defer p.purgeMutex.Unlock()

	if stats.NextExpiry == nil {
		p.purgeTimer.Reset(1000 * time.Hour)
		p.nextPurge = time.Now().Add(1000 * time.Hour)
	} else {
		if time.Until(*stats.NextExpiry) < p.GetMinWaitTime() {
			p.purgeTimer.Reset(p.GetMinWaitTime())
			p.nextPurge = time.Now().Add(p.GetMinWaitTime())
		} else {
			p.purgeTimer.Reset(time.Until(*stats.NextExpiry))
			p.nextPurge = *stats.NextExpiry
		}
	}
}

// setNextPurgeIfEarlier sets the next time the purger will run, if the provided
// time is sooner than the current scheduled purge time. While the purger is
// active this will be constantly updated, however if the purger is sleeping and
// new items are added this method ensures that the purger is woken up.
func (p *purger) setNextPurgeIfEarlier(t time.Time) {
	p.purgeMutex.Lock()
	defer p.purgeMutex.Unlock()

	if t.Before(p.nextPurge) {
		if p.purgeTimer == nil {
			return
		}

		p.purgeTimer.Stop()
		p.nextPurge = t
		p.purgeTimer.Reset(time.Until(t))
	}
}
