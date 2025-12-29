package peers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHeartbeatWorker_RunOnce_Success(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Heartbeat.Interval = 10 * time.Millisecond

	pm := NewPeerManager(cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/heartbeat", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pm.AddPeer(server.URL)

	worker := NewHeartbeatWorker(pm, cfg)
	worker.runOnce(context.Background())

	assert.True(t, pm.IsHealthy(server.URL))
}

func TestHeartbeatWorker_RunOnce_Failure(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	pm := NewPeerManager(cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	pm.AddPeer(server.URL)

	worker := NewHeartbeatWorker(pm, cfg)
	worker.runOnce(context.Background())

	assert.False(t, pm.IsHealthy(server.URL))
}

func TestHeartbeatWorker_RunOnce_NetworkError(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	pm := NewPeerManager(cfg)

	// invalid server URL → forces client.Do error
	pm.AddPeer("http://127.0.0.1:0")

	worker := NewHeartbeatWorker(pm, cfg)
	worker.runOnce(context.Background())

	assert.False(t, pm.IsHealthy("http://127.0.0.1:0"))
}

func TestHeartbeatWorker_ContextCancellation(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Heartbeat.Interval = 10 * time.Millisecond

	pm := NewPeerManager(cfg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pm.AddPeer(server.URL)

	worker := NewHeartbeatWorker(pm, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Start should exit immediately due to context cancellation
	assert.NotPanics(t, func() {
		worker.Start(ctx)
	})
}

func TestHeartbeatWorker_MultiplePeers(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	pm := NewPeerManager(cfg)

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

	worker := NewHeartbeatWorker(pm, cfg)
	worker.runOnce(context.Background())

	assert.True(t, pm.IsHealthy(okServer.URL))
	assert.False(t, pm.IsHealthy(failServer.URL))
}

func TestHeartbeatWorker_RequestCreationError_WithMalformedURL(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1

	pm := NewPeerManager(cfg)

	// Malformed URL triggers http.NewRequestWithContext error
	badPeer := "http://\n"
	pm.AddPeer(badPeer)

	worker := NewHeartbeatWorker(pm, cfg)

	worker.runOnce(context.Background())

	assert.False(
		t,
		pm.IsHealthy(badPeer),
		"peer should be marked unhealthy when request creation fails",
	)
}

func TestHeartbeatWorker_Start_ExecutesRunOnce(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Heartbeat.Interval = 5 * time.Millisecond
	cfg.Health.FailureThreshold = 1

	pm := NewPeerManager(cfg)

	// Heartbeat endpoint that returns failure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	pm.AddPeer(server.URL)

	worker := NewHeartbeatWorker(pm, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start heartbeat worker
	go worker.Start(ctx)

	// Wait until runOnce has definitely executed at least once
	assert.Eventually(t, func() bool {
		return !pm.IsHealthy(server.URL)
	}, 100*time.Millisecond, 5*time.Millisecond)

	cancel()
}
