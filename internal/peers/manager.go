package peers

import "sync"

// PeerState represents the health state of a peer.
type PeerState int

const (
	Healthy PeerState = iota
	Unhealthy
)

// Peer tracks the health-related state for a single peer
type Peer struct {
	Address      string
	State        PeerState
	FailureCount int
	SuccessCount int
}

// PeerManager manages the health state of multiple peers
type PeerManager struct {
	mu     sync.RWMutex
	peers  map[string]*Peer
	config PeerConfig
}

// NewPeerManager creates a new PeerManager
func NewPeerManager(cfg PeerConfig) *PeerManager {
	return &PeerManager{
		peers:  make(map[string]*Peer),
		config: cfg,
	}
}

// AddPeer registers a new Peer
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

// MarkFailure marks a peer as failed
func (pm *PeerManager) MarkFailure(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	peer, ok := pm.peers[addr]
	if !ok {
		return
	}
	peer.FailureCount++
	peer.SuccessCount = 0
	if peer.FailureCount >= pm.config.Health.FailureThreshold {
		peer.State = Unhealthy
	}
}

// MarkSuccess marks a peer as successful
func (pm *PeerManager) MarkSuccess(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	peer, ok := pm.peers[addr]
	if !ok {
		return
	}
	peer.SuccessCount++
	peer.FailureCount = 0
	if peer.SuccessCount >= pm.config.Health.SuccessThreshold {
		peer.State = Healthy
	}
}

func (pm *PeerManager) IsHealthy(addr string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peer, ok := pm.peers[addr]
	return ok && peer.State == Healthy
}

func (pm *PeerManager) GetPeers() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	out := make([]string, 0, len(pm.peers))
	for addr := range pm.peers {
		out = append(out, addr)
	}
	return out
}
