package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"distributed-cache/internal/logs"
	"distributed-cache/internal/metrics"
	"distributed-cache/internal/store"

	"github.com/stretchr/testify/assert"
)

func setUpTestServer() *httptest.Server {
	reg := metrics.NewRegistry()
	logger := logs.NewLogger(50, logs.DEBUG)
	st := store.NewStore(reg)

	h := NewHandler(st, reg, logger)

	mux := http.NewServeMux()
	handler := RegisterRoutes(mux, h)

	return httptest.NewServer(handler)
}

/* ---------------- PUT /kv ---------------- */

func TestSetKey(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	client := &http.Client{}

	t.Run("ValidRequest", func(t *testing.T) {
		body := []byte(`{"value":"hello"}`)
		req, _ := http.NewRequest(http.MethodPut, server.URL+"/kv/key1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("WithTTL", func(t *testing.T) {
		body := []byte(`{"value":"expiring", "ttl_ms": 100}`)
		req, _ := http.NewRequest(http.MethodPut, server.URL+"/kv/ttl-key", bytes.NewBuffer(body))

		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("MissingKeyInPath", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, server.URL+"/kv/", nil)
		resp, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, server.URL+"/kv/key1", bytes.NewBuffer([]byte(`{bad-json`)))
		resp, err := client.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

/* ---------------- GET /kv ---------------- */

func TestGetKey(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	body := []byte(`{"value":"found-me"}`)
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/kv/active-key", bytes.NewBuffer(body))
	http.DefaultClient.Do(req)

	t.Run("ValidKey", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/kv/active-key")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var res map[string]string
		json.NewDecoder(resp.Body).Decode(&res)
		assert.Equal(t, "found-me", res["value"])
		resp.Body.Close()
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/kv/missing-key")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("EmptyKeyInPath", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/kv/")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

/* ---------------- DELETE /kv ---------------- */

func TestDeleteKey(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	t.Run("SuccessfulDelete", func(t *testing.T) {
		reqSet, _ := http.NewRequest(http.MethodPut, server.URL+"/kv/to-delete", bytes.NewBuffer([]byte(`{"value":"x"}`)))
		http.DefaultClient.Do(reqSet)

		reqDel, _ := http.NewRequest(http.MethodDelete, server.URL+"/kv/to-delete", nil)
		resp, err := http.DefaultClient.Do(reqDel)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("EmptyKeyInPath", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, server.URL+"/kv/", nil)
		resp, err := http.DefaultClient.Do(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

/* ---------------- GET /admin/keys ---------------- */

func TestListKeys(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	t.Run("EmptyStore", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/admin/keys")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var data map[string]string
		json.NewDecoder(resp.Body).Decode(&data)
		assert.Len(t, data, 0)
	})

	t.Run("WithData", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, server.URL+"/kv/a", bytes.NewBuffer([]byte(`{"value":"1"}`)))
		http.DefaultClient.Do(req)

		resp, err := http.Get(server.URL + "/admin/keys")
		assert.NoError(t, err)

		var data map[string]string
		json.NewDecoder(resp.Body).Decode(&data)
		assert.Equal(t, "1", data["a"])
		resp.Body.Close()
	})
}

/* ---------------- GET /metrics ---------------- */

func TestGetMetrics(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/metrics")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var data map[string]int64
	err = json.NewDecoder(resp.Body).Decode(&data)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	resp.Body.Close()
}

/* ---------------- GET /health ---------------- */

func TestGetHealth(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var report map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&report)
	assert.NoError(t, err)

	assert.Contains(t, report, "overall_status")
	assert.Contains(t, report, "summary")
	assert.Contains(t, report, "signals")
	assert.Contains(t, report, "recommendations")

	resp.Body.Close()
}

/* ---------------- Route validation ---------------- */

func TestRouteValidation(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	t.Run("MethodNotAllowed", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/kv/key1", "application/json", nil)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}
