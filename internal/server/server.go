// Package server wires HTTP routes for the API.
//
// Public routes live under /api, matching the path the Ingress forwards
// unchanged. Anything outside /api is never exposed by the Ingress.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// New returns the API's root handler.
func New(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Internal only; outside /api, so the Ingress never routes it.
	mux.HandleFunc("GET /healthz", handleHealthz)

	mux.HandleFunc("GET /api/hello", handleHello)

	// TODO(milestone 2): /api/auth/login, /api/auth/callback, /api/me
	// TODO(milestone 3): /api/top-tracks

	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleHello(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "hello from resonance"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
