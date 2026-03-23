package architect

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ADRStatus represents the lifecycle state of an Architecture Decision Record.
type ADRStatus string

const (
	ADRProposed   ADRStatus = "proposed"
	ADRAccepted   ADRStatus = "accepted"
	ADRDeprecated ADRStatus = "deprecated"
	ADRSuperseded ADRStatus = "superseded"
)

// ADROption represents one considered option in an ADR.
type ADROption struct {
	Name   string
	Pros   []string
	Cons   []string
	Chosen bool
}

// ADR is an Architecture Decision Record following the MADR template.
type ADR struct {
	Number       int
	Title        string
	Status       ADRStatus
	Context      string
	Decision     string
	Consequences []string
	Date         string
	Supersedes   int
	Options      []ADROption
}

// NewADR creates a new ADR with proposed status and today's date.
func NewADR(number int, title string) *ADR {
	return &ADR{
		Number: number,
		Title:  title,
		Status: ADRProposed,
		Date:   time.Now().Format("2006-01-02"),
	}
}

// AddOption appends a considered option to the ADR.
func (a *ADR) AddOption(name string, pros, cons []string) {
	p := make([]string, len(pros))
	copy(p, pros)
	c := make([]string, len(cons))
	copy(c, cons)
	a.Options = append(a.Options, ADROption{
		Name: name,
		Pros: p,
		Cons: c,
	})
}

// Choose marks an option as chosen, clears previous selections, and sets status to Accepted.
func (a *ADR) Choose(optionName string) error {
	found := false
	newOptions := make([]ADROption, len(a.Options))
	for i, opt := range a.Options {
		chosen := opt.Name == optionName
		if chosen {
			found = true
		}
		newOptions[i] = ADROption{
			Name:   opt.Name,
			Pros:   opt.Pros,
			Cons:   opt.Cons,
			Chosen: chosen,
		}
	}
	if !found {
		return fmt.Errorf("option %q not found", optionName)
	}
	a.Options = newOptions
	a.Status = ADRAccepted
	return nil
}

// FormatMarkdown renders the ADR in MADR format.
func (a *ADR) FormatMarkdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# ADR-%04d: %s\n\n", a.Number, a.Title))
	b.WriteString(fmt.Sprintf("**Date:** %s\n\n", a.Date))

	b.WriteString("## Status\n\n")
	b.WriteString(string(a.Status))
	if a.Supersedes > 0 {
		b.WriteString(fmt.Sprintf("\n\nSupersedes: ADR-%04d", a.Supersedes))
	}
	b.WriteString("\n\n")

	b.WriteString("## Context\n\n")
	b.WriteString(a.Context)
	b.WriteString("\n\n")

	b.WriteString("## Decision\n\n")
	b.WriteString(a.Decision)
	b.WriteString("\n\n")

	if len(a.Options) > 0 {
		b.WriteString("## Options\n\n")
		for _, opt := range a.Options {
			label := opt.Name
			if opt.Chosen {
				label += " (chosen)"
			}
			b.WriteString(fmt.Sprintf("### %s\n\n", label))
			if len(opt.Pros) > 0 {
				b.WriteString("**Pros:**\n")
				for _, p := range opt.Pros {
					b.WriteString(fmt.Sprintf("- %s\n", p))
				}
				b.WriteString("\n")
			}
			if len(opt.Cons) > 0 {
				b.WriteString("**Cons:**\n")
				for _, c := range opt.Cons {
					b.WriteString(fmt.Sprintf("- %s\n", c))
				}
				b.WriteString("\n")
			}
		}
	}

	if len(a.Consequences) > 0 {
		b.WriteString("## Consequences\n\n")
		for _, c := range a.Consequences {
			b.WriteString(fmt.Sprintf("- %s\n", c))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ParseADRFromMarkdown parses an ADR from its MADR markdown representation.
func ParseADRFromMarkdown(md string) (*ADR, error) {
	titleRe := regexp.MustCompile(`^# ADR-(\d+):\s*(.+)$`)
	lines := strings.Split(md, "\n")

	var titleMatch []string
	for _, line := range lines {
		titleMatch = titleRe.FindStringSubmatch(strings.TrimSpace(line))
		if titleMatch != nil {
			break
		}
	}
	if titleMatch == nil {
		return nil, fmt.Errorf("could not find ADR title header")
	}

	num, _ := strconv.Atoi(titleMatch[1])
	adr := &ADR{
		Number: num,
		Title:  strings.TrimSpace(titleMatch[2]),
	}

	sections := parseSections(md)

	if v, ok := sections["Status"]; ok {
		trimmed := strings.TrimSpace(v)
		statusLine := strings.Split(trimmed, "\n")[0]
		adr.Status = ADRStatus(strings.TrimSpace(statusLine))
	}
	if v, ok := sections["Context"]; ok {
		adr.Context = strings.TrimSpace(v)
	}
	if v, ok := sections["Decision"]; ok {
		adr.Decision = strings.TrimSpace(v)
	}
	if v, ok := sections["Consequences"]; ok {
		adr.Consequences = parseListItems(v)
	}

	// Parse date
	dateRe := regexp.MustCompile(`\*\*Date:\*\*\s*(.+)`)
	if m := dateRe.FindStringSubmatch(md); m != nil {
		adr.Date = strings.TrimSpace(m[1])
	}

	// Parse supersedes
	superRe := regexp.MustCompile(`Supersedes:\s*ADR-(\d+)`)
	if m := superRe.FindStringSubmatch(md); m != nil {
		adr.Supersedes, _ = strconv.Atoi(m[1])
	}

	return adr, nil
}

// parseSections splits markdown into h2 sections.
func parseSections(md string) map[string]string {
	result := make(map[string]string)
	sectionRe := regexp.MustCompile(`(?m)^## (.+)$`)
	matches := sectionRe.FindAllStringSubmatchIndex(md, -1)

	for i, m := range matches {
		name := strings.TrimSpace(md[m[2]:m[3]])
		start := m[1]
		end := len(md)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		result[name] = md[start:end]
	}
	return result
}

// parseListItems extracts "- item" lines from markdown text.
func parseListItems(text string) []string {
	var items []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			items = append(items, strings.TrimPrefix(trimmed, "- "))
		}
	}
	return items
}
