package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"distributed-cache/internal/api"
	"distributed-cache/internal/logs"
	"distributed-cache/internal/metrics"
	"distributed-cache/internal/peers"
	"distributed-cache/internal/replication"
	"distributed-cache/internal/store"
	"distributed-cache/internal/ttl"
)

func main() {
	// Root context
	ctx := context.Background()

	// Logger
	logger := logs.NewLogger(1000, logs.DEBUG)

	// Metrics
	metricsRegistry := metrics.NewRegistry()
	metricsRegistry.Inc(metrics.ReplicationRetriesTotal)

	// Store
	cacheStore := store.NewStore(metricsRegistry)
	// logger.Error("panic: simulated failure")

	// Peer management
	peerConfig := peers.DefaultPeerConfig()
	peerManager := peers.NewPeerManager(peerConfig, metricsRegistry)

	peerManager.AddPeer("node-2")
	peerManager.MarkFailure("node-2")
	peerManager.MarkFailure("node-2")
	peerManager.MarkSuccess("node-2")

	// Replication
	replicator := replication.NewReplicator(
		"node-1",
		peerManager,
		peerConfig,
		logger,
		metricsRegistry,
	)
	_ = replicator // (replication will be triggered later via API or hooks)

	// TTL cleaner
	ttlCleaner := ttl.NewCleaner(
		cacheStore,
		5*time.Second,
		logger,
		metricsRegistry,
	)
	go ttlCleaner.Start(ctx)

	// API
	handler := api.NewHandler(
		cacheStore,
		metricsRegistry,
		logger,
		peerManager,
	)
	mux := http.NewServeMux()
	httpHandler := api.RegisterRoutes(mux, handler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: httpHandler,
	}

	logger.Info("server started on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
