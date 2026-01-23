package kv

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Listener   net.Listener
	Secret     string
	signingKey string
	Port       int
}

// In-memory key-value store
type KeyValueStore struct {
	store map[string]string
	mu    sync.RWMutex
}

var kvStore *KeyValueStore

// Define a sync.Once variable to ensure the singleton is only created once
var once sync.Once

func NewKeyValueStore() *KeyValueStore {
	// Ensure that the singleton is initialized only once
	once.Do(func() {
		kvStore = &KeyValueStore{
			store: make(map[string]string),
		}
		fmt.Println("Singleton instance created")
	})
	return kvStore
}

func (kv *KeyValueStore) Clear() {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.store = make(map[string]string)
}

func (kv *KeyValueStore) GetVal(key string) (val string, ok bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	val, ok = kv.store[key]
	return
}

func (kv *KeyValueStore) SetVal(key, val string) {
	kv.mu.Lock()
	kv.store[key] = val
	kv.mu.Unlock()
}

// Get retrieves a value by key from the key-value store.
func (kv *KeyValueStore) Get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value, ok := kv.GetVal(key)
	if !ok {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	if _, err := w.Write([]byte(value)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// Set stores a value for a key in the key-value store.
func (kv *KeyValueStore) Set(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value, err := io.ReadAll(r.Body)
	if err != nil || len(value) == 0 {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	kv.SetVal(key, string(value))
	w.WriteHeader(http.StatusOK)
}

func getRandomPort() (*Server, error) {
	const (
		minPort   = 1024
		portRange = 65535 - minPort
	)

	for {
		num, err := rand.Int(rand.Reader, big.NewInt(portRange))
		if err != nil {
			panic(fmt.Sprintf("crypto/rand failed: %v", err))
		}
		port := int(num.Int64()) + minPort
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return &Server{
				Listener: l,
				Port:     port,
			}, nil
		}
	}
}

func authMiddleware(signingKey string, maxAge time.Duration, next http.Handler) http.Handler {
	const bearerPrefix = "Bearer "
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		token := authHeader[len(bearerPrefix):]
		if !ValidateToken(token, signingKey, maxAge) {
			http.Error(w, "invalid auth", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func StartHttpServer(kvStore *KeyValueStore) (*Server, func(), error) {
	srv, err := getRandomPort()
	if err != nil {
		slog.Error("Failed to find available port", "error", err)
		return nil, nil, err
	}

	handler := http.NewServeMux()
	handler.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "okay")
	})
	handler.Handle("GET /mem/{key}", http.HandlerFunc(kvStore.Get))
	handler.Handle("POST /mem/{key}", http.HandlerFunc(kvStore.Set))

	srv.signingKey = randomString(32)
	srv.Secret = GenerateToken(srv.signingKey)

	authenticatedHandler := http.NewServeMux()
	authenticatedHandler.Handle("/", authMiddleware(srv.signingKey, TokenMaxAge, handler))
	http.DefaultServeMux = authenticatedHandler

	return srv, func() {
		if err := http.Serve(srv.Listener, nil); err != nil &&
			!strings.HasSuffix(err.Error(), "use of closed network connection") {
			slog.Error("Failed to start http server", "error", err)
		}
	}, nil
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(fmt.Sprintf("crypto/rand failed: %v", err))
		}
		b[i] = letters[num.Int64()]
	}
	return string(b)
}

// TokenMaxAge is the default expiration window for auth tokens.
const TokenMaxAge = time.Hour

// GenerateToken creates a signed token with timestamp and nonce.
// Format: {timestamp}.{nonce}.{signature}
func GenerateToken(secret string) string {
	timestamp := time.Now().Unix()
	nonce := randomString(16)
	payload := fmt.Sprintf("%d.%s", timestamp, nonce)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", payload, signature)
}

// ValidateToken checks if the token is valid and not expired.
func ValidateToken(token, secret string, maxAge time.Duration) bool {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return false
	}

	timestampStr, nonce, providedSig := parts[0], parts[1], parts[2]

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return false
	}

	// Check expiration
	tokenTime := time.Unix(timestamp, 0)
	if time.Since(tokenTime) > maxAge {
		return false
	}

	// Recompute signature
	payload := fmt.Sprintf("%d.%s", timestamp, nonce)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	// Constant-time comparison
	return subtle.ConstantTimeCompare([]byte(providedSig), []byte(expectedSig)) == 1
}
