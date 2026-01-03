package peers

import (
	"testing"

	"distributed-cache/internal/metrics"

	"github.com/stretchr/testify/assert"
)

func TestPeerManagerAddAndIsHealthy(t *testing.T) {
	cfg := DefaultPeerConfig()
	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	pm.AddPeer("node-1")
	assert.True(t, pm.IsHealthy("node-1"))
	assert.False(t, pm.IsHealthy("node-2"))
}

func TestPeerManagerMarkFailureTransitionsToUnhealthy(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 2

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	pm.AddPeer("node-1")

	pm.MarkFailure("node-1")
	assert.True(t, pm.IsHealthy("node-1"))

	pm.MarkFailure("node-1")
	assert.False(t, pm.IsHealthy("node-1"))

	snap := reg.Snapshot()
	assert.Equal(t, int64(2), snap[string(metrics.PeerFailuresTotal)])
	assert.Equal(t, int64(1), snap[string(metrics.PeersUnhealthy)])
}

func TestPeerManagerMarkSuccessRecoversPeer(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1
	cfg.Health.SuccessThreshold = 2

	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	pm.AddPeer("node-1")
	pm.MarkFailure("node-1")
	assert.False(t, pm.IsHealthy("node-1"))

	pm.MarkSuccess("node-1")
	assert.False(t, pm.IsHealthy("node-1"))

	pm.MarkSuccess("node-1")
	assert.True(t, pm.IsHealthy("node-1"))

	snap := reg.Snapshot()
	assert.Equal(t, int64(1), snap[string(metrics.PeersHealthy)])
}

func TestPeerManagerCountersResetCorrectly(t *testing.T) {
	cfg := DefaultPeerConfig()
	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	pm.AddPeer("node-1")

	pm.MarkSuccess("node-1")
	pm.MarkFailure("node-1")

	peer := pm.peers["node-1"]
	assert.Equal(t, 0, peer.SuccessCount)
	assert.Equal(t, 1, peer.FailureCount)
}

func TestPeerManagerUnknownPeerNoPanic(t *testing.T) {
	cfg := DefaultPeerConfig()
	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	assert.NotPanics(t, func() {
		pm.MarkFailure("unknown-peer")
		pm.MarkSuccess("unknown-peer")
	})
}

func TestPeerManagerGetPeersSnapshot(t *testing.T) {
	cfg := DefaultPeerConfig()
	reg := metrics.NewRegistry()
	pm := NewPeerManager(cfg, reg)

	pm.AddPeer("node-1")
	pm.AddPeer("node-2")

	peers := pm.GetPeers()
	assert.Len(t, peers, 2)
	assert.Contains(t, peers, "node-1")
	assert.Contains(t, peers, "node-2")
}
