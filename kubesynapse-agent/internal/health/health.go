// Package health provides HTTP health check endpoints for Kubernetes probes.
package health

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

// Server exposes health and readiness endpoints.
type Server struct {
	port    int
	ready   atomic.Bool
	metrics *Metrics
}

// Metrics tracks basic operational counts.
type Metrics struct {
	AlertsFired  atomic.Int64
	ErrorsTotal  atomic.Int64
	PodsWatched  atomic.Int64
	WebhooksSent atomic.Int64
}

// New creates a health server on the specified port.
func New(port int) *Server {
	return &Server{
		port:    port,
		metrics: &Metrics{},
	}
}

// SetReady marks the server as ready (K8s readiness probe will pass).
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// GetMetrics returns the metrics tracker.
func (s *Server) GetMetrics() *Metrics {
	return s.metrics
}

// Start begins serving health endpoints in a goroutine.
func (s *Server) Start() {
	mux := http.NewServeMux()

	// Liveness probe — always returns 200 if the process is running
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"alive"}`)
	})

	// Readiness probe — returns 200 only when K8s API connection is established
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.ready.Load() {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"status":"ready"}`)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, `{"status":"not ready"}`)
		}
	})

	// Prometheus-compatible metrics
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data := map[string]int64{
			"alerts_fired":  s.metrics.AlertsFired.Load(),
			"errors_total":  s.metrics.ErrorsTotal.Load(),
			"pods_watched":  s.metrics.PodsWatched.Load(),
			"webhooks_sent": s.metrics.WebhooksSent.Load(),
		}
		json.NewEncoder(w).Encode(data)
	})

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[health] Starting health server on %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("[health] Server error: %v", err)
		}
	}()
}
