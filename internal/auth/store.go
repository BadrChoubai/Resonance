package auth

import (
	"context"
	"errors"
	"sync"
)

// ErrNotFound is returned by a Store when no token is stored for a user.
var ErrNotFound = errors.New("auth: no token for user")

// Store holds each user's encrypted refresh token, keyed by Spotify user ID.
// Values are already encrypted; a Store never sees plaintext tokens.
type Store interface {
	Get(ctx context.Context, userID string) ([]byte, error)
	Put(ctx context.Context, userID string, encryptedRefreshToken []byte) error
	Delete(ctx context.Context, userID string) error
}

// MemoryStore is a Store that lives only as long as the process, so every
// user has to log in again after a restart.
type MemoryStore struct {
	mu     sync.Mutex
	tokens map[string][]byte
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tokens: make(map[string][]byte)}
}

func (s *MemoryStore) Get(_ context.Context, userID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.tokens[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func (s *MemoryStore) Put(_ context.Context, userID string, encryptedRefreshToken []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[userID] = encryptedRefreshToken
	return nil
}

func (s *MemoryStore) Delete(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, userID)
	return nil
}
