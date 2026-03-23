package devsecops

import "context"

// Severity classifies the severity of a finding.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Finding represents a single security issue found by a scanner.
type Finding struct {
	ID          string            `json:"id"`
	Scanner     string            `json:"scanner"`      // which scanner found it
	Severity    Severity          `json:"severity"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	FilePath    string            `json:"file_path,omitempty"`
	Line        int               `json:"line,omitempty"`
	RuleID      string            `json:"rule_id,omitempty"`
	Confidence  float64           `json:"confidence"` // 0.0-1.0
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ScanRequest configures what to scan.
type ScanRequest struct {
	Path       string            `json:"path"`                  // file or directory
	Recursive  bool              `json:"recursive"`
	IncludeExt []string          `json:"include_ext,omitempty"` // e.g., [".go", ".py"]
	ExcludeDirs []string         `json:"exclude_dirs,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
}

// ScanResult holds findings from a scan.
type ScanResult struct {
	Scanner  string    `json:"scanner"`
	Findings []Finding `json:"findings"`
	Errors   []string  `json:"errors,omitempty"`
	Scanned  int       `json:"files_scanned"`
}

// Scanner is the interface that all built-in and external scanners implement.
type Scanner interface {
	// Name returns the scanner identifier.
	Name() string

	// Description returns a short description.
	Description() string

	// Scan runs the scanner against the given request.
	Scan(ctx context.Context, req ScanRequest) (ScanResult, error)
}

// ScannerRegistry manages available scanners.
type ScannerRegistry struct {
	scanners map[string]Scanner
}

// NewScannerRegistry creates an empty registry.
func NewScannerRegistry() *ScannerRegistry {
	return &ScannerRegistry{scanners: make(map[string]Scanner)}
}

// Register adds a scanner to the registry.
func (r *ScannerRegistry) Register(s Scanner) {
	r.scanners[s.Name()] = s
}

// Get returns a scanner by name.
func (r *ScannerRegistry) Get(name string) (Scanner, bool) {
	s, ok := r.scanners[name]
	return s, ok
}

// Names returns all registered scanner names.
func (r *ScannerRegistry) Names() []string {
	names := make([]string, 0, len(r.scanners))
	for name := range r.scanners {
		names = append(names, name)
	}
	return names
}

// ScanAll runs all registered scanners in sequence.
func (r *ScannerRegistry) ScanAll(ctx context.Context, req ScanRequest) ([]ScanResult, error) {
	var results []ScanResult
	for _, s := range r.scanners {
		result, err := s.Scan(ctx, req)
		if err != nil {
			results = append(results, ScanResult{
				Scanner: s.Name(),
				Errors:  []string{err.Error()},
			})
			continue
		}
		results = append(results, result)
	}
	return results, nil
}
