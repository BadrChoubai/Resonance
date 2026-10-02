// Package auth handles the Spotify authorization-code flow, sessions, and
// encrypted refresh-token storage. The browser only ever holds an httpOnly
// session cookie; Spotify tokens stay server-side.
package auth
