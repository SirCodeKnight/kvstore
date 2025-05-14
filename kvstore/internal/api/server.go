package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/SirCodeKnight/kvstore/internal/metrics"
	"github.com/SirCodeKnight/kvstore/internal/raft"
	"github.com/SirCodeKnight/kvstore/internal/storage"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server represents the REST API server
type Server struct {
	node      *raft.Node
	logger    *zap.Logger
	metrics   *metrics.Metrics
	router    *mux.Router
	address   string
}

// NewServer creates a new API server
func NewServer(node *raft.Node, addr string, metrics *metrics.Metrics, logger *zap.Logger) *Server {
	s := &Server{
		node:    node,
		logger:  logger,
		metrics: metrics,
		address: addr,
	}
	
	// Create router
	router := mux.NewRouter()
	
	// Key-value endpoints
	router.HandleFunc("/v1/kv/{key}", s.handleGet).Methods("GET")
	router.HandleFunc("/v1/kv/{key}", s.handleSet).Methods("PUT", "POST")
	router.HandleFunc("/v1/kv/{key}", s.handleDelete).Methods("DELETE")
	router.HandleFunc("/v1/kv", s.handleGetAll).Methods("GET")
	
	// Raft endpoints
	router.HandleFunc("/v1/raft/status", s.handleRaftStatus).Methods("GET")
	router.HandleFunc("/v1/raft/join", s.handleRaftJoin).Methods("POST")
	
	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler())
	
	// Health check
	router.HandleFunc("/health", s.handleHealth).Methods("GET")
	
	s.router = router
	return s
}

// Run starts the server
func (s *Server) Run() error {
	s.logger.Info("starting API server", zap.String("address", s.address))
	return http.ListenAndServe(s.address, s.router)
}

// handleGet handles GET requests for a key
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	
	start := time.Now()
	value, err := s.node.Get(key)
	duration := time.Since(start)
	
	s.metrics.ObserveGetLatency(duration.Seconds())
	
	if err != nil {
		if err == storage.ErrKeyNotFound || err == storage.ErrKeyExpired {
			s.metrics.IncGetMiss()
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		
		s.logger.Error("failed to get key", zap.String("key", key), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	s.metrics.IncGetHit()
	
	// Set content type based on data
	w.Header().Set("Content-Type", "application/octet-stream")
	
	// Write the value
	w.Write(value.Data)
}

// handleSet handles PUT/POST requests to set a key
func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	
	// Read the value from the request body
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Parse TTL from query string
	var expiration int64 = 0
	if ttlStr := r.URL.Query().Get("ttl"); ttlStr != "" {
		ttl, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid TTL", http.StatusBadRequest)
			return
		}
		
		if ttl > 0 {
			expiration = time.Now().Add(time.Duration(ttl) * time.Second).UnixNano()
		}
	}
	
	// Create the value
	value := storage.Value{
		Data:       data,
		Expiration: expiration,
	}
	
	// Set the key
	start := time.Now()
	err = s.node.Set(key, value)
	duration := time.Since(start)
	
	s.metrics.ObserveSetLatency(duration.Seconds())
	s.metrics.IncSet()
	
	if err != nil {
		if err == raft.ErrNotLeader {
			http.Error(w, "not the leader", http.StatusTemporaryRedirect)
			return
		}
		
		s.logger.Error("failed to set key", zap.String("key", key), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleDelete handles DELETE requests for a key
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	
	start := time.Now()
	err := s.node.Delete(key)
	duration := time.Since(start)
	
	s.metrics.ObserveDeleteLatency(duration.Seconds())
	s.metrics.IncDelete()
	
	if err != nil {
		if err == raft.ErrNotLeader {
			http.Error(w, "not the leader", http.StatusTemporaryRedirect)
			return
		}
		
		s.logger.Error("failed to delete key", zap.String("key", key), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleGetAll handles GET requests for all keys
func (s *Server) handleGetAll(w http.ResponseWriter, r *http.Request) {
	keys := s.node.Keys()
	
	response := struct {
		Keys []string `json:"keys"`
	}{
		Keys: keys,
	}
	
	// Return the keys as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRaftStatus returns the status of the Raft cluster
func (s *Server) handleRaftStatus(w http.ResponseWriter, r *http.Request) {
	status := struct {
		Leader  string `json:"leader"`
		IsLeader bool   `json:"is_leader"`
		NodeID   string `json:"node_id"`
	}{
		Leader:   s.node.Leader(),
		IsLeader: s.node.IsLeader(),
		NodeID:   s.node.ID,
	}
	
	// Return the status as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleRaftJoin handles POST requests to join the Raft cluster
func (s *Server) handleRaftJoin(w http.ResponseWriter, r *http.Request) {
	if !s.node.IsLeader() {
		http.Error(w, "not the leader", http.StatusTemporaryRedirect)
		return
	}
	
	var request struct {
		NodeID string `json:"node_id"`
		Addr   string `json:"addr"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if request.NodeID == "" || request.Addr == "" {
		http.Error(w, "node_id and addr are required", http.StatusBadRequest)
		return
	}
	
	// Add the node to the cluster
	err := s.node.AddNode(request.NodeID, request.Addr)
	if err != nil {
		s.logger.Error("failed to add node to cluster", 
			zap.String("node_id", request.NodeID), 
			zap.String("addr", request.Addr), 
			zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleHealth handles GET requests for health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}