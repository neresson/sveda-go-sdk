package host

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// MemoryTokenStore mints opaque bearer tokens for MCP authentication (development and simple hosts).
type MemoryTokenStore struct {
	mu     sync.Mutex
	ttl    time.Duration
	tokens map[string]time.Time
}

func NewMemoryTokenStore(ttl time.Duration) *MemoryTokenStore {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &MemoryTokenStore{
		ttl:    ttl,
		tokens: make(map[string]time.Time),
	}
}

func (s *MemoryTokenStore) Mint() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	expires := time.Now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	s.tokens[token] = expires
	return token, nil
}

func (s *MemoryTokenStore) Validate(token string) bool {
	if token == "" {
		return false
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	expires, ok := s.tokens[token]
	return ok && now.Before(expires)
}

func (s *MemoryTokenStore) pruneLocked(now time.Time) {
	for token, expires := range s.tokens {
		if !now.Before(expires) {
			delete(s.tokens, token)
		}
	}
}
