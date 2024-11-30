package kv

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
)

type Server struct {
	Listener net.Listener
	Port     int
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

// Get value by key
func (kv *KeyValueStore) Get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	value, ok := kv.GetVal(key)
	if !ok {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	_, err := w.Write([]byte(value))
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// Set value for a key
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
	//TODO define maximal retries
	for {
		port := rand.Intn(65535-1024) + 1024 // Random port between 1024 and 65535
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return &Server{
				Listener: l,
				Port:     port,
			}, nil
		}
	}
}

func StartHttpServer(kvStore *KeyValueStore) (error, *Server, func()) {
	srv, err := getRandomPort()
	if err != nil {
		slog.Error("Failed to find available port", "error", err)
		return err, nil, nil
	}

	handler := http.NewServeMux()
	http.DefaultServeMux = handler

	handler.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "okay")
	})

	handler.Handle("GET /mem/{key}", http.HandlerFunc(kvStore.Get))
	handler.Handle("POST /mem/{key}", http.HandlerFunc(kvStore.Set))

	return nil, srv, func() {
		if err := http.Serve(srv.Listener, handler); err != nil &&
			!strings.HasSuffix(err.Error(), "use of closed network connection") {
			slog.Error("Failed to start http server", "error", err)
		}
	}

}
