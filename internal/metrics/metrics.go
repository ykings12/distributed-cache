package metrics

import (
	"sync"
	"sync/atomic"
)

// MetricKey is a strongly typed metric identifier.
type MetricKey string

// Metric keys (centralized)
const (
	// Cache
	CacheKeysTotal    MetricKey = "cache_keys_total"
	CacheSetsTotal    MetricKey = "cache_sets_total"
	CacheGetsTotal    MetricKey = "cache_gets_total"
	CacheMissesTotal  MetricKey = "cache_misses_total"
	CacheExpiredTotal MetricKey = "cache_expired_total"

	// Replication
	ReplicationAttemptsTotal MetricKey = "replication_attempts_total"
	ReplicationSuccessTotal  MetricKey = "replication_success_total"
	ReplicationFailureTotal  MetricKey = "replication_failure_total"
	ReplicationRetriesTotal  MetricKey = "replication_retries_total"

	// TTL
	TTLCleanupRunsTotal MetricKey = "ttl_cleanup_runs_total"
	TTLKeysRemovedTotal MetricKey = "ttl_keys_removed_total"

	// Peers
	PeersHealthy      MetricKey = "peers_healthy"
	PeersUnhealthy    MetricKey = "peers_unhealthy"
	PeerFailuresTotal MetricKey = "peer_failures_total"

	// Heartbeat metrics
	HeartbeatRunsTotal     MetricKey = "heartbeat_runs_total"
	HeartbeatSuccessTotal  MetricKey = "heartbeat_success_total"
	HeartbeatFailuresTotal MetricKey = "heartbeat_failures_total"
)

// Registry stores all metrics.
type Registry struct {
	mu       sync.RWMutex
	counters map[MetricKey]*int64
}

// NewRegistry creates a metrics registry.
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[MetricKey]*int64),
	}
}

// Inc increments a metric by 1.
func (r *Registry) Inc(key MetricKey) {
	r.Add(key, 1)
}

// Add increments a metric by delta.
func (r *Registry) Add(key MetricKey, delta int64) {
	r.mu.RLock()
	ptr, ok := r.counters[key]
	r.mu.RUnlock()

	if ok {
		atomic.AddInt64(ptr, delta)
		return
	}

	// Slow path: metric not yet initialized
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if ptr, ok = r.counters[key]; ok {
		atomic.AddInt64(ptr, delta)
		return
	}

	var val int64
	r.counters[key] = &val
	atomic.AddInt64(&val, delta)
}
