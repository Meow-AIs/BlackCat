package agent

import (
	"sync"
	"testing"

	"github.com/meowai/blackcat/internal/llm"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager()
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}
}

func TestSessionManager_Create(t *testing.T) {
	sm := NewSessionManager()

	s := sm.Create("proj-1", "user-1")
	if s == nil {
		t.Fatal("Create returned nil")
	}
	if s.Session.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q, want %q", s.Session.ProjectID, "proj-1")
	}
	if s.Session.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", s.Session.UserID, "user-1")
	}
	if s.Session.State != StateIdle {
		t.Errorf("State = %q, want %q", s.Session.State, StateIdle)
	}
	if s.Session.ID == "" {
		t.Error("ID should not be empty")
	}
	if s.Session.CreatedAt == 0 {
		t.Error("CreatedAt should not be zero")
	}
}

func TestSessionManager_Get(t *testing.T) {
	sm := NewSessionManager()

	s := sm.Create("proj-1", "user-1")
	got, ok := sm.Get(s.Session.ID)
	if !ok {
		t.Fatal("Get returned false for existing session")
	}
	if got.Session.ID != s.Session.ID {
		t.Errorf("ID = %q, want %q", got.Session.ID, s.Session.ID)
	}
}

func TestSessionManager_Get_NotFound(t *testing.T) {
	sm := NewSessionManager()

	_, ok := sm.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent session")
	}
}

func TestSessionManager_AddMessage(t *testing.T) {
	sm := NewSessionManager()

	s := sm.Create("proj-1", "user-1")
	err := sm.AddMessage(s.Session.ID, llm.Message{Role: llm.RoleUser, Content: "Hello"})
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	got, _ := sm.Get(s.Session.ID)
	if len(got.Messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(got.Messages))
	}
	if got.Messages[0].Content != "Hello" {
		t.Errorf("message content = %q, want %q", got.Messages[0].Content, "Hello")
	}
}

func TestSessionManager_AddMessage_NotFound(t *testing.T) {
	sm := NewSessionManager()

	err := sm.AddMessage("nonexistent", llm.Message{Role: llm.RoleUser, Content: "Hello"})
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestSessionManager_List(t *testing.T) {
	sm := NewSessionManager()

	sm.Create("proj-1", "user-1")
	sm.Create("proj-2", "user-2")

	sessions := sm.List()
	if len(sessions) != 2 {
		t.Errorf("got %d sessions, want 2", len(sessions))
	}
}

func TestSessionManager_List_Empty(t *testing.T) {
	sm := NewSessionManager()

	sessions := sm.List()
	if len(sessions) != 0 {
		t.Errorf("got %d sessions, want 0", len(sessions))
	}
}

func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager()

	s := sm.Create("proj-1", "user-1")
	err := sm.Delete(s.Session.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, ok := sm.Get(s.Session.ID)
	if ok {
		t.Error("session should be deleted")
	}
}

func TestSessionManager_Delete_NotFound(t *testing.T) {
	sm := NewSessionManager()

	err := sm.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestSessionManager_ConcurrentAccess(t *testing.T) {
	sm := NewSessionManager()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := sm.Create("proj", "user")
			_ = sm.AddMessage(s.Session.ID, llm.Message{Role: llm.RoleUser, Content: "msg"})
			_, _ = sm.Get(s.Session.ID)
			_ = sm.List()
		}()
	}
	wg.Wait()

	sessions := sm.List()
	if len(sessions) != 50 {
		t.Errorf("got %d sessions, want 50", len(sessions))
	}
}
