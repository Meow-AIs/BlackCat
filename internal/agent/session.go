package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/meowai/blackcat/internal/llm"
)

// ActiveSession extends Session with runtime state.
type ActiveSession struct {
	Session  Session       `json:"session"`
	Messages []llm.Message `json:"messages"`
	Plan     *Plan         `json:"plan,omitempty"`
}

// SessionManager provides thread-safe session CRUD operations.
type SessionManager struct {
	sessions map[string]*ActiveSession
	mu       sync.RWMutex
	counter  int64
}

// NewSessionManager creates an empty session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*ActiveSession),
	}
}

// Create starts a new session for the given project and user.
func (sm *SessionManager) Create(projectID, userID string) *ActiveSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.counter++
	id := fmt.Sprintf("sess-%d-%d", time.Now().UnixNano(), sm.counter)

	session := &ActiveSession{
		Session: Session{
			ID:        id,
			ProjectID: projectID,
			UserID:    userID,
			State:     StateIdle,
			CreatedAt: time.Now().Unix(),
		},
		Messages: []llm.Message{},
	}

	sm.sessions[id] = session
	return session
}

// Get retrieves a session by ID. Returns the session and true if found.
func (sm *SessionManager) Get(id string) (*ActiveSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.sessions[id]
	return s, ok
}

// AddMessage appends a message to the specified session's history.
func (sm *SessionManager) AddMessage(sessionID string, msg llm.Message) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	s.Messages = append(s.Messages, msg)
	return nil
}

// List returns all sessions as a slice. The returned slice is a new
// allocation and safe to iterate without holding the lock.
func (sm *SessionManager) List() []Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		result = append(result, s.Session)
	}
	return result
}

// Delete removes a session by ID. Returns an error if not found.
func (sm *SessionManager) Delete(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessions[id]; !ok {
		return fmt.Errorf("session %q not found", id)
	}

	delete(sm.sessions, id)
	return nil
}
