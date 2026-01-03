package replication

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"distributed-cache/internal/logs"
	"distributed-cache/internal/metrics"
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

	reg := metrics.NewRegistry()
	pm := peers.NewPeerManager(cfg, reg)
	pm.AddPeer(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger, reg)

	replicator.Replicate(context.Background(), "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&calls) == 1
	}, time.Second, 10*time.Millisecond)

	assert.True(t, pm.IsHealthy(server.URL))

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.ReplicationAttemptsTotal)])
	assert.Equal(t, int64(1), snap[string(metrics.ReplicationSuccessTotal)])
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

	reg := metrics.NewRegistry()
	pm := peers.NewPeerManager(cfg, reg)
	pm.AddPeer(server.URL)
	pm.MarkFailure(server.URL) // force unhealthy

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger, reg)

	replicator.Replicate(context.Background(), "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(0), atomic.LoadInt32(&calls))

	snap := reg.Snapshot()
	assert.Equal(t, int64(0), snap[string(metrics.ReplicationAttemptsTotal)])
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

	reg := metrics.NewRegistry()
	pm := peers.NewPeerManager(cfg, reg)
	pm.AddPeer(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger, reg)

	replicator.Replicate(context.Background(), "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&calls) >= 2
	}, time.Second, 10*time.Millisecond)

	assert.True(t, pm.IsHealthy(server.URL))

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.ReplicationSuccessTotal)])
	assert.GreaterOrEqual(t, snap[string(metrics.ReplicationRetriesTotal)], int64(1))
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

	reg := metrics.NewRegistry()
	pm := peers.NewPeerManager(cfg, reg)
	pm.AddPeer(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger, reg)

	replicator.Replicate(context.Background(), "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	assert.Eventually(t, func() bool {
		return !pm.IsHealthy(server.URL)
	}, time.Second, 10*time.Millisecond)

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.ReplicationFailureTotal)])
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

	reg := metrics.NewRegistry()
	pm := peers.NewPeerManager(cfg, reg)
	pm.AddPeer(server.URL)

	logger := logs.NewLogger(10, logs.DEBUG)
	replicator := NewReplicator("node-A", pm, cfg, logger, reg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	replicator.Replicate(ctx, "key", store.Entry{
		Value:     "val",
		Timestamp: 1,
	})

	time.Sleep(50 * time.Millisecond)
	assert.LessOrEqual(t, atomic.LoadInt32(&calls), int32(1))
}

func TestSendOnce_RequestCreationError(t *testing.T) {
	cfg := peers.DefaultPeerConfig()

	reg := metrics.NewRegistry()
	pm := peers.NewPeerManager(cfg, reg)

	logger := logs.NewLogger(10, logs.DEBUG)
	r := NewReplicator("node-A", pm, cfg, logger, reg)

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
