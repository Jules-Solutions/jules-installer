// Package tui — styles.go defines all Lipgloss styles using the Jules.Solutions brand.
package tui

import "github.com/charmbracelet/lipgloss"

// Brand palette — Jules.Solutions
const (
	colorPrimary  = lipgloss.Color("#6366F1") // indigo
	colorSurface  = lipgloss.Color("#0F0F23") // near-black
	colorSuccess  = lipgloss.Color("#22C55E") // green
	colorWarning  = lipgloss.Color("#F59E0B") // amber
	colorError    = lipgloss.Color("#EF4444") // red
	colorText     = lipgloss.Color("#E2E8F0") // light grey
	colorMuted    = lipgloss.Color("#64748B") // slate
	colorHighlight = lipgloss.Color("#818CF8") // lighter indigo
)

var (
	// titleStyle renders the main installer title in primary brand color.
	titleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// subtitleStyle renders secondary headings.
	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorHighlight)

	// successStyle renders success messages (check marks, completion messages).
	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	// warningStyle renders warnings.
	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	// errorStyle renders error messages.
	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	// mutedStyle renders de-emphasised text (hints, secondary info).
	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// highlightStyle renders inline emphasis within body text.
	highlightStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true)

	// boxStyle wraps content in a rounded border in the surface colour.
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	// bodyStyle is the default text style for body copy.
	bodyStyle = lipgloss.NewStyle().
			Foreground(colorText)

	// codeStyle highlights inline code or user codes.
	codeStyle = lipgloss.NewStyle().
			Foreground(colorSurface).
			Background(colorHighlight).
			Padding(0, 1)
)
