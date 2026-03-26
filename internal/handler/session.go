package handler

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// SessionStore manages short-lived auth sessions for paid upload flows.
// A session is created after wallet signature verification (Sign #1) and
// can be reused for subsequent requests (e.g. PUT /api/videos/{id})
// without requiring an additional wallet signature.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
	ttl      time.Duration
}

type session struct {
	address   string
	createdAt time.Time
}

// NewSessionStore creates a session store with the given TTL and starts
// a background goroutine that purges expired sessions every minute.
func NewSessionStore(ttl time.Duration) *SessionStore {
	s := &SessionStore{
		sessions: make(map[string]*session),
		ttl:      ttl,
	}
	go s.cleanup()
	return s
}

// Create generates a cryptographically random session token bound to the
// given wallet address and returns it.
func (s *SessionStore) Create(address string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	s.mu.Lock()
	s.sessions[token] = &session{
		address:   address,
		createdAt: time.Now(),
	}
	s.mu.Unlock()

	return token
}

// Validate checks whether the token exists and has not expired.
// Returns the bound wallet address and true if valid.
func (s *SessionStore) Validate(token string) (string, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[token]
	s.mu.RUnlock()

	if !ok {
		return "", false
	}
	if time.Since(sess.createdAt) > s.ttl {
		s.mu.Lock()
		delete(s.sessions, token)
		s.mu.Unlock()
		return "", false
	}
	return sess.address, true
}

func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, sess := range s.sessions {
			if now.Sub(sess.createdAt) > s.ttl {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}
