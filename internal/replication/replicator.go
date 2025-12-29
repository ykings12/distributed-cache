package replication

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"distributed-cache/internal/logs"
	"distributed-cache/internal/peers"
	"distributed-cache/internal/store"
)

// Replicator handles reliable, health-aware replication of writes.
type Replicator struct {
	nodeID string

	peers  *peers.PeerManager
	config peers.PeerConfig

	logger *logs.Logger
	client *http.Client
}

// NewReplicator creates a replication engine integrated with
// peer health tracking and retry policies.
func NewReplicator(
	nodeID string,
	peerManager *peers.PeerManager,
	cfg peers.PeerConfig,
	logger *logs.Logger,
) *Replicator {
	return &Replicator{
		nodeID: nodeID,
		peers:  peerManager,
		config: cfg,
		logger: logger,
		client: &http.Client{
			Timeout: cfg.Timeout.ReplicationTimeout,
		},
	}
}

// Replicate sends a cache write to all healthy peers asynchronously.
// Replication is retry-aware, cancellable, and updates peer health.
func (r *Replicator) Replicate(
	ctx context.Context,
	key string,
	entry store.Entry,
) {
	payload := Payload{
		Key:            key,
		Entry:          entry,
		OriginalNodeID: r.nodeID,
	}

	for _, peer := range r.peers.GetPeers() {

		// Skip unhealthy peers
		if !r.peers.IsHealthy(peer) {
			r.logger.Debug("skipping unhealthy peer " + peer)
			continue
		}

		peer := peer // capture loop variable
		go r.sendWithRetry(ctx, peer, payload)
	}
}

// sendWithRetry performs replication using the Retry engine
// and updates peer health based on the final outcome.
func (r *Replicator) sendWithRetry(
	ctx context.Context,
	peer string,
	payload Payload,
) {
	err := peers.Retry(ctx, r.config.Retry, func() error {
		return r.sendOnce(ctx, peer, payload)
	})

	if err != nil {
		r.peers.MarkFailure(peer)
		r.logger.Warn("replication failed to peer " + peer)
		return
	}

	r.peers.MarkSuccess(peer)
	r.logger.Debug("replication succeeded to peer " + peer)
}

// sendOnce performs a single HTTP replication attempt.
// It returns an error so Retry() can decide what to do.
func (r *Replicator) sendOnce(
	ctx context.Context,
	peer string,
	payload Payload,
) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		peer+"/internal/replicate",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return http.ErrHandlerTimeout // treated as retryable
	}

	return nil
}
