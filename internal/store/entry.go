package store

import "time"

// Entry represents a single value stored in the cache.
//
// Design choices:
// - Timestamp is used for Last-Write-Wins (LWW) conflict resolution.
// - ExpiresAt enables TTL-based expiration.
// - Zero value of ExpiresAt means "no expiration".
type Entry struct {
	Value     string
	Timestamp int64
	ExpiresAt time.Time
}

// IsExpired checks whether the entry is expired at the given time.
func (e Entry) IsExpired(now time.Time) bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return now.After(e.ExpiresAt)

}
