package replication

import (
	"bytes"
	"distributed-cache/internal/logs"
	"distributed-cache/internal/store"
	"encoding/json"
	"net/http"
	"time"
)

// Replicator handles replication of cache entries across distributed nodes
type Replicator struct {
	nodeID string       // ID of the current node
	peers  []string     // IDs of peer nodes
	logger *logs.Logger // shared Logger instance for replication events
	client *http.Client // HTTP client for replication requests
}

// NewReplicator creates a new Replicator instance
func NewReplicator(
	nodeID string,
	peers []string,
	logger *logs.Logger,
) *Replicator {
	return &Replicator{
		nodeID: nodeID,
		peers:  peers,
		logger: logger,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

//Replicate sends the given key and entry to all peer nodes asynchronously
//Behavior:
//1. Does not block the caller
//2. Spawns a goroutine for each peer to handle replication

// this method is called after the local write is successful
func (r *Replicator) Replicate(key string, entry store.Entry) {
	payload := Payload{
		Key:            key,
		Entry:          entry,
		OriginalNodeID: r.nodeID,
	}

	for _, peer := range r.peers {
		peer := peer //capture range variable
		go r.sendToPeer(peer, payload)
	}
}

func (r *Replicator) sendToPeer(peer string, payload Payload) {
	body, err := json.Marshal(payload)
	if err != nil {
		r.logger.Error("Failed to marshal payload for replication to peer")
		return
	}

	req, err := http.NewRequest(
		http.MethodPost,
		peer+"/internal/replicate",
		bytes.NewBuffer(body),
	)
	if err != nil {
		r.logger.Error("Failed to create replication request to peer: " + peer)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Error("Failed to send replication request to peer: " + peer)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		r.logger.Warn("unexpected response from peer " + peer + " during replication: " + resp.Status)
	}
	r.logger.Debug("replicated key " + payload.Key + " to peer " + peer)
}
