package builtin

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// SessionState represents the state of an interactive session.
type SessionState string

const (
	SessionRunning      SessionState = "running"
	SessionWaitingInput SessionState = "waiting_input"
	SessionExited       SessionState = "exited"
)

// RingBuffer is a fixed-size circular buffer for capturing output.
type RingBuffer struct {
	buf  []byte
	size int
	pos  int
	full bool
	mu   sync.Mutex
}

// NewRingBuffer creates a new ring buffer with the given size.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buf:  make([]byte, size),
		size: size,
	}
}

// Write implements io.Writer for the ring buffer.
func (rb *RingBuffer) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	n := len(p)
	for _, b := range p {
		rb.buf[rb.pos] = b
		rb.pos = (rb.pos + 1) % rb.size
		if rb.pos == 0 && !rb.full {
			rb.full = true
		}
	}
	// Handle case where we wrote more than size in one call
	if n >= rb.size {
		rb.full = true
		rb.pos = n % rb.size
	}
	return n, nil
}

// String returns the current content of the ring buffer.
func (rb *RingBuffer) String() string {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full {
		return string(rb.buf[:rb.pos])
	}
	// When full, data starts at pos (oldest) and wraps around
	result := make([]byte, rb.size)
	copy(result, rb.buf[rb.pos:])
	copy(result[rb.size-rb.pos:], rb.buf[:rb.pos])
	return string(result)
}

// Reset clears the ring buffer.
func (rb *RingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.pos = 0
	rb.full = false
}

// Len returns the number of bytes currently in the buffer.
func (rb *RingBuffer) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.full {
		return rb.size
	}
	return rb.pos
}

// InteractiveSession represents a running interactive process.
type InteractiveSession struct {
	ID       string
	Command  string
	Args     []string
	WorkDir  string
	Started  time.Time
	State    SessionState
	ExitCode int

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *RingBuffer
	stderr *RingBuffer
	mu     sync.Mutex
	done   chan struct{}
}

// SessionManager manages multiple interactive sessions.
type SessionManager struct {
	sessions    map[string]*InteractiveSession
	mu          sync.RWMutex
	maxSessions int
}

// NewSessionManager creates a new session manager with the given limit.
func NewSessionManager(maxSessions int) *SessionManager {
	return &SessionManager{
		sessions:    make(map[string]*InteractiveSession),
		maxSessions: maxSessions,
	}
}

// Start creates and starts a new interactive session.
func (sm *SessionManager) Start(id, command string, args []string, workDir string) (*InteractiveSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[id]; exists {
		return nil, fmt.Errorf("session %q already exists", id)
	}

	activeCount := 0
	for _, s := range sm.sessions {
		s.mu.Lock()
		state := s.State
		s.mu.Unlock()
		if state != SessionExited {
			activeCount++
		}
	}
	if activeCount >= sm.maxSessions {
		return nil, fmt.Errorf("max sessions (%d) reached", sm.maxSessions)
	}

	cmd := exec.Command(command, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdoutBuf := NewRingBuffer(64 * 1024) // 64KB
	stderrBuf := NewRingBuffer(16 * 1024) // 16KB

	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting command: %w", err)
	}

	sess := &InteractiveSession{
		ID:      id,
		Command: command,
		Args:    args,
		WorkDir: workDir,
		Started: time.Now(),
		State:   SessionRunning,
		cmd:     cmd,
		stdin:   stdinPipe,
		stdout:  stdoutBuf,
		stderr:  stderrBuf,
		done:    make(chan struct{}),
	}

	// Watch for process exit in background
	go func() {
		err := cmd.Wait()
		sess.mu.Lock()
		sess.State = SessionExited
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				sess.ExitCode = exitErr.ExitCode()
			} else {
				sess.ExitCode = -1
			}
		}
		sess.mu.Unlock()
		close(sess.done)
	}()

	sm.sessions[id] = sess
	return sess, nil
}

// SendInput writes input to a session's stdin.
func (sm *SessionManager) SendInput(id, input string) error {
	sm.mu.RLock()
	sess, ok := sm.sessions[id]
	sm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %q not found", id)
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.State == SessionExited {
		return fmt.Errorf("session %q has exited", id)
	}

	// Append newline if not present
	if len(input) == 0 || input[len(input)-1] != '\n' {
		input += "\n"
	}

	_, err := io.WriteString(sess.stdin, input)
	return err
}

// ReadOutput returns the current stdout content from a session.
func (sm *SessionManager) ReadOutput(id string) (string, error) {
	sm.mu.RLock()
	sess, ok := sm.sessions[id]
	sm.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("session %q not found", id)
	}

	output := sess.stdout.String()
	errOutput := sess.stderr.String()

	if errOutput != "" {
		output += "\n[stderr]\n" + errOutput
	}

	return output, nil
}

// GetSession returns a copy of the session's public state.
func (sm *SessionManager) GetSession(id string) (InteractiveSession, bool) {
	sm.mu.RLock()
	sess, ok := sm.sessions[id]
	sm.mu.RUnlock()

	if !ok {
		return InteractiveSession{}, false
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	return InteractiveSession{
		ID:       sess.ID,
		Command:  sess.Command,
		Args:     sess.Args,
		WorkDir:  sess.WorkDir,
		Started:  sess.Started,
		State:    sess.State,
		ExitCode: sess.ExitCode,
	}, true
}

// Kill terminates a session's process.
func (sm *SessionManager) Kill(id string) error {
	sm.mu.RLock()
	sess, ok := sm.sessions[id]
	sm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %q not found", id)
	}

	sess.mu.Lock()
	if sess.State == SessionExited {
		sess.mu.Unlock()
		return nil
	}
	sess.mu.Unlock()

	// Close stdin first to signal EOF
	sess.stdin.Close()

	// Kill the process directly (Windows has no SIGTERM)
	if sess.cmd.Process != nil {
		sess.cmd.Process.Kill()
	}

	// Wait for exit with timeout
	select {
	case <-sess.done:
	case <-time.After(3 * time.Second):
		// Force kill already sent above
	}

	return nil
}

// List returns copies of all sessions.
func (sm *SessionManager) List() []InteractiveSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]InteractiveSession, 0, len(sm.sessions))
	for _, sess := range sm.sessions {
		sess.mu.Lock()
		result = append(result, InteractiveSession{
			ID:       sess.ID,
			Command:  sess.Command,
			Args:     sess.Args,
			WorkDir:  sess.WorkDir,
			Started:  sess.Started,
			State:    sess.State,
			ExitCode: sess.ExitCode,
		})
		sess.mu.Unlock()
	}
	return result
}

// Cleanup kills all sessions and frees resources.
func (sm *SessionManager) Cleanup() {
	sm.mu.RLock()
	ids := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		ids = append(ids, id)
	}
	sm.mu.RUnlock()

	for _, id := range ids {
		sm.Kill(id)
	}
}
