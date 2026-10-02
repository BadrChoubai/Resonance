package spotify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	c := New("id", "secret", "http://127.0.0.1:5173/api/auth/callback")
	c.accountsURL = srv.URL
	c.apiURL = srv.URL
	return c
}

func TestAuthorizeURL(t *testing.T) {
	c := New("id", "secret", "http://127.0.0.1:5173/api/auth/callback")
	u, err := url.Parse(c.AuthorizeURL("xyz", "user-top-read"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	for k, want := range map[string]string{
		"client_id":     "id",
		"response_type": "code",
		"redirect_uri":  "http://127.0.0.1:5173/api/auth/callback",
		"state":         "xyz",
		"scope":         "user-top-read",
	} {
		if got := q.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestExchange(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, secret, ok := r.BasicAuth()
		if r.URL.Path != "/api/token" || !ok || id != "id" || secret != "secret" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "authorization_code" || r.FormValue("code") != "abc" {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		w.Write([]byte(`{"access_token":"at","refresh_token":"rt","expires_in":3600}`))
	}))

	tok, err := c.Exchange(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "at" || tok.RefreshToken != "rt" || tok.Expiry.IsZero() {
		t.Errorf("got %+v", tok)
	}
}

func TestMeError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"expired"}`, http.StatusUnauthorized)
	}))

	_, err := c.Me(context.Background(), "at")
	var se *Error
	if !errors.As(err, &se) || se.Status != http.StatusUnauthorized {
		t.Fatalf("err = %v, want *Error with 401", err)
	}
}
