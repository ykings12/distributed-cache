package replication

import "distributed-cache/internal/store"

//Payload represents the data structure used for replication between nodes

// Each payload contains:
// Key: the cache key being replicated
// Entry: the full value+metadata(timestamp,TTL)
// OriginalNodeID: the ID of the node where the change originated
type Payload struct {
	Key            string      `json:"key"`
	Entry          store.Entry `json:"entry"`
	OriginalNodeID string      `json:"original_node_id"`
}

