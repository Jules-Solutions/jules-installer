// Package tui — screens.go contains the View() render functions for each installer state.
package tui

import (
	"fmt"
	"strings"
)

// renderWelcome renders the welcome / splash screen.
func renderWelcome(m Model) string {
	var sb strings.Builder

	// Vertical padding to centre the content roughly.
	topPad := (m.height - 18) / 2
	if topPad < 0 {
		topPad = 0
	}
	sb.WriteString(strings.Repeat("\n", topPad))

	// ASCII banner box.
	banner := titleStyle.Render(`
    ╔══════════════════════════════════════╗
    ║       Jules.Solutions Setup          ║
    ║                                      ║
    ║  One file. Everything configured.    ║
    ╚══════════════════════════════════════╝`)
	sb.WriteString(banner)
	sb.WriteString("\n\n")

	// Tag line.
	tagline := subtitleStyle.Render("    Authenticates you · Audits your environment · Gets you running")
	sb.WriteString(tagline)
	sb.WriteString("\n\n")

	// Version hint.
	if m.version != "" {
		sb.WriteString(mutedStyle.Render("    " + m.version))
		sb.WriteString("\n\n")
	}

	sb.WriteString(bodyStyle.Render("    Press ") + highlightStyle.Render("Enter") + bodyStyle.Render(" to begin…"))
	sb.WriteString("\n")

	return sb.String()
}

// renderAuth renders the authentication flow screen.
func renderAuth(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  Authentication"))
	sb.WriteString("\n")
	sb.WriteString(HRule(m.width - 4))
	sb.WriteString("\n\n")

	switch m.authState {
	case authStateBrowser:
		sb.WriteString("  " + SpinnerWithMessage(m.spinnerFrame, "Opening browser…"))
		sb.WriteString("\n\n")
		sb.WriteString("  " + mutedStyle.Render("Waiting for you to log in at:"))
		sb.WriteString("\n  " + highlightStyle.Render("  https://auth.jules.solutions"))
		sb.WriteString("\n\n")
		sb.WriteString("  " + mutedStyle.Render("If the browser didn't open, press ") +
			highlightStyle.Render("d") + mutedStyle.Render(" for device code or ") +
			highlightStyle.Render("k") + mutedStyle.Render(" to paste a key."))

	case authStateDevice:
		sb.WriteString("  " + subtitleStyle.Render("Device Code"))
		sb.WriteString("\n\n")
		if m.deviceCode != "" {
			sb.WriteString("  " + bodyStyle.Render("Go to:"))
			sb.WriteString("\n  " + codeStyle.Render("  "+m.deviceVerifyURI+"  "))
			sb.WriteString("\n\n")
			sb.WriteString("  " + bodyStyle.Render("Enter code:"))
			sb.WriteString("\n  " + codeStyle.Render("  "+m.deviceCode+"  "))
			sb.WriteString("\n\n")
			sb.WriteString("  " + SpinnerWithMessage(m.spinnerFrame, "Waiting for confirmation…"))
		} else {
			sb.WriteString("  " + SpinnerWithMessage(m.spinnerFrame, "Requesting device code…"))
		}

	case authStateAPIKey:
		sb.WriteString("  " + subtitleStyle.Render("API Key"))
		sb.WriteString("\n\n")
		sb.WriteString("  " + bodyStyle.Render("Paste your Jules.Solutions API key below."))
		sb.WriteString("\n  " + mutedStyle.Render("Keys start with dck_  ·  Found at jules.solutions/settings/api-keys"))
		sb.WriteString("\n\n")
		if m.textInput != nil {
			sb.WriteString("  " + m.textInput.View())
		}
		sb.WriteString("\n\n")
		if m.authErrMsg != "" {
			sb.WriteString("  " + errorStyle.Render(m.authErrMsg))
			sb.WriteString("\n")
		}
		sb.WriteString("  " + mutedStyle.Render("Press Enter to validate · Esc to go back"))

	case authStateSuccess:
		sb.WriteString("  " + successStyle.Render("✓ Authenticated successfully"))
		sb.WriteString("\n\n")
		sb.WriteString("  " + KeyValueRow("Method", string(m.authMethod)))
		sb.WriteString("\n")
		sb.WriteString("  " + KeyValueRow("Key", truncateKey(m.apiKey)))
		sb.WriteString("\n\n")
		sb.WriteString("  " + mutedStyle.Render("Press Enter to continue…"))
	}

	return sb.String()
}

// renderAudit renders the environment audit screen (stub).
func renderAudit(_ Model) string {
	return "\n" + titleStyle.Render("  Environment Audit") + "\n\n" +
		mutedStyle.Render("  Coming soon — environment checks will appear here.") + "\n"
}

// renderSetup renders the setup questions screen (stub).
func renderSetup(_ Model) string {
	return "\n" + titleStyle.Render("  Setup") + "\n\n" +
		mutedStyle.Render("  Coming soon — vault path and preferences will appear here.") + "\n"
}

// renderDownload renders the vault download progress screen (stub).
func renderDownload(_ Model) string {
	return "\n" + titleStyle.Render("  Downloading Vault") + "\n\n" +
		mutedStyle.Render("  Coming soon — download progress will appear here.") + "\n"
}

// renderConfig renders the config writing screen (stub).
func renderConfig(_ Model) string {
	return "\n" + titleStyle.Render("  Writing Configuration") + "\n\n" +
		mutedStyle.Render("  Coming soon — config writing will appear here.") + "\n"
}

// renderDone renders the completion / handoff screen.
func renderDone(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(successStyle.Render("  ✓ Setup Complete!"))
	sb.WriteString("\n\n")
	sb.WriteString(bodyStyle.Render("  Jules.Solutions is ready. Claude Code has been configured."))
	sb.WriteString("\n\n")
	sb.WriteString(mutedStyle.Render("  Press ") + highlightStyle.Render("Enter") + mutedStyle.Render(" or ") +
		highlightStyle.Render("q") + mutedStyle.Render(" to exit."))
	sb.WriteString("\n")
	_ = m
	return sb.String()
}

// renderError renders the error screen.
func renderError(m Model) string {
	return fmt.Sprintf("\n%s\n\n  %s\n\n  %s\n",
		errorStyle.Render("  Something went wrong"),
		bodyStyle.Render(m.err.Error()),
		mutedStyle.Render("Press q to exit."),
	)
}

// truncateKey masks all but the last 6 characters of an API key for display.
func truncateKey(key string) string {
	if len(key) <= 6 {
		return strings.Repeat("*", len(key))
	}
	return strings.Repeat("*", len(key)-6) + key[len(key)-6:]
}
