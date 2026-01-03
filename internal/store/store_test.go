package store

import (
	"sync"
	"testing"
	"time"

	"distributed-cache/internal/metrics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreGet_Set(t *testing.T) {
	store := NewStore(metrics.NewRegistry())

	t.Run("set and get existing key", func(t *testing.T) {
		store.Set("key1", Entry{
			Value:     "hello",
			Timestamp: 1,
		})

		val, ok := store.Get("key1")
		require.True(t, ok)
		assert.Equal(t, "hello", val)
	})

	t.Run("get non-existing key", func(t *testing.T) {
		_, ok := store.Get("missing")
		assert.False(t, ok)
	})
}

func TestStoreDelete(t *testing.T) {
	store := NewStore(metrics.NewRegistry())

	store.Set("key1", Entry{
		Value:     "1",
		Timestamp: 1,
	})

	store.Delete("key1")

	_, ok := store.Get("key1")
	assert.False(t, ok)
}

func TestStoreLastWriteWins(t *testing.T) {
	store := NewStore(metrics.NewRegistry())

	store.Set("key1", Entry{
		Value:     "old",
		Timestamp: 1,
	})

	val, ok := store.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "old", val)

	store.Set("key1", Entry{
		Value:     "new",
		Timestamp: 2,
	})

	val, _ = store.Get("key1")
	assert.Equal(t, "new", val)
}

func TestStoreConcurrentWrites(t *testing.T) {
	store := NewStore(metrics.NewRegistry())

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(ts int64) {
			defer wg.Done()
			store.Set("key", Entry{
				Value:     "value",
				Timestamp: ts,
			})
		}(int64(i))
	}

	wg.Wait()

	_, ok := store.Get("key")
	assert.True(t, ok)
}

func TestStoreRemoveExpired(t *testing.T) {
	store := NewStore(metrics.NewRegistry())

	store.Set("k1", Entry{
		Value:     "v1",
		Timestamp: 1,
		ExpiresAt: time.Now().Add(-time.Second),
	})

	store.Set("k2", Entry{
		Value:     "v2",
		Timestamp: 2,
	})

	removed := store.RemoveExpired()
	assert.Equal(t, 1, removed)

	_, ok := store.Get("k1")
	assert.False(t, ok)

	_, ok = store.Get("k2")
	assert.True(t, ok)
}

func TestStoreList_FiltersExpiredKeys(t *testing.T) {
	store := NewStore(metrics.NewRegistry())

	store.Set("alive", Entry{
		Value:     "ok",
		Timestamp: 1,
		ExpiresAt: time.Now().Add(time.Second),
	})

	store.Set("expired", Entry{
		Value:     "gone",
		Timestamp: 2,
		ExpiresAt: time.Now().Add(-time.Second),
	})

	result := store.List()

	_, okAlive := result["alive"]
	_, okExpired := result["expired"]

	assert.True(t, okAlive, "non-expired key should be listed")
	assert.False(t, okExpired, "expired key should not be listed")
}

func TestStoreGet_ExpiredKeyIsDeleted(t *testing.T) {
	reg := metrics.NewRegistry()
	store := NewStore(reg)

	store.Set("temp", Entry{
		Value:     "value",
		Timestamp: 1,
		ExpiresAt: time.Now().Add(-time.Millisecond),
	})

	// Call Get → should trigger expiration path
	val, ok := store.Get("temp")

	assert.False(t, ok)
	assert.Equal(t, "", val)

	// Ensure key was deleted
	_, ok = store.Get("temp")
	assert.False(t, ok)

	// Verify metrics side-effects
	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.CacheExpiredTotal)])
	assert.Equal(t, int64(0), snap[string(metrics.CacheKeysTotal)])
}
