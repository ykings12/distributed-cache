package metrics

import "sync/atomic"

// Snapshot returns a deep copy of all metrics.
// Safe for concurrent use and immune to external mutation.
func (r *Registry) Snapshot() map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[string]int64, len(r.counters))
	for key, ptr := range r.counters {
		out[string(key)] = atomic.LoadInt64(ptr)
	}
	return out
}
