package kv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// GetValue retrieves a value from the KV store via HTTP
func GetValue(host, auth, key string) (string, error) {
	url := fmt.Sprintf("http://%s/mem/%s", host, key)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // Key not found, return empty string
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get key-value pair: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// SetValue stores a value in the KV store via HTTP
func SetValue(host, auth, key, value string) error {
	url := fmt.Sprintf("http://%s/mem/%s", host, key)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer([]byte(value)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set key-value pair: %s", resp.Status)
	}

	return nil
}
