package architect

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// C4 Context Diagram
// ---------------------------------------------------------------------------

func TestRenderC4Mermaid_ContextDiagram(t *testing.T) {
	d := C4Diagram{
		Title: "System Context",
		Type:  DiagC4Context,
		Elements: []C4Element{
			{ID: "user", Label: "User", Type: "person", Description: "End user"},
			{ID: "sys", Label: "BlackCat", Type: "system", Description: "AI Agent CLI"},
			{ID: "llm", Label: "LLM Provider", Type: "system", Description: "Claude/GPT", External: true},
		},
		Relationships: []C4Relationship{
			{From: "user", To: "sys", Label: "Uses"},
			{From: "sys", To: "llm", Label: "Sends prompts", Technology: "HTTPS"},
		},
	}

	output := RenderC4Mermaid(d)
	if !strings.HasPrefix(output, "C4Context") {
		t.Error("expected C4Context prefix")
	}
	if !strings.Contains(output, "title System Context") {
		t.Error("expected title")
	}
	if !strings.Contains(output, "Person(user") {
		t.Error("expected Person element for user")
	}
	if !strings.Contains(output, "System(sys") {
		t.Error("expected System element for sys")
	}
	if !strings.Contains(output, "System_Ext(llm") {
		t.Error("expected System_Ext for external llm")
	}
	if !strings.Contains(output, "Rel(sys, llm") {
		t.Error("expected relationship")
	}
}

// ---------------------------------------------------------------------------
// C4 Container Diagram
// ---------------------------------------------------------------------------

func TestRenderC4Mermaid_ContainerDiagram(t *testing.T) {
	d := C4Diagram{
		Title: "Container View",
		Type:  DiagC4Container,
		Elements: []C4Element{
			{ID: "api", Label: "API Server", Type: "container", Technology: "Go"},
			{ID: "db", Label: "Database", Type: "database", Technology: "SQLite"},
			{ID: "queue", Label: "Task Queue", Type: "queue", Technology: "In-memory"},
		},
	}

	output := RenderC4Mermaid(d)
	if !strings.HasPrefix(output, "C4Container") {
		t.Error("expected C4Container prefix")
	}
	if !strings.Contains(output, "Container(api") {
		t.Error("expected Container for api")
	}
	if !strings.Contains(output, "ContainerDb(db") {
		t.Error("expected ContainerDb for database")
	}
	if !strings.Contains(output, "ContainerQueue(queue") {
		t.Error("expected ContainerQueue for queue")
	}
}

// ---------------------------------------------------------------------------
// External Person
// ---------------------------------------------------------------------------

func TestRenderC4Mermaid_ExternalPerson(t *testing.T) {
	d := C4Diagram{
		Type: DiagC4Context,
		Elements: []C4Element{
			{ID: "ext", Label: "External User", Type: "person", External: true},
		},
	}
	output := RenderC4Mermaid(d)
	if !strings.Contains(output, "Person_Ext(ext") {
		t.Error("expected Person_Ext for external person")
	}
}

// ---------------------------------------------------------------------------
// Sequence Diagram
// ---------------------------------------------------------------------------

func TestRenderSequenceMermaid(t *testing.T) {
	d := SequenceDiagram{
		Title: "Auth Flow",
		Participants: []SequenceParticipant{
			{ID: "U", Label: "User", Type: "actor"},
			{ID: "API", Label: "API Server", Type: "participant"},
			{ID: "DB", Label: "Database", Type: "participant"},
		},
		Messages: []SequenceMessage{
			{From: "U", To: "API", Label: "Login", Type: "sync"},
			{From: "API", To: "DB", Label: "Query user", Type: "sync"},
			{From: "DB", To: "API", Label: "User data", Type: "reply"},
			{From: "API", To: "U", Label: "JWT token", Type: "reply"},
		},
	}

	output := RenderSequenceMermaid(d)
	if !strings.HasPrefix(output, "sequenceDiagram") {
		t.Error("expected sequenceDiagram prefix")
	}
	if !strings.Contains(output, "actor U as User") {
		t.Error("expected actor for user")
	}
	if !strings.Contains(output, "participant API as API Server") {
		t.Error("expected participant for API")
	}
	if !strings.Contains(output, "U->>API: Login") {
		t.Error("expected sync message")
	}
	if !strings.Contains(output, "DB-->>API: User data") {
		t.Error("expected reply message")
	}
}

func TestRenderSequenceMermaid_AsyncMessage(t *testing.T) {
	d := SequenceDiagram{
		Participants: []SequenceParticipant{
			{ID: "A", Label: "Service A"},
			{ID: "Q", Label: "Queue"},
		},
		Messages: []SequenceMessage{
			{From: "A", To: "Q", Label: "Publish event", Type: "async"},
		},
	}

	output := RenderSequenceMermaid(d)
	if !strings.Contains(output, "A--))Q: Publish event") {
		t.Errorf("expected async arrow, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// Flowchart
// ---------------------------------------------------------------------------

func TestRenderFlowchartMermaid(t *testing.T) {
	steps := []string{"Start", "Process", "Validate", "Complete"}
	output := RenderFlowchartMermaid("My Flow", steps)

	if !strings.HasPrefix(output, "flowchart TD") {
		t.Error("expected flowchart TD prefix")
	}
	if !strings.Contains(output, "S0") {
		t.Error("expected S0 node")
	}
	if !strings.Contains(output, "S0 --> S1") {
		t.Error("expected S0 --> S1 edge")
	}
	if !strings.Contains(output, "S2 --> S3") {
		t.Error("expected S2 --> S3 edge")
	}
}

func TestRenderFlowchartMermaid_Empty(t *testing.T) {
	output := RenderFlowchartMermaid("Empty", nil)
	if !strings.HasPrefix(output, "flowchart TD") {
		t.Error("expected flowchart TD even for empty")
	}
}

func TestRenderFlowchartMermaid_SingleStep(t *testing.T) {
	output := RenderFlowchartMermaid("Single", []string{"Only Step"})
	if !strings.Contains(output, "S0") {
		t.Error("expected S0")
	}
	if strings.Contains(output, "-->") {
		t.Error("single step should have no edges")
	}
}

// ---------------------------------------------------------------------------
// Technology in relationship
// ---------------------------------------------------------------------------

func TestRenderC4Mermaid_RelWithTechnology(t *testing.T) {
	d := C4Diagram{
		Type: DiagC4Context,
		Relationships: []C4Relationship{
			{From: "a", To: "b", Label: "calls", Technology: "gRPC"},
		},
	}
	output := RenderC4Mermaid(d)
	if !strings.Contains(output, `"gRPC"`) {
		t.Error("expected technology in relationship")
	}
}

func TestRenderC4Mermaid_RelWithoutTechnology(t *testing.T) {
	d := C4Diagram{
		Type: DiagC4Context,
		Relationships: []C4Relationship{
			{From: "a", To: "b", Label: "calls"},
		},
	}
	output := RenderC4Mermaid(d)
	// Should NOT have a trailing comma or empty tech
	if strings.Contains(output, ", \"\"") {
		t.Error("should not have empty technology string")
	}
}
