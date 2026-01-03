package peers

import (
	"sync"

	"distributed-cache/internal/metrics"
)

// PeerState represents the health state of a peer.
type PeerState int

const (
	Healthy PeerState = iota
	Unhealthy
)

// Peer tracks the health-related state for a single peer.
type Peer struct {
	Address      string
	State        PeerState
	FailureCount int
	SuccessCount int
}

// PeerManager manages the health state of multiple peers.
type PeerManager struct {
	mu      sync.RWMutex
	peers   map[string]*Peer
	config  PeerConfig
	metrics *metrics.Registry
}

// NewPeerManager creates a new PeerManager.
func NewPeerManager(cfg PeerConfig, metricsRegistry *metrics.Registry) *PeerManager {
	return &PeerManager{
		peers:   make(map[string]*Peer),
		config:  cfg,
		metrics: metricsRegistry,
	}
}

// AddPeer registers a new peer.
func (pm *PeerManager) AddPeer(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.peers[addr]; !exists {
		pm.peers[addr] = &Peer{
			Address: addr,
			State:   Healthy,
		}
	}
}

// MarkFailure records a failure and may mark the peer unhealthy.
func (pm *PeerManager) MarkFailure(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	peer, ok := pm.peers[addr]
	if !ok {
		return
	}

	peer.FailureCount++
	peer.SuccessCount = 0

	pm.metrics.Inc(metrics.PeerFailuresTotal)

	if peer.State == Healthy &&
		peer.FailureCount >= pm.config.Health.FailureThreshold {

		peer.State = Unhealthy
		pm.metrics.Inc(metrics.PeersUnhealthy)
	}
}

// MarkSuccess records a success and may recover the peer.
func (pm *PeerManager) MarkSuccess(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	peer, ok := pm.peers[addr]
	if !ok {
		return
	}

	peer.SuccessCount++
	peer.FailureCount = 0

	if peer.State == Unhealthy &&
		peer.SuccessCount >= pm.config.Health.SuccessThreshold {

		peer.State = Healthy
		pm.metrics.Inc(metrics.PeersHealthy)
	}
}

// IsHealthy returns whether a peer is healthy.
func (pm *PeerManager) IsHealthy(addr string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peer, ok := pm.peers[addr]
	return ok && peer.State == Healthy
}

// GetPeers returns a snapshot of peer addresses.
func (pm *PeerManager) GetPeers() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	out := make([]string, 0, len(pm.peers))
	for addr := range pm.peers {
		out = append(out, addr)
	}
	return out
}

// PeerSnapshot represents a safe, read-only view of a peer
type PeerSnapshot struct {
	Address      string `json:"address"`
	State        string `json:"state"`
	FailureCount int    `json:"failure_count"`
	SuccessCount int    `json:"success_count"`
}

// Snapshot returns a copy of all peer states
func (pm *PeerManager) Snapshot() []PeerSnapshot {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	out := make([]PeerSnapshot, 0, len(pm.peers))
	for _, p := range pm.peers {
		state := "healthy"
		if p.State == Unhealthy {
			state = "unhealthy"
		}

		out = append(out, PeerSnapshot{
			Address:      p.Address,
			State:        state,
			FailureCount: p.FailureCount,
			SuccessCount: p.SuccessCount,
		})
	}
	return out
}
