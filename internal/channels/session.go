package channels

import "sync"

// SessionManager tracks per-user sessions across channels.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]string // userKey → sessionID
}

// NewSessionManager creates a session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]string)}
}

// Get returns the session ID for a user, or empty if none.
func (sm *SessionManager) Get(userKey string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[userKey]
}

// Set assigns a session ID to a user.
func (sm *SessionManager) Set(userKey, sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[userKey] = sessionID
}

// Remove deletes a user's session.
func (sm *SessionManager) Remove(userKey string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, userKey)
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}
