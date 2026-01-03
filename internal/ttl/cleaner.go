package ttl

import (
	"context"
	"time"

	"distributed-cache/internal/logs"
	"distributed-cache/internal/metrics"
)

// Store defines the minimal contract required by the TTL cleaner.
type Store interface {
	RemoveExpired() int
}

// Cleaner periodically removes expired keys from the store.
type Cleaner struct {
	store    Store
	interval time.Duration
	logger   *logs.Logger
	metrics  *metrics.Registry
}

// NewCleaner creates a new TTL cleaner.
func NewCleaner(
	store Store,
	interval time.Duration,
	logger *logs.Logger,
	metricsRegistry *metrics.Registry,
) *Cleaner {
	return &Cleaner{
		store:    store,
		interval: interval,
		logger:   logger,
		metrics:  metricsRegistry,
	}
}

// Start runs the cleanup loop until the context is cancelled.
func (c *Cleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.metrics.Inc(metrics.TTLCleanupRunsTotal)
			c.runOnce()
		case <-ctx.Done():
			c.logger.Debug("ttl cleaner stopped")
			return
		}
	}
}

// runOnce performs a single cleanup cycle.
func (c *Cleaner) runOnce() {
	removed := c.store.RemoveExpired()
	if removed > 0 {
		c.metrics.Add(metrics.TTLKeysRemovedTotal, int64(removed))
		c.logger.Info("ttl cleaner removed expired keys")
	}
}
