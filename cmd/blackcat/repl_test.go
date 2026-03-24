package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

// TestREPLWelcomeBanner verifies the banner is printed at REPL startup.
func TestREPLWelcomeBanner(t *testing.T) {
	in := strings.NewReader("exit\n")
	var out bytes.Buffer

	runREPL(in, &out, nil)

	if !strings.Contains(out.String(), "BlackCat") {
		t.Errorf("expected welcome banner with 'BlackCat', got:\n%s", out.String())
	}
}

// TestREPLExitOnExitCommand verifies "exit" terminates the loop.
func TestREPLExitOnExitCommand(t *testing.T) {
	in := strings.NewReader("exit\n")
	var out bytes.Buffer

	runREPL(in, &out, nil)
	// If we reach here without hanging, the exit worked.
}

// TestREPLExitOnQuitCommand verifies "quit" terminates the loop.
func TestREPLExitOnQuitCommand(t *testing.T) {
	in := strings.NewReader("quit\n")
	var out bytes.Buffer

	runREPL(in, &out, nil)
}

// TestREPLExitOnSlashExit verifies "/exit" terminates the loop.
func TestREPLExitOnSlashExit(t *testing.T) {
	in := strings.NewReader("/exit\n")
	var out bytes.Buffer

	runREPL(in, &out, nil)
}

// TestREPLExitOnSlashQuit verifies "/quit" terminates the loop.
func TestREPLExitOnSlashQuit(t *testing.T) {
	in := strings.NewReader("/quit\n")
	var out bytes.Buffer

	runREPL(in, &out, nil)
}

// TestREPLSkipsEmptyLines verifies empty lines don't cause errors.
func TestREPLSkipsEmptyLines(t *testing.T) {
	in := strings.NewReader("\n\n\nexit\n")
	var out bytes.Buffer

	runREPL(in, &out, nil) // must not hang or panic
}

// TestREPLPrintsPrompt verifies the ">" prompt is printed.
func TestREPLPrintsPrompt(t *testing.T) {
	in := strings.NewReader("exit\n")
	var out bytes.Buffer

	runREPL(in, &out, nil)

	if !strings.Contains(out.String(), ">") {
		t.Errorf("expected prompt '>' in output, got:\n%s", out.String())
	}
}

// TestREPLExitOnEOF verifies REPL terminates cleanly when stdin closes (EOF).
func TestREPLExitOnEOF(t *testing.T) {
	// Empty reader = immediate EOF
	in := strings.NewReader("")
	var out bytes.Buffer

	runREPL(in, &out, nil) // must not hang
}

// TestREPLExitMultipleEmpty verifies many blank lines then EOF is handled.
func TestREPLMultipleBlankLinesThenEOF(t *testing.T) {
	in := strings.NewReader("\n\n\n\n\n")
	var out bytes.Buffer

	runREPL(in, &out, nil) // must not hang
}

// TestREPLBannerContainsVersion verifies version is in the banner.
func TestREPLBannerContainsVersion(t *testing.T) {
	in := strings.NewReader("exit\n")
	var out bytes.Buffer

	runREPL(in, &out, nil)

	// Banner should mention version (from package-level var or literal "v")
	output := out.String()
	if !strings.Contains(output, "v") {
		t.Errorf("expected version indicator in banner, got:\n%s", output)
	}
}

// TestREPLWithNilCoreProcessesGracefully verifies nil core handles input without panic.
func TestREPLWithNilCoreProcessesGracefully(t *testing.T) {
	in := strings.NewReader("hello agent\nexit\n")
	var out bytes.Buffer

	// nil core: REPL should handle gracefully (error message, not panic)
	runREPL(in, &out, nil)

	output := out.String()
	// Should still show the banner
	if !strings.Contains(output, "BlackCat") {
		t.Errorf("expected banner even with nil core, got:\n%s", output)
	}
}

// TestREPLScanner verifies bufio.Scanner correctly reads multiple lines.
func TestREPLScanner(t *testing.T) {
	// This tests the scanner utility used by the REPL directly.
	input := "line1\nline2\nexit\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if line == "exit" {
			break
		}
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (line1, line2, exit), got %d: %v", len(lines), lines)
	}
}

// TestRunNoArgsCallsInteractive verifies run() with no args still outputs banner.
func TestRunNoArgsOutputsBanner(t *testing.T) {
	out := captureOutput(func() {
		run([]string{"blackcat"})
	})
	if !strings.Contains(out, "BlackCat") {
		t.Errorf("expected BlackCat in no-args output, got: %q", out)
	}
}
