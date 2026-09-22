package host

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type tokenRecord struct {
	expires time.Time
	userID  string
}

// MemoryTokenStore mints opaque bearer tokens for MCP authentication (development and simple hosts).
type MemoryTokenStore struct {
	mu     sync.Mutex
	ttl    time.Duration
	tokens map[string]tokenRecord
}

func NewMemoryTokenStore(ttl time.Duration) *MemoryTokenStore {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &MemoryTokenStore{
		ttl:    ttl,
		tokens: make(map[string]tokenRecord),
	}
}

func (s *MemoryTokenStore) Mint() (string, error) {
	return s.MintFor("")
}

func (s *MemoryTokenStore) MintFor(userID string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	expires := time.Now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	s.tokens[token] = tokenRecord{expires: expires, userID: userID}
	return token, nil
}

func (s *MemoryTokenStore) Validate(token string) bool {
	_, ok := s.Lookup(token)
	return ok
}

func (s *MemoryTokenStore) Lookup(token string) (userID string, ok bool) {
	if token == "" {
		return "", false
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	record, found := s.tokens[token]
	if !found || !now.Before(record.expires) {
		return "", false
	}
	return record.userID, true
}

func (s *MemoryTokenStore) pruneLocked(now time.Time) {
	for token, record := range s.tokens {
		if !now.Before(record.expires) {
			delete(s.tokens, token)
		}
	}
}
