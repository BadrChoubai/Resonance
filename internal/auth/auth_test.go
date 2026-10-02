package auth

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/BadrChoubai/resonance/internal/spotify"
)

func testKeys(t *testing.T) keys {
	t.Helper()
	k, err := deriveKeys(bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestDeriveKeysRejectsShortKey(t *testing.T) {
	if _, err := deriveKeys(make([]byte, 16)); err == nil {
		t.Fatal("want error for 16-byte key")
	}
}

func TestSealOpen(t *testing.T) {
	k := testKeys(t)
	sealed := k.seal("alice", "refresh-token")

	got, err := k.open("alice", sealed)
	if err != nil || got != "refresh-token" {
		t.Fatalf("open = %q, %v", got, err)
	}
	if _, err := k.open("bob", sealed); err == nil {
		t.Error("token sealed for alice opened as bob")
	}
}

func TestSession(t *testing.T) {
	k := testKeys(t)
	now := time.Now()
	v := k.signSession("alice", now.Add(time.Hour))

	if id, ok := k.verifySession(v, now); !ok || id != "alice" {
		t.Errorf("verify = %q, %v", id, ok)
	}
	if _, ok := k.verifySession(v, now.Add(2*time.Hour)); ok {
		t.Error("expired session accepted")
	}

	// Swap in another user ID but keep the original signature.
	forged := strings.Replace(v, v[:strings.IndexByte(v, '.')], "Ym9i", 1)
	if _, ok := k.verifySession(forged, now); ok {
		t.Error("forged session accepted")
	}
	if _, ok := k.verifySession("garbage", now); ok {
		t.Error("garbage accepted")
	}
}

func TestLoginThenCallbackRejectsWrongState(t *testing.T) {
	s, err := New(slog.New(slog.NewTextHandler(io.Discard, nil)),
		spotify.New("id", "secret", "http://127.0.0.1:5173/api/auth/callback"),
		NewMemoryStore(), bytes.Repeat([]byte{1}, 32), false)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	s.HandleLogin(rec, httptest.NewRequest(http.MethodGet, "/api/auth/login", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("login status = %d", rec.Code)
	}
	loc, _ := url.Parse(rec.Header().Get("Location"))
	state := loc.Query().Get("state")
	if state == "" {
		t.Fatal("no state in authorize URL")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/callback?code=abc&state=wrong", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	rec = httptest.NewRecorder()
	s.HandleCallback(rec, req)

	if got := rec.Header().Get("Location"); got != "/?login=failed" {
		t.Errorf("redirect = %q, want /?login=failed", got)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie && c.MaxAge > 0 {
			t.Error("session cookie set despite state mismatch")
		}
	}
}
