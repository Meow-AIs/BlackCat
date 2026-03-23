package mcp

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

// StdioTransport communicates with an MCP server process via stdin/stdout.
type StdioTransport struct {
	command string
	args    []string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
	started bool
}

// NewStdioTransport creates a new stdio transport for the given command.
// Call Start() to launch the subprocess.
func NewStdioTransport(command string, args ...string) *StdioTransport {
	return &StdioTransport{
		command: command,
		args:    args,
	}
}

// Start launches the subprocess and sets up the stdin/stdout pipes.
func (t *StdioTransport) Start() error {
	t.cmd = exec.Command(t.command, t.args...)
	t.cmd.Env = filteredEnvForMCP()

	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	t.stdout = bufio.NewScanner(stdout)

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	t.started = true
	return nil
}

// Send writes a message followed by a newline to the subprocess stdin.
func (t *StdioTransport) Send(msg []byte) error {
	if !t.started {
		return fmt.Errorf("transport not started: call Start() first")
	}
	_, err := fmt.Fprintf(t.stdin, "%s\n", msg)
	if err != nil {
		return fmt.Errorf("write to stdin: %w", err)
	}
	return nil
}

// Receive reads the next line from the subprocess stdout.
func (t *StdioTransport) Receive() ([]byte, error) {
	if !t.started {
		return nil, fmt.Errorf("transport not started: call Start() first")
	}
	if !t.stdout.Scan() {
		if err := t.stdout.Err(); err != nil {
			return nil, fmt.Errorf("read from stdout: %w", err)
		}
		return nil, fmt.Errorf("subprocess closed stdout")
	}
	// Return a copy of the bytes so they remain valid after the next Scan
	data := t.stdout.Bytes()
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// Close terminates the subprocess and cleans up resources.
func (t *StdioTransport) Close() error {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Wait()
	}
	return nil
}
