package ttl

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"distributed-cache/internal/logs"
	"distributed-cache/internal/metrics"

	"github.com/stretchr/testify/assert"
)

/* ---------------- Mock Store ---------------- */

type mockStore struct {
	removed int32
}

func (m *mockStore) RemoveExpired() int {
	return int(atomic.AddInt32(&m.removed, 1))
}

/* ---------------- Tests ---------------- */

func TestCleaner_RunOnce_RemovesExpiredAndUpdatesMetrics(t *testing.T) {
	store := &mockStore{}
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	cleaner := NewCleaner(store, time.Second, logger, reg)

	cleaner.runOnce()

	assert.Equal(t, int32(1), atomic.LoadInt32(&store.removed))

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.TTLKeysRemovedTotal)])
}

func TestCleaner_Start_RunsPeriodicallyAndTracksRuns(t *testing.T) {
	store := &mockStore{}
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	cleaner := NewCleaner(store, 5*time.Millisecond, logger, reg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cleaner.Start(ctx)

	assert.Eventually(t, func() bool {
		snap := reg.Snapshot()
		return snap[string(metrics.TTLCleanupRunsTotal)] >= 2
	}, 100*time.Millisecond, 5*time.Millisecond)
}

func TestCleaner_Start_StopsOnContextCancel(t *testing.T) {
	store := &mockStore{}
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(10, logs.DEBUG)

	cleaner := NewCleaner(store, 5*time.Millisecond, logger, reg)

	ctx, cancel := context.WithCancel(context.Background())
	go cleaner.Start(ctx)

	time.Sleep(20 * time.Millisecond)
	cancel()

	runsAtCancel := reg.Snapshot()[string(metrics.TTLCleanupRunsTotal)]

	time.Sleep(30 * time.Millisecond)
	runsAfter := reg.Snapshot()[string(metrics.TTLCleanupRunsTotal)]

	// Allow at most one extra tick due to race with ticker
	assert.LessOrEqual(t, runsAfter, runsAtCancel+1)
}
