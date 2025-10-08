package kv

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewKeyValueStore(t *testing.T) {
	store1 := NewKeyValueStore()
	store2 := NewKeyValueStore()

	assert.Same(t, store1, store2, "Expected singleton instance, got different instances")
}

func TestKeyValueStore_SetVal(t *testing.T) {
	store := NewKeyValueStore()
	key, value := "foo", "bar"

	store.SetVal(key, value)
	got, ok := store.GetVal(key)
	assert.True(t, ok, "Expected key to exist")
	assert.Equal(t, value, got, "Expected value %s, got %s", value, got)
}

func TestKeyValueStore_Clear(t *testing.T) {
	store := NewKeyValueStore()
	store.SetVal("foo", "bar")

	store.Clear()
	_, ok := store.GetVal("foo")
	assert.False(t, ok, "Expected store to be cleared, but key still exists")
}

func TestKeyValueStore_Get(t *testing.T) {
	store := NewKeyValueStore()
	store.SetVal("foo", "bar")

	req := httptest.NewRequest(http.MethodGet, "/mem/foo", nil)
	req.SetPathValue("key", "foo")
	w := httptest.NewRecorder()

	store.Get(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status %d, got %d", http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	assert.Equal(t, "bar", body, "Expected body %s, got %s", "bar", body)
}

func TestKeyValueStore_Get_NotFound(t *testing.T) {
	store := NewKeyValueStore()
	store.Clear()

	req := httptest.NewRequest(http.MethodGet, "/mem/foo", nil)
	req.SetPathValue("key", "foo")
	w := httptest.NewRecorder()

	store.Get(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
}

func TestKeyValueStore_Set(t *testing.T) {
	store := NewKeyValueStore()
	body := bytes.NewBufferString("bar")
	req := httptest.NewRequest(http.MethodPost, "/mem/foo", body)
	req.SetPathValue("key", "foo")
	w := httptest.NewRecorder()

	store.Set(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status %d, got %d", http.StatusOK, resp.StatusCode)

	value, ok := store.GetVal("foo")
	assert.True(t, ok, "Expected key to exist after set")
	assert.Equal(t, "bar", value, "Expected value %s, got %s", "bar", value)
}

func TestKeyValueStore_Set_InvalidRequest(t *testing.T) {
	store := NewKeyValueStore()
	req := httptest.NewRequest(http.MethodPost, "/mem/foo", nil)
	req.SetPathValue("key", "foo")
	w := httptest.NewRecorder()

	store.Set(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
}

func TestStartHttpServer(t *testing.T) {
	srv, fnc, err := StartHttpServer(NewKeyValueStore())
	assert.NoError(t, err, "Failed to start HTTP server")

	defer func() {
		srv.Listener.Close()
	}()

	go fnc()

	url := fmt.Sprintf("http://localhost:%d/health", srv.Port)

	// Test the /health endpoint
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", srv.Secret))
	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status %d, got %d", http.StatusOK, resp.StatusCode)

	expectedBody := "okay\n"
	body := w.Body.String()
	assert.Equal(t, expectedBody, body, "Expected body %s, got %s", expectedBody, body)

	// Test the /mem/{key} endpoints with the HTTP server
	store := NewKeyValueStore()
	store.SetVal("foo", "bar")

	url = fmt.Sprintf("http://localhost:%d/mem/foo", srv.Port)

	req = httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", srv.Secret))
	req.SetPathValue("key", "foo")
	w = httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, req)

	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status %d, got %d", http.StatusOK, resp.StatusCode)

	body = w.Body.String()
	assert.Equal(t, "bar", body, "Expected body %s, got %s", "bar", body)
}
