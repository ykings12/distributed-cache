package peers

import (
	"context"
	"net/http"
	"time"
)

// HeartbeatWorker periodically checks peer liveness
type HeartbeatWorker struct {
	manager *PeerManager
	client  *http.Client
	config  PeerConfig
}

// NewHeartbeatWorker creates a new heartbeat worker
func NewHeartbeatWorker(
	manager *PeerManager,
	cfg PeerConfig,
) *HeartbeatWorker {
	return &HeartbeatWorker{
		manager: manager,
		client:  &http.Client{Timeout: cfg.Timeout.HeartbeatTimeout},
		config:  cfg,
	}
}

// Start begins the heartbeat loop
// Stops immediately when the ctx is cancelled
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

func (hw *HeartbeatWorker) runOnce(ctx context.Context) {
	for _, peer := range hw.manager.GetPeers() {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			peer+"/internal/heartbeat",
			nil,
		)
		if err != nil {
			hw.manager.MarkFailure(peer)
			continue
		}

		resp, err := hw.client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			hw.manager.MarkFailure(peer)
		} else {
			hw.manager.MarkSuccess(peer)
		}

		if resp != nil {
			defer resp.Body.Close()
		}
	}
}
