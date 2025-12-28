package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoveryMiddleware(t *testing.T) {
	// 1. Create a handler that intentionally panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom!")
	})

	// 2. Wrap it with the RecoveryMiddleware
	recoveredHandler := RecoveryMiddleware(panicHandler)

	// 3. Send a request to it
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// This should NOT crash the test thanks to your middleware
	recoveredHandler.ServeHTTP(rr, req)

	// 4. Assert that we got a 500 status back instead of a crash
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "internal server error")
}

func TestChain(t *testing.T) {
	// This test ensures your Chain function actually wraps correctly
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "true")
			next.ServeHTTP(w, r)
		})
	}

	chained := Chain(finalHandler, mw)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	chained.ServeHTTP(rr, req)

	assert.Equal(t, "true", rr.Header().Get("X-Test"))
	assert.Equal(t, http.StatusOK, rr.Code)
}
