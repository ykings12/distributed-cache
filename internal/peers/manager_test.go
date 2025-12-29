package peers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeerManagerAndPeerAndIsHealthy(t *testing.T) {
	cfg := DefaultPeerConfig()
	pm := NewPeerManager(cfg)

	pm.AddPeer("node-1")
	assert.True(t, pm.IsHealthy("node-1"), "newly added peer should be healthy")
	assert.False(t, pm.IsHealthy("node-2"), "non-existent peer should not be healthy")
}

func TestPeerManagerMarkFailureMarkUnhealthyAfterThreshold(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 2

	pm := NewPeerManager(cfg)
	pm.AddPeer("node-1")

	pm.MarkFailure("node-1")

	assert.True(t, pm.IsHealthy("node-1"), "peer should still be healthy after 1 failure")

	pm.MarkFailure("node-1")
	assert.False(t, pm.IsHealthy("node-1"), "peer should be unhealthy after 2 failures")
}

func TestPeerManagerMarkSuccessRecoverPeer(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 1
	cfg.Health.SuccessThreshold = 2

	pm := NewPeerManager(cfg)
	pm.AddPeer("node-1")

	pm.MarkFailure("node-1")
	assert.False(t, pm.IsHealthy("node-1"), "peer should be unhealthy after failure")

	pm.MarkSuccess("node-1")
	assert.False(t, pm.IsHealthy("node-1"), "peer should not recover after 1 success")

	pm.MarkSuccess("node-1")
	assert.True(t, pm.IsHealthy("node-1"), "peer should recover after success threshold is met")
}

func TestPeerManagerMarkFailureResetsSuccessCount(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.SuccessThreshold = 2

	pm := NewPeerManager(cfg)
	pm.AddPeer("node-1")

	pm.MarkSuccess("node-1")
	pm.MarkFailure("node-1")

	peer := pm.peers["node-1"]
	assert.Equal(t, 0, peer.SuccessCount, "success count should be reset after failure")
}

func TestPeerManagerMarkSuccessResetsFailureCount(t *testing.T) {
	cfg := DefaultPeerConfig()
	cfg.Health.FailureThreshold = 2

	pm := NewPeerManager(cfg)
	pm.AddPeer("node-1")

	pm.MarkFailure("node-1")
	pm.MarkSuccess("node-1")

	peer := pm.peers["node-1"]
	assert.Equal(t, 0, peer.FailureCount, "failure count should be reset after success")
}

func TestPeerManagerUnknownPeerNoPanic(t *testing.T) {
	cfg := DefaultPeerConfig()
	pm := NewPeerManager(cfg)

	assert.NotPanics(t, func() {
		pm.MarkFailure("unknown-peer")
		pm.MarkSuccess("unknown-peer")
	}, "marking unknown peer should not panic")
}

func TestPeerManagerGetPeersReturnsSnapshot(t *testing.T) {
	cfg := DefaultPeerConfig()
	pm := NewPeerManager(cfg)

	pm.AddPeer("node-1")
	pm.AddPeer("node-2")

	peers := pm.GetPeers()
	assert.Len(t, peers, 2, "should return a snapshot of all peers")
	assert.Contains(t, peers, "node-1", "snapshot should contain node-1")
	assert.Contains(t, peers, "node-2", "snapshot should contain node-2")
}
