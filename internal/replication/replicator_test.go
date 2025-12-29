package replication

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"distributed-cache/internal/logs"
	"distributed-cache/internal/peers"
	"distributed-cache/internal/store"

	"github.com/stretchr/testify/assert"
)

func TestReplicator_HealthyPeer_ReplicationSucceeds(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := peers.DefaultPeerConfig()
	pm := peers.NewPeerManager(cfg)
	pm.AddPeer(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger)

	replicator.Replicate(context.Background(), "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&calls) == 1
	}, time.Second, 10*time.Millisecond)

	assert.True(t, pm.IsHealthy(server.URL))
}

func TestReplicator_UnhealthyPeer_IsSkipped(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := peers.DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	pm := peers.NewPeerManager(cfg)
	pm.AddPeer(server.URL)

	// Make peer unhealthy
	pm.MarkFailure(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger)

	replicator.Replicate(context.Background(), "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(0), atomic.LoadInt32(&calls))
}

func TestReplicator_RetryThenSuccess(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := peers.DefaultPeerConfig()
	cfg.Retry.MaxRetries = 2
	cfg.Retry.BaseBackoff = 1 * time.Millisecond
	cfg.Retry.JitterFn = func(d time.Duration) time.Duration { return 0 }

	pm := peers.NewPeerManager(cfg)
	pm.AddPeer(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger)

	replicator.Replicate(context.Background(), "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&calls) >= 2
	}, time.Second, 10*time.Millisecond)

	assert.True(t, pm.IsHealthy(server.URL))
}

func TestReplicator_RetryExhaustion_MarksUnhealthy(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := peers.DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1
	cfg.Retry.MaxRetries = 1
	cfg.Retry.BaseBackoff = 1 * time.Millisecond
	cfg.Retry.JitterFn = func(d time.Duration) time.Duration { return 0 }

	pm := peers.NewPeerManager(cfg)
	pm.AddPeer(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger)

	replicator.Replicate(context.Background(), "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	assert.Eventually(t, func() bool {
		return !pm.IsHealthy(server.URL)
	}, time.Second, 10*time.Millisecond)
}

func TestReplicator_ContextCancelled_NoRetry(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := peers.DefaultPeerConfig()
	cfg.Retry.MaxRetries = 5
	cfg.Retry.BaseBackoff = 10 * time.Millisecond

	pm := peers.NewPeerManager(cfg)
	pm.AddPeer(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	replicator.Replicate(ctx, "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	time.Sleep(50 * time.Millisecond)
	assert.LessOrEqual(t, atomic.LoadInt32(&calls), int32(1))
}

func TestSendOnce_RequestCreationError(t *testing.T) {
	cfg := peers.DefaultPeerConfig()
	pm := peers.NewPeerManager(cfg)

	logger := logs.NewLogger(10, logs.DEBUG)
	r := NewReplicator("node-A", pm, cfg, logger)

	payload := Payload{
		Key: "key",
		Entry: store.Entry{
			Value: "value",
		},
		OriginalNodeID: "node-A",
	}

	err := r.sendOnce(context.Background(), "http://\n", payload)

	assert.Error(t, err)
}
