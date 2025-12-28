package api

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *Handler) http.Handler {
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

	mux.HandleFunc("/admin/keys", h.ListKeys)

	// Now this return statement is valid
	return Chain(
		mux,
		RecoveryMiddleware,
		LoggingMiddleware,
	)
}
