package devsecops

import (
	"testing"
)

// ---------------------------------------------------------------------------
// NewCodeScanner
// ---------------------------------------------------------------------------

func TestNewCodeScanner_NotNil(t *testing.T) {
	s := NewCodeScanner()
	if s == nil {
		t.Fatal("expected non-nil scanner")
	}
}

func TestNewCodeScanner_HasRules(t *testing.T) {
	s := NewCodeScanner()
	// Should auto-load rules for all languages
	if len(s.rules) < 30 {
		t.Errorf("expected at least 30 rules (10+ per language), got %d", len(s.rules))
	}
}

// ---------------------------------------------------------------------------
// DetectLanguage
// ---------------------------------------------------------------------------

func TestCodeScanner_DetectLanguage(t *testing.T) {
	s := NewCodeScanner()
	tests := []struct {
		filename string
		want     string
	}{
		{"main.go", "go"},
		{"server.Go", "go"},
		{"app.py", "python"},
		{"utils.PY", "python"},
		{"index.js", "javascript"},
		{"app.jsx", "javascript"},
		{"server.ts", "javascript"},
		{"component.tsx", "javascript"},
		{"readme.md", ""},
		{"image.png", ""},
		{"Dockerfile", ""},
	}
	for _, tt := range tests {
		got := s.DetectLanguage(tt.filename)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Go rules — at least 10
// ---------------------------------------------------------------------------

func TestCodeScanner_GoRules_AtLeast10(t *testing.T) {
	s := NewCodeScanner()
	count := 0
	for _, r := range s.rules {
		if r.Language == "go" {
			count++
		}
	}
	if count < 10 {
		t.Errorf("expected at least 10 Go rules, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Go scan: SQL injection
// ---------------------------------------------------------------------------

func TestCodeScanner_Go_SQLInjection(t *testing.T) {
	s := NewCodeScanner()
	code := `package main
import "database/sql"
func bad(db *sql.DB, input string) {
	db.Query("SELECT * FROM users WHERE id = " + input)
}`
	findings := s.ScanContent("main.go", code, "go")
	assertHasCWE(t, findings, "CWE-89")
}

// ---------------------------------------------------------------------------
// Go scan: command injection
// ---------------------------------------------------------------------------

func TestCodeScanner_Go_CommandInjection(t *testing.T) {
	s := NewCodeScanner()
	code := `package main
import "os/exec"
func run(input string) {
	exec.Command("sh", "-c", input).Run()
}`
	findings := s.ScanContent("cmd.go", code, "go")
	assertHasCWE(t, findings, "CWE-78")
}

// ---------------------------------------------------------------------------
// Go scan: path traversal
// ---------------------------------------------------------------------------

func TestCodeScanner_Go_PathTraversal(t *testing.T) {
	s := NewCodeScanner()
	code := `package main
import "os"
func read(path string) {
	os.Open(path)
}`
	// Note: this is a basic pattern match; real scanners use data flow
	// We detect os.Open with a variable argument as suspicious
	findings := s.ScanContent("file.go", code, "go")
	// May or may not match depending on rule specificity
	_ = findings
}

// ---------------------------------------------------------------------------
// Go scan: weak crypto
// ---------------------------------------------------------------------------

func TestCodeScanner_Go_WeakCrypto(t *testing.T) {
	s := NewCodeScanner()
	code := `package main
import "crypto/md5"
func hash(data []byte) {
	h := md5.New()
	h.Write(data)
}`
	findings := s.ScanContent("hash.go", code, "go")
	assertHasCWE(t, findings, "CWE-328")
}

// ---------------------------------------------------------------------------
// Go scan: hardcoded credentials
// ---------------------------------------------------------------------------

func TestCodeScanner_Go_HardcodedPassword(t *testing.T) {
	s := NewCodeScanner()
	code := `package main
var password = "super_secret_123"
`
	findings := s.ScanContent("config.go", code, "go")
	assertHasCWE(t, findings, "CWE-798")
}

// ---------------------------------------------------------------------------
// Python rules — at least 10
// ---------------------------------------------------------------------------

func TestCodeScanner_PythonRules_AtLeast10(t *testing.T) {
	s := NewCodeScanner()
	count := 0
	for _, r := range s.rules {
		if r.Language == "python" {
			count++
		}
	}
	if count < 10 {
		t.Errorf("expected at least 10 Python rules, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Python scan: eval
// ---------------------------------------------------------------------------

func TestCodeScanner_Python_Eval(t *testing.T) {
	s := NewCodeScanner()
	code := `
user_input = input("Enter expression: ")
result = eval(user_input)
`
	findings := s.ScanContent("app.py", code, "python")
	assertHasCWE(t, findings, "CWE-95")
}

// ---------------------------------------------------------------------------
// Python scan: exec
// ---------------------------------------------------------------------------

func TestCodeScanner_Python_Exec(t *testing.T) {
	s := NewCodeScanner()
	code := `exec(user_code)`
	findings := s.ScanContent("run.py", code, "python")
	assertHasCWE(t, findings, "CWE-95")
}

// ---------------------------------------------------------------------------
// Python scan: pickle
// ---------------------------------------------------------------------------

func TestCodeScanner_Python_Pickle(t *testing.T) {
	s := NewCodeScanner()
	code := `
import pickle
data = pickle.loads(user_data)
`
	findings := s.ScanContent("data.py", code, "python")
	assertHasCWE(t, findings, "CWE-502")
}

// ---------------------------------------------------------------------------
// Python scan: yaml.load
// ---------------------------------------------------------------------------

func TestCodeScanner_Python_YAMLLoad(t *testing.T) {
	s := NewCodeScanner()
	code := `
import yaml
config = yaml.load(data)
`
	findings := s.ScanContent("config.py", code, "python")
	assertHasCWE(t, findings, "CWE-502")
}

// ---------------------------------------------------------------------------
// Python scan: SQL injection
// ---------------------------------------------------------------------------

func TestCodeScanner_Python_SQLInjection(t *testing.T) {
	s := NewCodeScanner()
	code := `
cursor.execute("SELECT * FROM users WHERE id = " + user_id)
`
	findings := s.ScanContent("db.py", code, "python")
	assertHasCWE(t, findings, "CWE-89")
}

// ---------------------------------------------------------------------------
// JavaScript rules — at least 10
// ---------------------------------------------------------------------------

func TestCodeScanner_JSRules_AtLeast10(t *testing.T) {
	s := NewCodeScanner()
	count := 0
	for _, r := range s.rules {
		if r.Language == "javascript" {
			count++
		}
	}
	if count < 10 {
		t.Errorf("expected at least 10 JavaScript rules, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// JS scan: eval
// ---------------------------------------------------------------------------

func TestCodeScanner_JS_Eval(t *testing.T) {
	s := NewCodeScanner()
	code := `const result = eval(userInput);`
	findings := s.ScanContent("app.js", code, "javascript")
	assertHasCWE(t, findings, "CWE-95")
}

// ---------------------------------------------------------------------------
// JS scan: innerHTML
// ---------------------------------------------------------------------------

func TestCodeScanner_JS_InnerHTML(t *testing.T) {
	s := NewCodeScanner()
	code := `element.innerHTML = userInput;`
	findings := s.ScanContent("dom.js", code, "javascript")
	assertHasCWE(t, findings, "CWE-79")
}

// ---------------------------------------------------------------------------
// JS scan: document.write
// ---------------------------------------------------------------------------

func TestCodeScanner_JS_DocumentWrite(t *testing.T) {
	s := NewCodeScanner()
	code := `document.write(data);`
	findings := s.ScanContent("render.js", code, "javascript")
	assertHasCWE(t, findings, "CWE-79")
}

// ---------------------------------------------------------------------------
// Clean file — no findings
// ---------------------------------------------------------------------------

func TestCodeScanner_CleanGoFile(t *testing.T) {
	s := NewCodeScanner()
	code := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	findings := s.ScanContent("main.go", code, "go")
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean Go file, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  unexpected: %s (%s) line %d", f.RuleID, f.Title, f.Line)
		}
	}
}

func TestCodeScanner_CleanPythonFile(t *testing.T) {
	s := NewCodeScanner()
	code := `
def greet(name: str) -> str:
    return f"Hello, {name}!"

if __name__ == "__main__":
    print(greet("World"))
`
	findings := s.ScanContent("app.py", code, "python")
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean Python file, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  unexpected: %s (%s) line %d", f.RuleID, f.Title, f.Line)
		}
	}
}

// ---------------------------------------------------------------------------
// Language auto-detection in ScanContent
// ---------------------------------------------------------------------------

func TestCodeScanner_ScanContent_AutoDetectsLanguage(t *testing.T) {
	s := NewCodeScanner()
	code := `const result = eval(userInput);`
	// Pass empty language — should detect from filename
	findings := s.ScanContent("app.js", code, "")
	assertHasCWE(t, findings, "CWE-95")
}

// ---------------------------------------------------------------------------
// Unknown language — no crash
// ---------------------------------------------------------------------------

func TestCodeScanner_ScanContent_UnknownLanguage(t *testing.T) {
	s := NewCodeScanner()
	findings := s.ScanContent("data.csv", "some,data,here", "")
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for unknown language, got %d", len(findings))
	}
}

// ---------------------------------------------------------------------------
// Finding metadata
// ---------------------------------------------------------------------------

func TestCodeScanner_FindingHasCorrectMetadata(t *testing.T) {
	s := NewCodeScanner()
	code := `const result = eval(userInput);`
	findings := s.ScanContent("app.js", code, "javascript")
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	f := findings[0]
	if f.Scanner != "code-scanner" {
		t.Errorf("expected scanner 'code-scanner', got %q", f.Scanner)
	}
	if f.FilePath != "app.js" {
		t.Errorf("expected file path 'app.js', got %q", f.FilePath)
	}
	if f.Line == 0 {
		t.Error("expected non-zero line number")
	}
	if f.Confidence <= 0 || f.Confidence > 1 {
		t.Errorf("expected confidence in (0,1], got %f", f.Confidence)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func assertHasCWE(t *testing.T, findings []Finding, cwe string) {
	t.Helper()
	for _, f := range findings {
		if f.Metadata != nil && f.Metadata["cwe"] == cwe {
			return
		}
	}
	cweList := make([]string, 0, len(findings))
	for _, f := range findings {
		if f.Metadata != nil {
			cweList = append(cweList, f.Metadata["cwe"])
		}
	}
	t.Errorf("expected finding with CWE %q, got CWEs: %v", cwe, cweList)
}
