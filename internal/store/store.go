package store

import (
	"sync"
	"time"

	"distributed-cache/internal/metrics"
)

// Store is a concurrency-safe in-memory key–value store.
//
// Design principles:
// - Safe for concurrent access using RWMutex
// - Uses Last-Write-Wins (LWW) via logical timestamps
// - TTL expiration handled using wall-clock time (time.Now)
//
// Note:
// TTL testing uses short sleeps instead of injecting a clock,
// keeping the store free of test-only concerns.
type Store struct {
	mu      sync.RWMutex
	data    map[string]Entry
	metrics *metrics.Registry
}

// NewStore initializes and returns a new Store.
func NewStore(metricsRegistry *metrics.Registry) *Store {
	return &Store{
		data:    make(map[string]Entry),
		metrics: metricsRegistry,
	}
}

// Set inserts or updates a key using Last-Write-Wins semantics.
//
// Rules:
// - If the key does not exist, insert it.
// - If the key exists, overwrite only if the incoming timestamp is newer.
func (s *Store) Set(key string, entry Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.Inc(metrics.CacheSetsTotal)

	existing, exists := s.data[key]
	if exists && entry.Timestamp <= existing.Timestamp {
		return
	}

	if !exists {
		s.metrics.Inc(metrics.CacheKeysTotal)
	}

	s.data[key] = entry
}

// Get retrieves a value from the store.
//
// Behavior:
// - Returns (value, true) if key exists and is not expired
// - If the key is expired, it is deleted and treated as missing
func (s *Store) Get(key string) (string, bool) {
	s.metrics.Inc(metrics.CacheGetsTotal)

	s.mu.RLock()
	entry, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		s.metrics.Inc(metrics.CacheMissesTotal)
		return "", false
	}

	if entry.IsExpired(time.Now()) {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()

		s.metrics.Inc(metrics.CacheExpiredTotal)
		s.metrics.Add(metrics.CacheKeysTotal, -1)

		return "", false
	}

	return entry.Value, true
}

// Delete removes a key from the store.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[key]; ok {
		delete(s.data, key)
		s.metrics.Add(metrics.CacheKeysTotal, -1)
	}
}

// List returns a snapshot of all non-expired entries.
// Used by admin APIs and UI.
func (s *Store) List() map[string]Entry {
	now := time.Now()
	result := make(map[string]Entry)

	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range s.data {
		if !v.IsExpired(now) {
			result[k] = v
		}
	}
	return result
}

// RemoveExpired removes all expired keys from the store.
//
// This will be used by the background TTL cleaner.
func (s *Store) RemoveExpired() int {
	now := time.Now()
	removed := 0

	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range s.data {
		if v.IsExpired(now) {
			delete(s.data, k)
			removed++
		}
	}

	if removed > 0 {
		s.metrics.Add(metrics.CacheExpiredTotal, int64(removed))
		s.metrics.Add(metrics.CacheKeysTotal, -int64(removed))
	}

	return removed
}
