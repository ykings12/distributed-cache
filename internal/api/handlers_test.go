package api

import (
	"bytes"
	"distributed-cache/internal/store"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setUpTestServer() *httptest.Server {
	st := store.NewStore()
	h := NewHandler(st)
	mux := http.NewServeMux()
	handler := RegisterRoutes(mux, h)
	return httptest.NewServer(handler)
}

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

func TestGetKey(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	// Pre-seed a key for the valid test
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

func TestDeleteKey(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	t.Run("SuccessfulDelete", func(t *testing.T) {
		// Set then delete
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

func TestListKeys(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	t.Run("EmptyStore", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/admin/keys")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var data map[string]string
		json.NewDecoder(resp.Body).Decode(&data)
		assert.Equal(t, 0, len(data))
	})

	t.Run("WithData", func(t *testing.T) {
		// Add data using PUT
		req, _ := http.NewRequest(http.MethodPut, server.URL+"/kv/a", bytes.NewBuffer([]byte(`{"value":"1"}`)))
		http.DefaultClient.Do(req)

		resp, err := http.Get(server.URL + "/admin/keys")
		assert.NoError(t, err)

		var data map[string]string
		json.NewDecoder(resp.Body).Decode(&data)
		assert.True(t, len(data) >= 1)
		assert.Equal(t, "1", data["a"])
		resp.Body.Close()
	})
}

func TestRouteValidation(t *testing.T) {
	server := setUpTestServer()
	defer server.Close()

	t.Run("MethodNotAllowed", func(t *testing.T) {
		// Sending a POST request to /kv/ which only allows PUT, GET, DELETE
		resp, err := http.Post(server.URL+"/kv/key1", "application/json", nil)

		assert.NoError(t, err)
		// This will trigger the 'default' case in your switch statement
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}
