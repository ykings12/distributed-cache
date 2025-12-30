package ttl

import (
	"context"
	"distributed-cache/internal/logs"
	"time"
)

// Store defines the minimal contract required by the TTL cleaner
// This keeps the cleaner interface decoupled from the concrete store implementation
type Store interface {
	RemoveExpired() int
}

// Cleaner periodically removes expired keys from the store
type Cleaner struct {
	store    Store
	interval time.Duration
	logger   *logs.Logger
}

// NewCleaner creates a new instance of TTL Cleaner
func NewCleaner(
	store Store,
	interval time.Duration,
	logger *logs.Logger,
) *Cleaner {
	return &Cleaner{
		store:    store,
		interval: interval,
		logger:   logger,
	}
}

// Start runs the cleanup loop until the context is cancelled.
// It blocks and should typically be run in a separate goroutine.
func (c *Cleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.runOnce()
		case <-ctx.Done():
			c.logger.Debug("ttl cleaner stopped")
			return
		}
	}
}

// runOnce performs a single cleanup cycle
func (c *Cleaner) runOnce() {
	removed := c.store.RemoveExpired()
	if removed > 0 {
		c.logger.Info("ttl cleaner removed expired keys")
	}
}
