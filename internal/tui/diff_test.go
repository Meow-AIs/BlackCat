package tui

import (
	"strings"
	"testing"
)

func TestParseUnifiedDiff(t *testing.T) {
	diff := `--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
-import "fmt"
+import (
+	"fmt"
+)
 func main() {}`

	lines := ParseUnifiedDiff(diff)
	if len(lines) == 0 {
		t.Fatal("expected parsed diff lines")
	}

	// Check we have context, added, and removed lines
	var hasAdded, hasRemoved, hasContext bool
	for _, l := range lines {
		switch l.Type {
		case DiffAdded:
			hasAdded = true
		case DiffRemoved:
			hasRemoved = true
		case DiffContext:
			hasContext = true
		}
	}
	if !hasAdded {
		t.Error("expected added lines in diff")
	}
	if !hasRemoved {
		t.Error("expected removed lines in diff")
	}
	if !hasContext {
		t.Error("expected context lines in diff")
	}
}

func TestParseUnifiedDiffEmpty(t *testing.T) {
	lines := ParseUnifiedDiff("")
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty diff, got %d", len(lines))
	}
}

func TestParseUnifiedDiffOnlyAdded(t *testing.T) {
	diff := `--- /dev/null
+++ b/new_file.go
@@ -0,0 +1,3 @@
+package new
+
+func New() {}`

	lines := ParseUnifiedDiff(diff)
	for _, l := range lines {
		if l.Type == DiffRemoved {
			t.Error("expected no removed lines in new file diff")
		}
	}
	if len(lines) == 0 {
		t.Fatal("expected parsed lines")
	}
}

func TestRenderDiff(t *testing.T) {
	lines := []DiffLine{
		{Type: DiffContext, Content: "package main", LineNum: 1},
		{Type: DiffRemoved, Content: "old line", LineNum: 2},
		{Type: DiffAdded, Content: "new line", LineNum: 2},
		{Type: DiffContext, Content: "func main() {}", LineNum: 3},
	}

	output := RenderDiff(lines, 80)
	if !strings.Contains(output, "old line") {
		t.Error("expected render to contain removed line")
	}
	if !strings.Contains(output, "new line") {
		t.Error("expected render to contain added line")
	}
	if !strings.Contains(output, "-") {
		t.Error("expected '-' marker for removed lines")
	}
	if !strings.Contains(output, "+") {
		t.Error("expected '+' marker for added lines")
	}
}

func TestRenderDiffEmpty(t *testing.T) {
	output := RenderDiff(nil, 80)
	if output == "" {
		t.Error("expected non-empty render for empty diff (should show 'no changes')")
	}
}
