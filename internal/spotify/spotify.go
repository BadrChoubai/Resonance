// Package spotify is a minimal client for the Spotify Web API endpoints the
// app uses: the authorization-code token exchange, token refresh, and GET /me.
package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAccountsURL = "https://accounts.spotify.com"
	defaultAPIURL      = "https://api.spotify.com/v1"
)

// Client talks to Spotify's accounts service and Web API.
type Client struct {
	clientID     string
	clientSecret string
	redirectURI  string

	http        *http.Client
	accountsURL string
	apiURL      string
}

// New returns a Client for the app registered in the Spotify dashboard.
func New(clientID, clientSecret, redirectURI string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		http:         &http.Client{Timeout: 10 * time.Second},
		accountsURL:  defaultAccountsURL,
		apiURL:       defaultAPIURL,
	}
}

// Token is the result of an exchange or refresh. RefreshToken is empty when
// Spotify keeps the existing one on refresh.
type Token struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

// User is the subset of the current user's profile the app needs.
type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// Error is a non-2xx response from Spotify.
type Error struct {
	Status int
	Body   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("spotify: status %d: %s", e.Status, e.Body)
}

// AuthorizeURL returns the consent page URL the browser is sent to.
func (c *Client) AuthorizeURL(state string, scopes ...string) string {
	q := url.Values{
		"client_id":     {c.clientID},
		"response_type": {"code"},
		"redirect_uri":  {c.redirectURI},
		"state":         {state},
		"scope":         {strings.Join(scopes, " ")},
	}
	return c.accountsURL + "/authorize?" + q.Encode()
}

// Exchange trades an authorization code for tokens.
func (c *Client) Exchange(ctx context.Context, code string) (Token, error) {
	return c.token(ctx, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {c.redirectURI},
	})
}

// Refresh gets a new access token using a refresh token.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	return c.token(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

func (c *Client) token(ctx context.Context, form url.Values) (Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.accountsURL+"/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := c.do(req, &body); err != nil {
		return Token{}, err
	}
	return Token{
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(body.ExpiresIn) * time.Second),
	}, nil
}

// Me returns the profile of the user the access token belongs to.
func (c *Client) Me(ctx context.Context, accessToken string) (User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/me", nil)
	if err != nil {
		return User{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var u User
	if err := c.do(req, &u); err != nil {
		return User{}, err
	}
	return u, nil
}

func (c *Client) do(req *http.Request, v any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return &Error{Status: res.StatusCode, Body: string(b)}
	}
	return json.NewDecoder(res.Body).Decode(v)
}
