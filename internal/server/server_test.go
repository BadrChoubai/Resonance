package server

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BadrChoubai/resonance/internal/auth"
	"github.com/BadrChoubai/resonance/internal/spotify"
)

func TestRoutes(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	sp := spotify.New("id", "secret", "http://127.0.0.1:5173/api/auth/callback")
	a, err := auth.New(log, sp, auth.NewMemoryStore(), bytes.Repeat([]byte{1}, 32), false)
	if err != nil {
		t.Fatal(err)
	}
	h := New(log, a, sp)

	tests := []struct {
		path string
		want int
	}{
		{"/healthz", http.StatusOK},
		{"/api/hello", http.StatusOK},
		{"/hello", http.StatusNotFound},
		{"/api/auth/login", http.StatusFound},
		{"/api/me", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if rec.Code != tt.want {
			t.Errorf("GET %s = %d, want %d", tt.path, rec.Code, tt.want)
		}
	}
}
