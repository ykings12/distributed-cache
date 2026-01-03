package peers

import (
	"context"
	"net/http"
	"time"

	"distributed-cache/internal/metrics"
)

// HeartbeatWorker periodically checks peer liveness.
type HeartbeatWorker struct {
	manager *PeerManager
	client  *http.Client
	config  PeerConfig
	metrics *metrics.Registry
}

// NewHeartbeatWorker creates a new heartbeat worker.
func NewHeartbeatWorker(
	manager *PeerManager,
	cfg PeerConfig,
	metricsRegistry *metrics.Registry,
) *HeartbeatWorker {
	return &HeartbeatWorker{
		manager: manager,
		client:  &http.Client{Timeout: cfg.Timeout.HeartbeatTimeout},
		config:  cfg,
		metrics: metricsRegistry,
	}
}

// Start begins the heartbeat loop.
// Stops immediately when the ctx is cancelled.
func (hw *HeartbeatWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(hw.config.Heartbeat.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hw.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// runOnce performs a single heartbeat check for all peers.
func (hw *HeartbeatWorker) runOnce(ctx context.Context) {
	hw.metrics.Inc(metrics.HeartbeatRunsTotal)

	for _, peer := range hw.manager.GetPeers() {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			peer+"/internal/heartbeat",
			nil,
		)
		if err != nil {
			hw.metrics.Inc(metrics.HeartbeatFailuresTotal)
			hw.manager.MarkFailure(peer)
			continue
		}

		resp, err := hw.client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			hw.metrics.Inc(metrics.HeartbeatFailuresTotal)
			hw.manager.MarkFailure(peer)
		} else {
			hw.metrics.Inc(metrics.HeartbeatSuccessTotal)
			hw.manager.MarkSuccess(peer)
		}

		if resp != nil {
			resp.Body.Close()
		}
	}
}
