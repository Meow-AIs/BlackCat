package tui

// Theme defines the color scheme for the TUI.
type Theme struct {
	Primary    string // main accent color
	Secondary  string
	Success    string
	Error      string
	Warning    string
	Muted      string
	Background string
}

// DefaultTheme returns a neutral light theme.
func DefaultTheme() Theme {
	return Theme{
		Primary:    "#7C3AED",
		Secondary:  "#06B6D4",
		Success:    "#10B981",
		Error:      "#EF4444",
		Warning:    "#F59E0B",
		Muted:      "#6B7280",
		Background: "#1F2937",
	}
}

// CatTheme returns a dark theme with cat-inspired purple/orange accents.
func CatTheme() Theme {
	return Theme{
		Primary:    "#A855F7", // purple - cat eyes
		Secondary:  "#FB923C", // orange - cat fur
		Success:    "#34D399",
		Error:      "#F87171",
		Warning:    "#FBBF24",
		Muted:      "#9CA3AF",
		Background: "#111827",
	}
}
