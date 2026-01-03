package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"distributed-cache/internal/ai"
	"distributed-cache/internal/logs"
	"distributed-cache/internal/metrics"
	"distributed-cache/internal/peers"
	"distributed-cache/internal/store"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	store    *store.Store
	metrics  *metrics.Registry
	analyzer *ai.HealthAnalyzer
	peers    *peers.PeerManager
}

// NewHandler creates a new API handler.
func NewHandler(
	store *store.Store,
	metrics *metrics.Registry,
	logger *logs.Logger,
	peers *peers.PeerManager,
) *Handler {
	return &Handler{
		store:    store,
		metrics:  metrics,
		analyzer: ai.NewHealthAnalyzer(metrics, logger),
		peers:    peers,
	}
}

/* ---------------- PUT /kv/{key} ---------------- */

type setRequest struct {
	Value string `json:"value"`
	TTLms int64  `json:"ttl_ms,omitempty"`
}

func (h *Handler) SetKey(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")
	if key == "" {
		http.Error(w, "missing key in URL", http.StatusBadRequest)
		return
	}

	var req setRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	entry := store.Entry{
		Value:     req.Value,
		Timestamp: time.Now().UnixNano(),
	}

	if req.TTLms > 0 {
		entry.ExpiresAt = time.Now().Add(time.Duration(req.TTLms) * time.Millisecond)
	}

	h.store.Set(key, entry)
	w.WriteHeader(http.StatusNoContent)
}

/* ---------------- GET /kv/{key} ---------------- */

func (h *Handler) GetKey(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	value, ok := h.store.Get(key)
	if !ok {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"value": value,
	})
}

/* ---------------- DELETE /kv/{key} ---------------- */

func (h *Handler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	h.store.Delete(key)
	w.WriteHeader(http.StatusNoContent)
}

/* ---------------- GET /admin/keys ---------------- */

func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	entries := h.store.List()

	resp := make(map[string]string)
	for k, v := range entries {
		resp[k] = v.Value
	}

	_ = json.NewEncoder(w).Encode(resp)
}

/* ---------------- GET /metrics ---------------- */

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.metrics.Snapshot())
}

/* ---------------- GET /health ---------------- */

func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	report := h.analyzer.Analyze()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}
