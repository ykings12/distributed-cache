package replication

import (
	"distributed-cache/internal/logs"
	"distributed-cache/internal/store"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReplicator(t *testing.T) {
	// Helper to create a fresh logger for each subtest
	newTestLogger := func() *logs.Logger {
		return logs.NewLogger(100, logs.DEBUG)
	}

	t.Run("SendsReplicationRequest", func(t *testing.T) {
		logger := newTestLogger()
		var received bool
		peerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = true
			w.WriteHeader(http.StatusNoContent)
		}))
		defer peerServer.Close()

		replicator := NewReplicator("node-A", []string{peerServer.URL}, logger)
		replicator.Replicate("key1", store.Entry{Value: "hello", Timestamp: 123})

		assert.Eventually(t, func() bool { return received }, 1*time.Second, 10*time.Millisecond)
	})

	t.Run("NoPeers", func(t *testing.T) {
		logger := newTestLogger() // Fresh logger!
		replicator := NewReplicator("node-A", []string{}, logger)
		replicator.Replicate("key2", store.Entry{Value: "world"})

		assert.Len(t, logger.GetLast(10), 0)
	})

	t.Run("PeerReturns500", func(t *testing.T) {
		logger := newTestLogger()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		replicator := NewReplicator("node-A", []string{server.URL}, logger)
		replicator.Replicate("test-key", store.Entry{Value: "v"})

		time.Sleep(100 * time.Millisecond)

		// Check all logs, not just the last one
		found := false
		for _, entry := range logger.GetLast(10) {
			if entry.Level == logs.WARN {
				found = true
				assert.Contains(t, entry.Message, "unexpected response")
			}
		}
		assert.True(t, found, "Expected a Warning log for 500 error")
	})

	t.Run("InvalidURL", func(t *testing.T) {
		logger := newTestLogger()
		// Control characters like \x7f make NewRequest fail immediately
		replicator := NewReplicator("node-A", []string{"http://localhost:8080/\x7f"}, logger)
		replicator.Replicate("test-key", store.Entry{Value: "v"})

		time.Sleep(100 * time.Millisecond)
		logs := logger.GetLast(10)
		assert.NotEmpty(t, logs)
		assert.Contains(t, logs[0].Message, "Failed to create replication request")
	})

	t.Run("NetworkCallFailure", func(t *testing.T) {
		logger := newTestLogger()
		// Use a non-existent local port
		replicator := NewReplicator("node-A", []string{"http://localhost:12345"}, logger)

		replicator.Replicate("fail-key", store.Entry{Value: "v"})

		// Use Eventually instead of Sleep to ensure the goroutine finished
		assert.Eventually(t, func() bool {
			entries := logger.GetLast(10)
			for _, e := range entries {
				if e.Level == logs.ERROR && assert.Contains(t, e.Message, "Failed to send") {
					return true
				}
			}
			return false
		}, 1*time.Second, 10*time.Millisecond, "Should have logged a network send error")
	})
}
