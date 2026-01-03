package peers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"distributed-cache/internal/metrics"

	"github.com/stretchr/testify/assert"
)

func TestHeartbeatWorker_RunOnce_Success(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Heartbeat.Interval = 10 * time.Millisecond

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/heartbeat", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pm.AddPeer(server.URL)

	worker := NewHeartbeatWorker(pm, cfg, reg)
	worker.runOnce(context.Background())

	assert.True(t, pm.IsHealthy(server.URL))

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.HeartbeatRunsTotal)])
	assert.Equal(t, int64(1), snap[string(metrics.HeartbeatSuccessTotal)])
}

func TestHeartbeatWorker_RunOnce_Failure(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	pm.AddPeer(server.URL)

	worker := NewHeartbeatWorker(pm, cfg, reg)
	worker.runOnce(context.Background())

	assert.False(t, pm.IsHealthy(server.URL))

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.HeartbeatFailuresTotal)])
}

func TestHeartbeatWorker_RunOnce_NetworkError(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	badPeer := "http://127.0.0.1:0"
	pm.AddPeer(badPeer)

	worker := NewHeartbeatWorker(pm, cfg, reg)
	worker.runOnce(context.Background())

	assert.False(t, pm.IsHealthy(badPeer))

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.HeartbeatFailuresTotal)])
}

func TestHeartbeatWorker_ContextCancellation(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Heartbeat.Interval = 10 * time.Millisecond

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	worker := NewHeartbeatWorker(pm, cfg, reg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.NotPanics(t, func() {
		worker.Start(ctx)
	})
}

func TestHeartbeatWorker_MultiplePeers(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	pm.AddPeer(okServer.URL)
	pm.AddPeer(failServer.URL)

	worker := NewHeartbeatWorker(pm, cfg, reg)
	worker.runOnce(context.Background())

	assert.True(t, pm.IsHealthy(okServer.URL))
	assert.False(t, pm.IsHealthy(failServer.URL))

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.HeartbeatSuccessTotal)])
	assert.Equal(t, int64(1), snap[string(metrics.HeartbeatFailuresTotal)])
}

func TestHeartbeatWorker_Start_ExecutesRunOnce(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Heartbeat.Interval = 5 * time.Millisecond
	cfg.Health.FailureThreshold = 1

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	pm.AddPeer(server.URL)

	worker := NewHeartbeatWorker(pm, cfg, reg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go worker.Start(ctx)

	assert.Eventually(t, func() bool {
		return !pm.IsHealthy(server.URL)
	}, 100*time.Millisecond, 5*time.Millisecond)
}

func TestHeartbeatWorker_RequestCreationError_IncrementsMetrics(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	// Malformed URL → forces http.NewRequestWithContext error
	badPeer := "http://\n"
	pm.AddPeer(badPeer)

	worker := NewHeartbeatWorker(pm, cfg, reg)

	worker.runOnce(context.Background())

	// Peer should be marked unhealthy
	assert.False(t, pm.IsHealthy(badPeer))

	// Metrics must be incremented
	snap := reg.Snapshot()
	assert.Equal(
		t,
		int64(1),
		snap[string(metrics.HeartbeatFailuresTotal)],
	)
}
