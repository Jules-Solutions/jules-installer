// Package tui — screens.go contains the View() render functions for each installer state.
package tui

import (
	"fmt"
	"strings"

	"github.com/Jules-Solutions/jules-installer/internal/audit"
	"github.com/Jules-Solutions/jules-installer/internal/config"
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

// renderAudit renders the environment audit screen with real check results.
func renderAudit(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  Environment Audit"))
	sb.WriteString("\n")
	if m.width > 4 {
		sb.WriteString(HRule(m.width - 4))
	}
	sb.WriteString("\n\n")

	if len(m.auditResults) == 0 {
		sb.WriteString("  " + SpinnerWithMessage(m.spinnerFrame, "Scanning environment…"))
		return sb.String()
	}

	// Show check results.
	for _, c := range m.auditResults {
		detail := c.Version
		if c.Detail != "" {
			if detail != "" {
				detail += " — " + c.Detail
			} else {
				detail = c.Detail
			}
		}
		sb.WriteString("  " + StatusLine(c.Status, fmt.Sprintf("%-14s", c.Name), detail) + "\n")
	}

	// Summary line.
	pass, fail, warn := 0, 0, 0
	for _, c := range m.auditResults {
		switch c.Status {
		case audit.StatusPass:
			pass++
		case audit.StatusFail:
			fail++
		case audit.StatusWarn:
			warn++
		}
	}
	sb.WriteString("\n")
	summary := fmt.Sprintf("  %d passed", pass)
	if warn > 0 {
		summary += fmt.Sprintf(", %d warnings", warn)
	}
	if fail > 0 {
		summary += fmt.Sprintf(", %d failed", fail)
	}
	sb.WriteString(mutedStyle.Render(summary))
	sb.WriteString("\n\n")

	// Show install results if we just ran installers.
	if len(m.installResults) > 0 {
		sb.WriteString(subtitleStyle.Render("  Install results:"))
		sb.WriteString("\n")
		for _, r := range m.installResults {
			if r.Success {
				sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render(r.Name) + " " + mutedStyle.Render(r.Detail) + "\n")
			} else {
				sb.WriteString("  " + errorStyle.Render("✗") + " " + bodyStyle.Render(r.Name) + " " + mutedStyle.Render(r.Detail) + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// State-dependent prompt.
	switch m.auditSubState {
	case auditOfferInstall:
		installable := audit.CountInstallable(m.auditResults)
		sb.WriteString("  " + highlightStyle.Render(fmt.Sprintf("%d tools can be auto-installed.", installable)))
		sb.WriteString("\n")
		sb.WriteString("  " + bodyStyle.Render("Install them now? ") +
			highlightStyle.Render("y") + mutedStyle.Render("/") + highlightStyle.Render("n"))

	case auditInstalling:
		sb.WriteString("  " + SpinnerWithMessage(m.spinnerFrame, "Installing missing tools… (this may take a minute)"))

	default: // auditShowResults, auditRecheck
		sb.WriteString("  " + mutedStyle.Render("Press ") + highlightStyle.Render("Enter") + mutedStyle.Render(" to continue…"))
	}

	return sb.String()
}

// renderSetup renders the interactive setup questions.
func renderSetup(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  Setup"))
	sb.WriteString("\n")
	if m.width > 4 {
		sb.WriteString(HRule(m.width - 4))
	}
	sb.WriteString("\n\n")

	switch m.setupState {
	case setupVaultPath:
		sb.WriteString("  " + subtitleStyle.Render("Where should we put your vault?"))
		sb.WriteString("\n\n")
		sb.WriteString("  " + mutedStyle.Render("The vault contains your notes, projects, and agent configuration."))
		sb.WriteString("\n  " + mutedStyle.Render("It's a git repo — you can move it later."))
		sb.WriteString("\n\n")
		if m.setupVaultInput != nil {
			sb.WriteString("  " + m.setupVaultInput.View())
		}
		sb.WriteString("\n\n")
		sb.WriteString("  " + mutedStyle.Render("Press ") + highlightStyle.Render("Enter") + mutedStyle.Render(" to confirm"))

	case setupConfirmMCP:
		sb.WriteString("  " + subtitleStyle.Render("Configure Claude Code MCP connection?"))
		sb.WriteString("\n\n")
		sb.WriteString("  " + mutedStyle.Render("This will add jules.solutions as an MCP server when your vault is set up."))
		sb.WriteString("\n  " + mutedStyle.Render("Your API key is read from config — no secrets in the MCP config file."))
		sb.WriteString("\n\n")
		if m.setupConfigMCP {
			sb.WriteString("  " + highlightStyle.Render("> Yes") + "    " + mutedStyle.Render("  No"))
		} else {
			sb.WriteString("  " + mutedStyle.Render("  Yes") + "    " + highlightStyle.Render("> No"))
		}
		sb.WriteString("\n\n")
		sb.WriteString("  " + mutedStyle.Render("← → to choose, ") + highlightStyle.Render("Enter") + mutedStyle.Render(" to confirm"))
	}

	return sb.String()
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

// renderDone renders the completion screen with an honest summary.
func renderDone(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(successStyle.Render("  ✓ Setup Complete"))
	sb.WriteString("\n\n")

	// What was done.
	sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("Authenticated via ") + highlightStyle.Render(string(m.authMethod)))
	sb.WriteString("\n")

	configPath, _ := config.ConfigPath()
	sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("API key saved to ") + mutedStyle.Render(configPath))
	sb.WriteString("\n")

	if len(m.auditResults) > 0 {
		pass := 0
		for _, c := range m.auditResults {
			if c.Status == audit.StatusPass {
				pass++
			}
		}
		sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render(fmt.Sprintf("Environment audited (%d/%d checks passed)", pass, len(m.auditResults))))
		sb.WriteString("\n")
	}

	sb.WriteString("  " + mutedStyle.Render("⟳") + " " + bodyStyle.Render("Vault download — coming in next release"))
	sb.WriteString("\n\n")

	// Next steps.
	sb.WriteString(subtitleStyle.Render("  Next steps:"))
	sb.WriteString("\n")
	sb.WriteString("  " + bodyStyle.Render("  1. Your API key is saved — CLI tools will pick it up automatically"))
	sb.WriteString("\n")
	sb.WriteString("  " + bodyStyle.Render("  2. Install missing tools shown in the audit (if any)"))
	sb.WriteString("\n")
	sb.WriteString("  " + bodyStyle.Render("  3. Vault download will be available in the next release"))
	sb.WriteString("\n\n")

	sb.WriteString("  " + mutedStyle.Render("Press ") + highlightStyle.Render("Enter") + mutedStyle.Render(" or ") +
		highlightStyle.Render("q") + mutedStyle.Render(" to exit."))
	sb.WriteString("\n")

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
