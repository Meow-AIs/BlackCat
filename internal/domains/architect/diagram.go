package architect

import (
	"fmt"
	"strings"
)

// DiagramType identifies the kind of diagram to generate.
type DiagramType string

const (
	DiagC4Context   DiagramType = "c4_context"
	DiagC4Container DiagramType = "c4_container"
	DiagSequence    DiagramType = "sequence"
	DiagFlowchart   DiagramType = "flowchart"
)

// C4Element represents a system, container, or person in a C4 diagram.
type C4Element struct {
	ID          string
	Label       string
	Type        string // "person", "system", "container", "database", "queue"
	Technology  string
	Description string
	External    bool
}

// C4Relationship represents a connection between two C4 elements.
type C4Relationship struct {
	From       string // element ID
	To         string // element ID
	Label      string
	Technology string
}

// C4Diagram holds the data for a C4 diagram.
type C4Diagram struct {
	Title         string
	Type          DiagramType
	Elements      []C4Element
	Relationships []C4Relationship
}

// SequenceParticipant represents an actor or system in a sequence diagram.
type SequenceParticipant struct {
	ID    string
	Label string
	Type  string // "actor", "participant"
}

// SequenceMessage represents a message between participants.
type SequenceMessage struct {
	From  string
	To    string
	Label string
	Type  string // "sync", "async", "reply"
}

// SequenceDiagram holds the data for a sequence diagram.
type SequenceDiagram struct {
	Title        string
	Participants []SequenceParticipant
	Messages     []SequenceMessage
}

// RenderC4Mermaid generates Mermaid syntax for a C4 context or container diagram.
func RenderC4Mermaid(d C4Diagram) string {
	var b strings.Builder

	switch d.Type {
	case DiagC4Context:
		b.WriteString("C4Context\n")
	case DiagC4Container:
		b.WriteString("C4Container\n")
	default:
		b.WriteString("C4Context\n")
	}

	if d.Title != "" {
		b.WriteString(fmt.Sprintf("    title %s\n", d.Title))
	}
	b.WriteByte('\n')

	for _, e := range d.Elements {
		line := renderC4Element(e)
		b.WriteString(fmt.Sprintf("    %s\n", line))
	}
	b.WriteByte('\n')

	for _, r := range d.Relationships {
		tech := ""
		if r.Technology != "" {
			tech = fmt.Sprintf(", %q", r.Technology)
		}
		b.WriteString(fmt.Sprintf("    Rel(%s, %s, %q%s)\n", r.From, r.To, r.Label, tech))
	}

	return b.String()
}

func renderC4Element(e C4Element) string {
	desc := e.Description
	if desc == "" {
		desc = e.Label
	}

	switch e.Type {
	case "person":
		if e.External {
			return fmt.Sprintf("Person_Ext(%s, %q, %q)", e.ID, e.Label, desc)
		}
		return fmt.Sprintf("Person(%s, %q, %q)", e.ID, e.Label, desc)
	case "system":
		if e.External {
			return fmt.Sprintf("System_Ext(%s, %q, %q)", e.ID, e.Label, desc)
		}
		return fmt.Sprintf("System(%s, %q, %q)", e.ID, e.Label, desc)
	case "container":
		tech := ""
		if e.Technology != "" {
			tech = fmt.Sprintf(", %q", e.Technology)
		}
		return fmt.Sprintf("Container(%s, %q, %q%s)", e.ID, e.Label, desc, tech)
	case "database":
		tech := ""
		if e.Technology != "" {
			tech = fmt.Sprintf(", %q", e.Technology)
		}
		return fmt.Sprintf("ContainerDb(%s, %q, %q%s)", e.ID, e.Label, desc, tech)
	case "queue":
		tech := ""
		if e.Technology != "" {
			tech = fmt.Sprintf(", %q", e.Technology)
		}
		return fmt.Sprintf("ContainerQueue(%s, %q, %q%s)", e.ID, e.Label, desc, tech)
	default:
		return fmt.Sprintf("System(%s, %q, %q)", e.ID, e.Label, desc)
	}
}

// RenderSequenceMermaid generates Mermaid syntax for a sequence diagram.
func RenderSequenceMermaid(d SequenceDiagram) string {
	var b strings.Builder

	b.WriteString("sequenceDiagram\n")
	if d.Title != "" {
		b.WriteString(fmt.Sprintf("    title %s\n", d.Title))
	}

	for _, p := range d.Participants {
		switch p.Type {
		case "actor":
			b.WriteString(fmt.Sprintf("    actor %s as %s\n", p.ID, p.Label))
		default:
			b.WriteString(fmt.Sprintf("    participant %s as %s\n", p.ID, p.Label))
		}
	}
	b.WriteByte('\n')

	for _, m := range d.Messages {
		arrow := "->>"
		switch m.Type {
		case "async":
			arrow = "--))"
		case "reply":
			arrow = "-->>"
		}
		b.WriteString(fmt.Sprintf("    %s%s%s: %s\n", m.From, arrow, m.To, m.Label))
	}

	return b.String()
}

// RenderFlowchartMermaid generates a simple Mermaid flowchart from steps.
func RenderFlowchartMermaid(title string, steps []string) string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")

	for i, step := range steps {
		id := fmt.Sprintf("S%d", i)
		b.WriteString(fmt.Sprintf("    %s[%q]\n", id, step))
		if i > 0 {
			prev := fmt.Sprintf("S%d", i-1)
			b.WriteString(fmt.Sprintf("    %s --> %s\n", prev, id))
		}
	}

	return b.String()
}
