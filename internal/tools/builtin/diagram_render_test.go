package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestDiagramRenderToolInfo(t *testing.T) {
	tool := NewDiagramRenderTool()
	info := tool.Info()

	if info.Name != "render_diagram" {
		t.Errorf("expected name %q, got %q", "render_diagram", info.Name)
	}
	if info.Category != "multimodal" {
		t.Errorf("expected category %q, got %q", "multimodal", info.Category)
	}

	hasMermaidCode := false
	for _, p := range info.Parameters {
		if p.Name == "mermaid_code" && p.Required {
			hasMermaidCode = true
		}
	}
	if !hasMermaidCode {
		t.Error("expected required 'mermaid_code' parameter")
	}
}

func TestDiagramRenderToolExecuteMissingCode(t *testing.T) {
	tool := NewDiagramRenderTool()
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing mermaid_code")
	}
}

func TestValidateMermaidSyntax(t *testing.T) {
	valid := []string{
		"graph TD\n  A-->B",
		"graph LR\n  A-->B",
		"sequenceDiagram\n  Alice->>Bob: Hello",
		"classDiagram\n  class Animal",
		"stateDiagram-v2\n  [*] --> Active",
		"erDiagram\n  CUSTOMER ||--o{ ORDER : places",
		"flowchart TD\n  A-->B",
		"gantt\n  title A Gantt Diagram",
		"pie\n  title Pie Chart",
		"gitgraph\n  commit",
	}
	for _, code := range valid {
		if err := ValidateMermaidSyntax(code); err != nil {
			t.Errorf("expected valid for %q, got error: %v", code[:20], err)
		}
	}

	invalid := []string{
		"",
		"   ",
		"hello world",
		"SELECT * FROM table",
	}
	for _, code := range invalid {
		if err := ValidateMermaidSyntax(code); err == nil {
			t.Errorf("expected invalid for %q", code)
		}
	}
}

func TestDiagramRenderToolExecuteTextFormat(t *testing.T) {
	tool := NewDiagramRenderTool()
	code := "graph TD\n  A-->B\n  B-->C"
	result, err := tool.Execute(context.Background(), map[string]any{
		"mermaid_code": code,
		"format":       "text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "graph TD") {
		t.Errorf("expected output to contain mermaid code, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "Mermaid Diagram") {
		t.Errorf("expected output to contain header, got %q", result.Output)
	}
}

func TestDiagramRenderToolExecuteDefaultFormat(t *testing.T) {
	tool := NewDiagramRenderTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"mermaid_code": "graph TD\n  A-->B",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "graph TD") {
		t.Errorf("expected text format by default, got %q", result.Output)
	}
}

func TestDiagramRenderToolExecuteSVGFormat(t *testing.T) {
	tool := NewDiagramRenderTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"mermaid_code": "graph TD\n  A-->B",
		"format":       "svg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "mermaid-cli") {
		t.Errorf("expected SVG placeholder message, got %q", result.Output)
	}
}

func TestDiagramRenderToolExecuteInvalidSyntax(t *testing.T) {
	tool := NewDiagramRenderTool()
	result, err := tool.Execute(context.Background(), map[string]any{
		"mermaid_code": "this is not mermaid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}
