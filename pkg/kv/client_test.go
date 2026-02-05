package kv

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetValue(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		response   string
		wantValue  string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful get",
			key:        "test_key",
			response:   "test_value",
			statusCode: http.StatusOK,
			wantValue:  "test_value",
			wantErr:    false,
		},
		{
			name:       "key not found",
			key:        "missing_key",
			response:   "",
			statusCode: http.StatusNotFound,
			wantValue:  "",
			wantErr:    false,
		},
		{
			name:       "server error",
			key:        "error_key",
			response:   "",
			statusCode: http.StatusInternalServerError,
			wantValue:  "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify auth header
				auth := r.Header.Get("Authorization")
				if auth != "Bearer test_auth" {
					t.Errorf("expected Authorization header 'Bearer test_auth', got '%s'", auth)
				}

				// Verify path
				expectedPath := "/mem/" + tt.key
				if r.URL.Path != expectedPath {
					t.Errorf("expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != "" {
					_, _ = w.Write([]byte(tt.response))
				}
			}))
			defer server.Close()

			// Extract host from server URL (remove http:// prefix)
			host := strings.TrimPrefix(server.URL, "http://")

			got, err := GetValue(host, "test_auth", tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantValue {
				t.Errorf("GetValue() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestSetValue(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful set",
			key:        "test_key",
			value:      "test_value",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "server error",
			key:        "error_key",
			value:      "error_value",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedBody string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != http.MethodPost {
					t.Errorf("expected POST method, got %s", r.Method)
				}

				// Verify auth header
				auth := r.Header.Get("Authorization")
				if auth != "Bearer test_auth" {
					t.Errorf("expected Authorization header 'Bearer test_auth', got '%s'", auth)
				}

				// Verify path
				expectedPath := "/mem/" + tt.key
				if r.URL.Path != expectedPath {
					t.Errorf("expected path '%s', got '%s'", expectedPath, r.URL.Path)
				}

				// Read body
				buf := make([]byte, 1024)
				n, _ := r.Body.Read(buf)
				receivedBody = string(buf[:n])

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			// Extract host from server URL (remove http:// prefix)
			host := strings.TrimPrefix(server.URL, "http://")

			err := SetValue(host, "test_auth", tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && receivedBody != tt.value {
				t.Errorf("SetValue() sent body = %v, want %v", receivedBody, tt.value)
			}
		})
	}
}
