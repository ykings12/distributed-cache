package ttl

import (
	"context"
	"distributed-cache/internal/logs"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockStore struct {
	removed int32
}

func (m *mockStore) RemoveExpired() int {
	return int(atomic.AddInt32(&m.removed, 1))
}

func TestCleanerRunOnceImplementation(t *testing.T) {
	store := &mockStore{}
	logger := logs.NewLogger(10, logs.DEBUG)

	cleaner := NewCleaner(store, time.Second, logger)
	cleaner.runOnce()
	assert.Equal(t, int32(1), atomic.LoadInt32(&store.removed))
}

func TestCleanerStartRunsPeriodically(t *testing.T) {
	store := &mockStore{}
	logger := logs.NewLogger(10, logs.DEBUG)

	cleaner := NewCleaner(store, 5*time.Millisecond, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go cleaner.Start(ctx)

	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&store.removed) >= 2
	}, 100*time.Millisecond, 5*time.Millisecond)
}

func TestCleaner_Start_StopsOnContextCancel(t *testing.T) {
	store := &mockStore{}
	logger := logs.NewLogger(10, logs.DEBUG)

	cleaner := NewCleaner(store, 5*time.Millisecond, logger)

	ctx, cancel := context.WithCancel(context.Background())
	go cleaner.Start(ctx)

	// Allow cleaner to run a few times
	time.Sleep(20 * time.Millisecond)

	cancel() // request shutdown

	removedAtCancel := atomic.LoadInt32(&store.removed)

	// Give enough time that more ticks *would* have happened
	time.Sleep(50 * time.Millisecond)

	removedAfter := atomic.LoadInt32(&store.removed)

	// Allow at most ONE additional run due to race with ticker
	assert.LessOrEqual(
		t,
		removedAfter,
		removedAtCancel+1,
		"cleaner should stop shortly after context cancellation",
	)
}
