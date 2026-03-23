package builtin

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// --- RingBuffer tests ---

func TestRingBufferWrite(t *testing.T) {
	rb := NewRingBuffer(16)
	n, err := rb.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 5 {
		t.Errorf("expected n=5, got %d", n)
	}
	if rb.String() != "hello" {
		t.Errorf("expected 'hello', got %q", rb.String())
	}
}

func TestRingBufferOverflow(t *testing.T) {
	rb := NewRingBuffer(8)
	rb.Write([]byte("abcdefgh")) // fills exactly
	rb.Write([]byte("XY"))       // overwrites first 2 bytes

	got := rb.String()
	if got != "cdefghXY" {
		t.Errorf("expected 'cdefghXY', got %q", got)
	}
}

func TestRingBufferLen(t *testing.T) {
	rb := NewRingBuffer(8)
	if rb.Len() != 0 {
		t.Errorf("expected len 0, got %d", rb.Len())
	}
	rb.Write([]byte("abc"))
	if rb.Len() != 3 {
		t.Errorf("expected len 3, got %d", rb.Len())
	}
	rb.Write([]byte("defghijk")) // overflow
	if rb.Len() != 8 {
		t.Errorf("expected len 8, got %d", rb.Len())
	}
}

func TestRingBufferReset(t *testing.T) {
	rb := NewRingBuffer(16)
	rb.Write([]byte("hello"))
	rb.Reset()
	if rb.Len() != 0 {
		t.Errorf("expected len 0 after reset, got %d", rb.Len())
	}
	if rb.String() != "" {
		t.Errorf("expected empty string after reset, got %q", rb.String())
	}
}

func TestRingBufferConcurrentWrites(t *testing.T) {
	rb := NewRingBuffer(1024)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rb.Write([]byte("x"))
			}
		}()
	}
	wg.Wait()

	// Should not panic, and length should be <= buffer size
	if rb.Len() > 1024 {
		t.Errorf("len %d exceeds buffer size 1024", rb.Len())
	}
}

func TestRingBufferEmpty(t *testing.T) {
	rb := NewRingBuffer(8)
	if rb.String() != "" {
		t.Errorf("expected empty string, got %q", rb.String())
	}
}

// --- SessionManager tests ---

func TestSessionManagerStartAndRead(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()

	// Use a command that produces output and exits
	sess, err := sm.Start("test1", "echo", []string{"hello world"}, "")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if sess.ID != "test1" {
		t.Errorf("expected ID 'test1', got %q", sess.ID)
	}

	// Wait briefly for output
	time.Sleep(500 * time.Millisecond)

	output, err := sm.ReadOutput("test1")
	if err != nil {
		t.Fatalf("ReadOutput failed: %v", err)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected output containing 'hello world', got %q", output)
	}
}

func TestSessionManagerSendInput(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()

	// Start cat which echoes stdin to stdout
	_, err := sm.Start("cat1", "cat", nil, "")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = sm.SendInput("cat1", "ping")
	if err != nil {
		t.Fatalf("SendInput failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	output, err := sm.ReadOutput("cat1")
	if err != nil {
		t.Fatalf("ReadOutput failed: %v", err)
	}
	if !strings.Contains(output, "ping") {
		t.Errorf("expected output containing 'ping', got %q", output)
	}

	// Kill the cat process
	err = sm.Kill("cat1")
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
}

func TestSessionManagerKill(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()

	_, err := sm.Start("kill1", "cat", nil, "")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = sm.Kill("kill1")
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// Wait for process to exit
	time.Sleep(300 * time.Millisecond)

	sess, ok := sm.GetSession("kill1")
	if !ok {
		t.Fatal("session not found after kill")
	}
	if sess.State != SessionExited {
		t.Errorf("expected state %q, got %q", SessionExited, sess.State)
	}
}

func TestSessionManagerList(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()

	sm.Start("s1", "echo", []string{"a"}, "")
	sm.Start("s2", "echo", []string{"b"}, "")

	list := sm.List()
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

func TestSessionManagerMaxSessions(t *testing.T) {
	sm := NewSessionManager(2)
	defer sm.Cleanup()

	_, err := sm.Start("a", "cat", nil, "")
	if err != nil {
		t.Fatalf("Start a failed: %v", err)
	}
	_, err = sm.Start("b", "cat", nil, "")
	if err != nil {
		t.Fatalf("Start b failed: %v", err)
	}
	_, err = sm.Start("c", "cat", nil, "")
	if err == nil {
		t.Error("expected error for exceeding max sessions")
	}

	// Clean up
	sm.Kill("a")
	sm.Kill("b")
}

func TestSessionManagerNotFound(t *testing.T) {
	sm := NewSessionManager(5)

	_, err := sm.ReadOutput("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}

	err = sm.SendInput("nonexistent", "hello")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}

	err = sm.Kill("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestSessionManagerDuplicateID(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()

	_, err := sm.Start("dup", "echo", []string{"a"}, "")
	if err != nil {
		t.Fatalf("first Start failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	_, err = sm.Start("dup", "echo", []string{"b"}, "")
	if err == nil {
		t.Error("expected error for duplicate session ID")
	}

	sm.Kill("dup")
}

func TestSessionManagerGetSession(t *testing.T) {
	sm := NewSessionManager(5)
	defer sm.Cleanup()

	sm.Start("gs1", "echo", []string{"test"}, "")
	time.Sleep(200 * time.Millisecond)

	sess, ok := sm.GetSession("gs1")
	if !ok {
		t.Fatal("session not found")
	}
	if sess.Command != "echo" {
		t.Errorf("expected command 'echo', got %q", sess.Command)
	}

	_, ok = sm.GetSession("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent session")
	}
}
