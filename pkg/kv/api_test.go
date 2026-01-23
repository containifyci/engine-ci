package kv

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestGenerateToken(t *testing.T) {
	secret := "test-secret-key"
	token := GenerateToken(secret)

	// Token should have 3 parts separated by dots
	parts := bytes.Split([]byte(token), []byte("."))
	assert.Len(t, parts, 3, "Token should have 3 parts: timestamp.nonce.signature")

	// Token should be valid immediately
	assert.True(t, ValidateToken(token, secret, time.Hour), "Freshly generated token should be valid")
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret-key"

	tests := []struct {
		name   string
		token  string
		secret string
		maxAge time.Duration
		want   bool
	}{
		{
			name:   "valid token",
			token:  GenerateToken(secret),
			secret: secret,
			maxAge: time.Hour,
			want:   true,
		},
		{
			name:   "wrong secret",
			token:  GenerateToken(secret),
			secret: "wrong-secret",
			maxAge: time.Hour,
			want:   false,
		},
		{
			name:   "malformed token - missing parts",
			token:  "only-one-part",
			secret: secret,
			maxAge: time.Hour,
			want:   false,
		},
		{
			name:   "malformed token - invalid timestamp",
			token:  "not-a-number.nonce.signature",
			secret: secret,
			maxAge: time.Hour,
			want:   false,
		},
		{
			name:   "tampered signature",
			token:  fmt.Sprintf("%d.%s.%s", time.Now().Unix(), "nonce", "tampered-signature"),
			secret: secret,
			maxAge: time.Hour,
			want:   false,
		},
		{
			name:   "expired token",
			token:  GenerateToken(secret),
			secret: secret,
			maxAge: 0, // immediate expiration
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateToken(tt.token, tt.secret, tt.maxAge)
			assert.Equal(t, tt.want, got, "ValidateToken() = %v, want %v", got, tt.want)
		})
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	token := GenerateToken(secret)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test with zero maxAge (immediate expiration)
	middleware := authMiddleware(secret, 0, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Expired token should return 401")
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := "test-secret"
	token := GenerateToken(secret)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := authMiddleware(secret, time.Hour, handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Valid token should return 200")
}
