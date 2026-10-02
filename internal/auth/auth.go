// Package auth handles the Spotify authorization-code flow, sessions, and
// encrypted refresh-token storage. The browser only ever holds an httpOnly
// session cookie; Spotify tokens stay server-side.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/BadrChoubai/resonance/internal/spotify"
)

const (
	sessionCookie = "session"
	stateCookie   = "oauth_state"

	sessionTTL = 30 * 24 * time.Hour
	stateTTL   = 10 * time.Minute

	// Refresh a little early so a token doesn't expire mid-request.
	expiryLeeway = time.Minute
)

// Scopes requested from Spotify.
var scopes = []string{"user-top-read"}

// ErrNoSession means the user has no usable Spotify authorization and needs
// to log in again.
var ErrNoSession = errors.New("auth: no session")

// Service runs the login flow and hands out access tokens for logged-in users.
type Service struct {
	log     *slog.Logger
	spotify *spotify.Client
	store   Store
	keys    keys
	secure  bool

	mu     sync.Mutex
	access map[string]spotify.Token // short-lived access tokens; never persisted
}

// New returns a Service. sessionKey is the decoded SESSION_KEY; secure sets
// the Secure attribute on cookies and should be true whenever the app is
// served over HTTPS.
func New(log *slog.Logger, sp *spotify.Client, store Store, sessionKey []byte, secure bool) (*Service, error) {
	k, err := deriveKeys(sessionKey)
	if err != nil {
		return nil, err
	}
	return &Service{
		log:     log,
		spotify: sp,
		store:   store,
		keys:    k,
		secure:  secure,
		access:  make(map[string]spotify.Token),
	}, nil
}

// HandleLogin sends the browser to Spotify's consent page.
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state := rand.Text()
	http.SetCookie(w, s.cookie(stateCookie, state, "/api/auth", stateTTL))
	http.Redirect(w, r, s.spotify.AuthorizeURL(state, scopes...), http.StatusFound)
}

// HandleCallback finishes the login Spotify redirects back to, then sends
// the browser to the app. Failures land on /?login=failed.
func (s *Service) HandleCallback(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, s.cookie(stateCookie, "", "/api/auth", -1))

	userID, err := s.callback(r)
	if err != nil {
		s.log.Warn("login failed", "error", err)
		http.Redirect(w, r, "/?login=failed", http.StatusFound)
		return
	}
	value := s.keys.signSession(userID, time.Now().Add(sessionTTL))
	http.SetCookie(w, s.cookie(sessionCookie, value, "/", sessionTTL))
	http.Redirect(w, r, "/", http.StatusFound)
}

// callback verifies state, exchanges the code, and stores the refresh token.
// It returns the logged-in user's Spotify ID.
func (s *Service) callback(r *http.Request) (string, error) {
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		return "", errors.New("spotify returned error: " + e)
	}
	c, err := r.Cookie(stateCookie)
	if err != nil || subtle.ConstantTimeCompare([]byte(c.Value), []byte(q.Get("state"))) != 1 {
		return "", errors.New("state mismatch")
	}

	ctx := r.Context()
	tok, err := s.spotify.Exchange(ctx, q.Get("code"))
	if err != nil {
		return "", err
	}
	user, err := s.spotify.Me(ctx, tok.AccessToken)
	if err != nil {
		return "", err
	}
	if err := s.store.Put(ctx, user.ID, s.keys.seal(user.ID, tok.RefreshToken)); err != nil {
		return "", err
	}

	s.mu.Lock()
	s.access[user.ID] = tok
	s.mu.Unlock()
	return user.ID, nil
}

// UserID returns the Spotify user ID from a valid session cookie.
func (s *Service) UserID(r *http.Request) (string, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return "", false
	}
	return s.keys.verifySession(c.Value, time.Now())
}

// ClearSession removes the session cookie.
func (s *Service) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, s.cookie(sessionCookie, "", "/", -1))
}

// AccessToken returns a valid access token for userID, refreshing it if
// needed. It returns ErrNoSession if the user must log in again.
func (s *Service) AccessToken(ctx context.Context, userID string) (string, error) {
	s.mu.Lock()
	tok, ok := s.access[userID]
	s.mu.Unlock()
	if ok && time.Now().Add(expiryLeeway).Before(tok.Expiry) {
		return tok.AccessToken, nil
	}

	sealed, err := s.store.Get(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return "", ErrNoSession
	}
	if err != nil {
		return "", err
	}
	refresh, err := s.keys.open(userID, sealed)
	if err != nil {
		// Unreadable, e.g. SESSION_KEY changed; make the user log in again.
		s.log.Warn("cannot decrypt refresh token", "user", userID, "error", err)
		return "", ErrNoSession
	}

	tok, err = s.spotify.Refresh(ctx, refresh)
	var se *spotify.Error
	if errors.As(err, &se) && se.Status == http.StatusBadRequest {
		// invalid_grant: the user revoked access or the token expired.
		_ = s.store.Delete(ctx, userID)
		return "", ErrNoSession
	}
	if err != nil {
		return "", err
	}
	if tok.RefreshToken != "" {
		if err := s.store.Put(ctx, userID, s.keys.seal(userID, tok.RefreshToken)); err != nil {
			return "", err
		}
	}

	s.mu.Lock()
	s.access[userID] = tok
	s.mu.Unlock()
	return tok.AccessToken, nil
}

func (s *Service) cookie(name, value, path string, ttl time.Duration) *http.Cookie {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	}
	if ttl < 0 {
		c.MaxAge = -1
	} else {
		c.MaxAge = int(ttl.Seconds())
	}
	return c
}
