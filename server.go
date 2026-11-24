package main

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
)

// Server serves the rate-limited HTTP API.
type Server struct {
	cfg   Settings
	store *Store
}

// NewServer returns the API handler.
func NewServer(cfg Settings, store *Store) http.Handler {
	s := &Server{cfg: cfg, store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /resource", s.handleResource)
	mux.HandleFunc("GET /status", s.handleStatus)
	return mux
}

// clientKey identifies the caller by the host part of RemoteAddr. It ignores
// X-Forwarded-For because any client can forge that header to dodge its limit.
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleResource(w http.ResponseWriter, r *http.Request) {
	allowed, remaining, retryAfter := s.store.Check(clientKey(r))
	reset := int(math.Ceil((s.cfg.Capacity - float64(remaining)) / s.cfg.RefillPerSec))
	h := w.Header()
	h.Set("X-RateLimit-Limit", strconv.FormatFloat(s.cfg.Capacity, 'f', -1, 64))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	h.Set("X-RateLimit-Reset", strconv.Itoa(reset))
	if !allowed {
		h.Set("Retry-After", strconv.Itoa(max(retryAfter, 1)))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limited"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]float64{
		"limit":     s.cfg.Capacity,
		"remaining": float64(s.store.Peek(clientKey(r))),
	})
}
