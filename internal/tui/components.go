// Package tui — components.go defines reusable TUI components.
package tui

import (
	"fmt"
	"strings"
)

// spinnerFrames is the sequence of characters used for the animated spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// SpinnerWithMessage renders an animated spinner followed by a message.
// frame should be incremented by the caller each tick (wraps automatically).
func SpinnerWithMessage(frame int, message string) string {
	f := spinnerFrames[frame%len(spinnerFrames)]
	return titleStyle.Render(f) + " " + bodyStyle.Render(message)
}

// StatusLine renders a single status line with a leading icon.
//
// status must be one of: "pass", "fail", "warn", "skip", "pending".
func StatusLine(status, label, detail string) string {
	var icon string

	switch status {
	case "pass":
		icon = successStyle.Render("✓")
	case "fail":
		icon = errorStyle.Render("✗")
	case "warn":
		icon = warningStyle.Render("!")
	case "skip":
		icon = mutedStyle.Render("–")
	default: // pending / unknown
		icon = mutedStyle.Render("⟳")
	}

	line := icon + " " + bodyStyle.Render(label)
	if detail != "" {
		line += " " + mutedStyle.Render("("+detail+")")
	}
	return line
}

// ChecklistItem renders a single checklist entry with a checkbox.
func ChecklistItem(checked bool, label string) string {
	if checked {
		return successStyle.Render("[✓]") + " " + bodyStyle.Render(label)
	}
	return mutedStyle.Render("[ ]") + " " + bodyStyle.Render(label)
}

// KeyValueRow renders a label: value pair aligned for display.
func KeyValueRow(key, value string) string {
	return mutedStyle.Render(fmt.Sprintf("%-18s", key+":")) + " " + highlightStyle.Render(value)
}

// HRule returns a horizontal divider line of the given width.
func HRule(width int) string {
	return mutedStyle.Render(strings.Repeat("─", width))
}

// Paragraph wraps text at width and returns it styled as body text.
func Paragraph(text string) string {
	return bodyStyle.Render(text)
}
