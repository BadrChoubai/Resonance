// Package server wires HTTP routes for the API.
//
// Public routes live under /api, matching the path the Ingress forwards
// unchanged. Anything outside /api is never exposed by the Ingress.
package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/BadrChoubai/resonance/internal/auth"
	"github.com/BadrChoubai/resonance/internal/spotify"
)

type server struct {
	log     *slog.Logger
	auth    *auth.Service
	spotify *spotify.Client
}

// New returns the API's root handler.
func New(log *slog.Logger, a *auth.Service, sp *spotify.Client) http.Handler {
	s := &server{log: log, auth: a, spotify: sp}
	mux := http.NewServeMux()

	// Internal only; outside /api, so the Ingress never routes it.
	mux.HandleFunc("GET /healthz", handleHealthz)

	mux.HandleFunc("GET /api/hello", handleHello)
	mux.HandleFunc("GET /api/auth/login", a.HandleLogin)
	mux.HandleFunc("GET /api/auth/callback", a.HandleCallback)
	mux.HandleFunc("GET /api/me", s.handleMe)

	// TODO(milestone 3): /api/top-tracks

	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleHello(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "hello from resonance"})
}

func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.auth.UserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not logged in")
		return
	}
	token, err := s.auth.AccessToken(r.Context(), userID)
	if errors.Is(err, auth.ErrNoSession) {
		s.auth.ClearSession(w)
		writeError(w, http.StatusUnauthorized, "not logged in")
		return
	}
	if err != nil {
		s.log.Error("access token", "user", userID, "error", err)
		writeError(w, http.StatusBadGateway, "spotify unavailable")
		return
	}

	user, err := s.spotify.Me(r.Context(), token)
	if err != nil {
		s.log.Error("spotify me", "user", userID, "error", err)
		writeError(w, http.StatusBadGateway, "spotify unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"displayName": user.DisplayName})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
