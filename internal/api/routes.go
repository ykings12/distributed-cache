package api

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *Handler) http.Handler {
	// KV APIs
	mux.HandleFunc("/kv/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			h.SetKey(w, r)
		case http.MethodGet:
			h.GetKey(w, r)
		case http.MethodDelete:
			h.DeleteKey(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Admin APIs
	mux.HandleFunc("/admin/keys", h.ListKeys)

	// Observability APIs
	mux.HandleFunc("/metrics", h.GetMetrics)
	mux.HandleFunc("/health", h.GetHealth) //
	// Admin APIs
	mux.HandleFunc("/admin/peers", h.GetPeers)

	// Middlewares
	return Chain(
		mux,
		RecoveryMiddleware,
		LoggingMiddleware,
	)
}
