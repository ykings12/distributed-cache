package metrics

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistry_IncAndAdd(t *testing.T) {
	r := NewRegistry()

	r.Inc(CacheSetsTotal)
	r.Add(CacheSetsTotal, 2)

	snap := r.Snapshot()
	assert.Equal(t, int64(3), snap[string(CacheSetsTotal)])
}

func TestRegistry_MultipleMetrics(t *testing.T) {
	r := NewRegistry()

	r.Inc(CacheGetsTotal)
	r.Inc(CacheMissesTotal)
	r.Add(TTLKeysRemovedTotal, 5)

	snap := r.Snapshot()

	assert.Equal(t, int64(1), snap[string(CacheGetsTotal)])
	assert.Equal(t, int64(1), snap[string(CacheMissesTotal)])
	assert.Equal(t, int64(5), snap[string(TTLKeysRemovedTotal)])
}

func TestRegistry_ConcurrentUpdates(t *testing.T) {
	r := NewRegistry()
	wg := sync.WaitGroup{}

	workers := 50
	increments := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				r.Inc(ReplicationAttemptsTotal)
			}
		}()
	}

	wg.Wait()

	snap := r.Snapshot()
	assert.Equal(t, int64(workers*increments), snap[string(ReplicationAttemptsTotal)])
}

func TestRegistry_SnapshotIsDeepCopy(t *testing.T) {
	r := NewRegistry()

	r.Inc(CacheKeysTotal)
	snap1 := r.Snapshot()

	// Mutate snapshot
	snap1[string(CacheKeysTotal)] = 999

	// Fetch fresh snapshot
	snap2 := r.Snapshot()

	assert.Equal(t, int64(1), snap2[string(CacheKeysTotal)],
		"internal state should not be affected by snapshot mutation")
}

func TestRegistry_UnknownMetricHandledGracefully(t *testing.T) {
	r := NewRegistry()

	r.Inc("unknown_metric")

	snap := r.Snapshot()
	assert.Equal(t, int64(1), snap["unknown_metric"])
}
