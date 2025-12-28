package store

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreGet_Set(t *testing.T) {
	store := NewStore()

	t.Run("set and get exisiting key", func(t *testing.T) {
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
	store := NewStore()
	store.Set("key1", Entry{
		Value:     "1",
		Timestamp: 1})

	store.Delete("key1")

	_, ok := store.Get("key1")
	assert.False(t, ok)
}

func TestStoreLastWriteWins(t *testing.T) {
	store := NewStore()
	store.Set("key1", Entry{
		Value:     "old",
		Timestamp: 1,
	})

	val, ok := store.Get("key1")
	require.True(t, ok)
	assert.Equal(t, val, "old")

	store.Set("key1", Entry{Value: "new", Timestamp: 2})

	val, _ = store.Get("key1")
	assert.Equal(t, "new", val)
}

func TestStoreConcurrentWrites(t *testing.T) {
	store := NewStore()

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
	store := NewStore()

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
