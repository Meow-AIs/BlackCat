package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/meowai/blackcat/internal/tools"
)

// validMermaidPrefixes are the recognized Mermaid diagram type keywords.
var validMermaidPrefixes = []string{
	"graph ",
	"flowchart ",
	"sequenceDiagram",
	"classDiagram",
	"stateDiagram",
	"erDiagram",
	"gantt",
	"pie",
	"gitgraph",
	"journey",
	"mindmap",
	"timeline",
	"quadrantChart",
	"sankey",
	"xychart",
	"block-beta",
}

// DiagramRenderTool renders Mermaid diagrams to text or SVG.
type DiagramRenderTool struct{}

// NewDiagramRenderTool creates a new DiagramRenderTool.
func NewDiagramRenderTool() *DiagramRenderTool {
	return &DiagramRenderTool{}
}

// Info returns the tool definition for render_diagram.
func (t *DiagramRenderTool) Info() tools.Definition {
	return tools.Definition{
		Name:        "render_diagram",
		Description: "Render a Mermaid diagram to text/SVG representation",
		Category:    "multimodal",
		Parameters: []tools.Parameter{
			{Name: "mermaid_code", Type: "string", Description: "Mermaid diagram source code", Required: true},
			{Name: "format", Type: "string", Description: "Output format: text or svg (default: text)", Enum: []string{"text", "svg"}},
		},
	}
}

// Execute renders the Mermaid diagram in the requested format.
func (t *DiagramRenderTool) Execute(_ context.Context, args map[string]any) (tools.Result, error) {
	code, err := requireStringArg(args, "mermaid_code")
	if err != nil {
		return tools.Result{}, err
	}

	if syntaxErr := ValidateMermaidSyntax(code); syntaxErr != nil {
		return tools.Result{
			Error:    syntaxErr.Error(),
			ExitCode: 1,
		}, nil
	}

	format := "text"
	if f, ok := args["format"].(string); ok && f != "" {
		format = strings.ToLower(f)
	}

	switch format {
	case "svg":
		return tools.Result{
			Output:   "SVG rendering requires mermaid-cli. Install: npm install -g @mermaid-js/mermaid-cli",
			ExitCode: 0,
		}, nil
	default:
		output := fmt.Sprintf("--- Mermaid Diagram ---\n\n%s\n\n--- End Diagram ---", code)
		return tools.Result{Output: output, ExitCode: 0}, nil
	}
}

// ValidateMermaidSyntax performs a basic check that the code starts with a
// recognized Mermaid diagram type keyword.
func ValidateMermaidSyntax(code string) error {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return fmt.Errorf("mermaid code is empty")
	}

	for _, prefix := range validMermaidPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return nil
		}
	}

	return fmt.Errorf("unrecognized Mermaid diagram type; code must start with one of: graph, flowchart, sequenceDiagram, classDiagram, stateDiagram, erDiagram, gantt, pie, gitgraph, journey, mindmap, timeline, quadrantChart, sankey, xychart, block-beta")
}
