package tui

import (
	"fmt"
	"strings"
)

// DiffLineType classifies a line in a unified diff.
type DiffLineType string

const (
	DiffAdded   DiffLineType = "added"
	DiffRemoved DiffLineType = "removed"
	DiffContext DiffLineType = "context"
)

// DiffLine represents a single line in a parsed diff.
type DiffLine struct {
	Type    DiffLineType
	Content string
	LineNum int
}

// ParseUnifiedDiff parses a unified diff string into structured DiffLines.
func ParseUnifiedDiff(diff string) []DiffLine {
	if diff == "" {
		return nil
	}

	var result []DiffLine
	lineNum := 0

	for _, raw := range strings.Split(diff, "\n") {
		// Skip diff headers
		if strings.HasPrefix(raw, "---") || strings.HasPrefix(raw, "+++") {
			continue
		}
		if strings.HasPrefix(raw, "@@") {
			continue
		}

		if strings.HasPrefix(raw, "+") {
			lineNum++
			result = append(result, DiffLine{
				Type:    DiffAdded,
				Content: raw[1:],
				LineNum: lineNum,
			})
		} else if strings.HasPrefix(raw, "-") {
			lineNum++
			result = append(result, DiffLine{
				Type:    DiffRemoved,
				Content: raw[1:],
				LineNum: lineNum,
			})
		} else if strings.HasPrefix(raw, " ") {
			lineNum++
			result = append(result, DiffLine{
				Type:    DiffContext,
				Content: raw[1:],
				LineNum: lineNum,
			})
		}
	}

	return result
}

// RenderDiff renders parsed diff lines into a displayable string.
func RenderDiff(lines []DiffLine, width int) string {
	if len(lines) == 0 {
		return "  (no changes)"
	}

	var b strings.Builder
	for _, line := range lines {
		marker := " "
		switch line.Type {
		case DiffAdded:
			marker = "+"
		case DiffRemoved:
			marker = "-"
		}

		content := line.Content
		maxContent := width - 8 // margin for line num + marker
		if maxContent > 0 && len(content) > maxContent {
			content = content[:maxContent]
		}

		b.WriteString(fmt.Sprintf(" %s %3d  %s\n", marker, line.LineNum, content))
	}

	return b.String()
}
