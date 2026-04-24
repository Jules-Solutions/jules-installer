// Package tui — screens.go contains the View() render functions for each installer state.
package tui

import (
	"fmt"
	"strings"

	"github.com/Jules-Solutions/jules-installer/internal/audit"
	"github.com/Jules-Solutions/jules-installer/internal/config"
	"github.com/Jules-Solutions/jules-installer/internal/setup"
)

// renderTier renders the Tier 1 vs Tier 2 picker screen shown right after Welcome.
func renderTier(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  Choose Your Install"))
	sb.WriteString("\n")
	if m.width > 4 {
		sb.WriteString(HRule(m.width - 4))
	}
	sb.WriteString("\n\n")

	sb.WriteString("  " + subtitleStyle.Render("How much of Jules.Solutions do you want locally?"))
	sb.WriteString("\n\n")

	// --- Tier 1 option ---
	tier1Label := "Full install"
	tier1Desc := []string{
		"Local vault (~/Your.Life) with Claude Code + jules-local CLI.",
		"Best for daily drivers — full vault, playgrounds, ops tools.",
		"Requires: Git, Python, uv. The audit can install what's missing.",
	}
	if m.tierCursor == tierChoiceFull {
		sb.WriteString("  " + highlightStyle.Render("> 1. "+tier1Label))
		sb.WriteString("\n")
		for _, line := range tier1Desc {
			sb.WriteString("     " + bodyStyle.Render(line) + "\n")
		}
	} else {
		sb.WriteString("    " + bodyStyle.Render("1. "+tier1Label))
		sb.WriteString("\n")
		for _, line := range tier1Desc {
			sb.WriteString("     " + mutedStyle.Render(line) + "\n")
		}
	}
	sb.WriteString("\n")

	// --- Tier 2 option ---
	tier2Label := "Remote only (MCP)"
	tier2Desc := []string{
		"Just wire Claude Code up to jules.solutions over MCP. No vault, no jules-local.",
		"Best for: trying it out, headless servers, users who don't want a local install.",
		"Writes ~/.claude/.mcp.json — Claude Code picks it up on next restart.",
	}
	if m.tierCursor == tierChoiceRemote {
		sb.WriteString("  " + highlightStyle.Render("> 2. "+tier2Label))
		sb.WriteString("\n")
		for _, line := range tier2Desc {
			sb.WriteString("     " + bodyStyle.Render(line) + "\n")
		}
	} else {
		sb.WriteString("    " + bodyStyle.Render("2. "+tier2Label))
		sb.WriteString("\n")
		for _, line := range tier2Desc {
			sb.WriteString("     " + mutedStyle.Render(line) + "\n")
		}
	}
	sb.WriteString("\n")

	sb.WriteString("  " + mutedStyle.Render("↑/↓ or 1/2 to choose, ") +
		highlightStyle.Render("Enter") + mutedStyle.Render(" to confirm, ") +
		highlightStyle.Render("q") + mutedStyle.Render(" to quit."))
	sb.WriteString("\n")
	sb.WriteString("  " + mutedStyle.Render("(You can run ") +
		codeStyle.Render("jules-setup --tier 1") +
		mutedStyle.Render(" later to upgrade from Remote to Full.)"))
	sb.WriteString("\n")

	return sb.String()
}

// renderRerun renders the re-run action menu shown when a valid config already exists.
func renderRerun(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  Setup Already Configured"))
	sb.WriteString("\n")
	if m.width > 4 {
		sb.WriteString(HRule(m.width - 4))
	}
	sb.WriteString("\n\n")

	// Show current configured state so user knows what they have.
	tierLabel := "Tier 1 (Full install)"
	if m.tier == config.TierRemote {
		tierLabel = "Tier 2 (Remote MCP only)"
	}
	sb.WriteString("  " + KeyValueRow("Tier", tierLabel))
	sb.WriteString("\n")
	sb.WriteString("  " + KeyValueRow("API Key", truncateKey(m.apiKey)))
	sb.WriteString("\n")
	if cfg, err := config.LoadConfig(); err == nil {
		if cfg.Local.VaultPath != "" {
			sb.WriteString("  " + KeyValueRow("Vault", cfg.Local.VaultPath))
			sb.WriteString("\n")
		}
		if cfg.Local.MCPPath != "" {
			sb.WriteString("  " + KeyValueRow("MCP Config", cfg.Local.MCPPath))
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
	sb.WriteString("  " + subtitleStyle.Render("What would you like to do?"))
	sb.WriteString("\n\n")

	opts := []struct {
		label string
		desc  string
	}{
		{"Change tier", "Switch between Full install and Remote-only, re-run setup."},
		{"Re-run audit", "Re-check the environment (git, docker, python, CC, etc.)."},
		{"Re-write MCP config", "Regenerate .mcp.json at the recorded path. Use after an API key rotation."},
		{"Exit", "Leave the current setup untouched."},
	}

	for i, o := range opts {
		if rerunChoice(i) == m.rerunCursor {
			sb.WriteString("  " + highlightStyle.Render(fmt.Sprintf("> %d. %s", i+1, o.label)))
			sb.WriteString("\n     " + bodyStyle.Render(o.desc) + "\n")
		} else {
			sb.WriteString("    " + bodyStyle.Render(fmt.Sprintf("%d. %s", i+1, o.label)))
			sb.WriteString("\n     " + mutedStyle.Render(o.desc) + "\n")
		}
	}
	sb.WriteString("\n")

	if m.rerunMessage != "" {
		if m.rerunMessageErr {
			sb.WriteString("  " + errorStyle.Render(m.rerunMessage) + "\n\n")
		} else {
			sb.WriteString("  " + successStyle.Render(m.rerunMessage) + "\n\n")
		}
	}

	sb.WriteString("  " + mutedStyle.Render("↑/↓ or 1-4 to choose, ") +
		highlightStyle.Render("Enter") + mutedStyle.Render(" to confirm."))
	sb.WriteString("\n")

	return sb.String()
}

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

	// Show check results. For Tier 2 users, Python/Docker/Node fails are
	// demoted to warnings — they don't block a remote-only install.
	tierStr := string(m.tier)
	for _, c := range m.auditResults {
		detail := c.Version
		if c.Detail != "" {
			if detail != "" {
				detail += " — " + c.Detail
			} else {
				detail = c.Detail
			}
		}
		displayStatus := c.StatusForTier(tierStr)
		// Append a small hint when a fail was demoted, so the user knows
		// why a missing tool is showing as a warning.
		if m.tier == config.TierRemote && c.Status == audit.StatusFail && displayStatus == audit.StatusWarn {
			if detail == "" {
				detail = "not needed for Tier 2"
			} else {
				detail = detail + " — not needed for Tier 2"
			}
		}
		sb.WriteString("  " + StatusLine(displayStatus, fmt.Sprintf("%-14s", c.Name), detail) + "\n")
	}

	// Summary line — counts reflect the tier-adjusted statuses, so a Tier 2
	// user doesn't see "3 failed" on items they explicitly opted out of.
	pass, fail, warn := 0, 0, 0
	for _, c := range m.auditResults {
		switch c.StatusForTier(tierStr) {
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
		// Tier 2 users only get offered tools they actually need (CC etc.),
		// never Python/Docker/Node.
		installable := audit.CountInstallableForTier(m.auditResults, tierStr)
		if installable == 0 {
			// Nothing tier-relevant to install. Skip the offer entirely.
			sb.WriteString("  " + mutedStyle.Render("Press ") + highlightStyle.Render("Enter") +
				mutedStyle.Render(" to continue…"))
			break
		}
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

// renderDownload renders the vault download progress screen.
func renderDownload(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render("  Vault Setup"))
	sb.WriteString("\n")
	if m.width > 4 {
		sb.WriteString(HRule(m.width - 4))
	}
	sb.WriteString("\n\n")

	if m.vaultDownloadMethod == "" {
		// Still in progress.
		sb.WriteString("  " + SpinnerWithMessage(m.spinnerFrame, "Setting up your vault…"))
		return sb.String()
	}

	// Done — show result.
	switch m.vaultDownloadMethod {
	case "existing":
		sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("Vault already exists — skipped download"))
	case "git_clone":
		sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("Vault cloned from GitHub"))
	case "scaffold":
		sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("Fresh vault scaffolded"))
	}
	sb.WriteString("\n")

	if m.vaultDownloadErr != nil {
		sb.WriteString("  " + warningStyle.Render("!") + " " + mutedStyle.Render(m.vaultDownloadErr.Error()))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderConfig renders the config writing screen.
func renderConfig(m Model) string {
	_ = m
	return "\n" + titleStyle.Render("  Writing Configuration") + "\n\n" +
		SpinnerWithMessage(0, "Writing config files…") + "\n"
}

// renderDone renders the completion screen with an honest, tier-aware summary.
func renderDone(m Model) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(successStyle.Render("  ✓ Setup Complete"))
	sb.WriteString("\n\n")

	// Tier indicator — makes it crystal clear to the user which path ran.
	tierLabel := "Full install (Tier 1)"
	if m.tier == config.TierRemote {
		tierLabel = "Remote MCP only (Tier 2)"
	}
	sb.WriteString("  " + KeyValueRow("Tier", tierLabel))
	sb.WriteString("\n\n")

	// Auth + config saved.
	sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("Authenticated via ") + highlightStyle.Render(string(m.authMethod)))
	sb.WriteString("\n")

	configPath, _ := config.ConfigPath()
	sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("API key saved to ") + mutedStyle.Render(configPath))
	sb.WriteString("\n")

	// Audit summary — use tier-aware counts so Tier 2 doesn't look like a failure.
	if len(m.auditResults) > 0 {
		tierStr := string(m.tier)
		pass := 0
		for _, c := range m.auditResults {
			if c.StatusForTier(tierStr) == audit.StatusPass {
				pass++
			}
		}
		sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render(fmt.Sprintf("Environment audited (%d/%d checks passed)", pass, len(m.auditResults))))
		sb.WriteString("\n")
	}

	// Tier-specific artifacts.
	if m.tier == config.TierRemote {
		// Tier 2: just the MCP config write.
		if m.mcpPathWritten != "" {
			sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("MCP config written (") + mutedStyle.Render(m.mcpPathWritten) + bodyStyle.Render(")"))
			sb.WriteString("\n")
		}
	} else {
		// Tier 1: vault + MCP + jules-local.
		switch m.vaultDownloadMethod {
		case "existing":
			sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("Vault already set up"))
		case "git_clone":
			sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("Vault cloned from GitHub"))
		case "scaffold":
			sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("Fresh vault scaffolded (offline fallback)"))
		default:
			sb.WriteString("  " + mutedStyle.Render("⟳") + " " + bodyStyle.Render("Vault not yet downloaded"))
		}
		sb.WriteString("\n")

		if m.setupConfigMCP && m.mcpPathWritten != "" {
			sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("MCP config written (") + mutedStyle.Render(m.mcpPathWritten) + bodyStyle.Render(")"))
			sb.WriteString("\n")
		}

		// jules-local install status (Tier 1 only).
		if m.installLocalErr != nil {
			sb.WriteString("  " + warningStyle.Render("⚠") + " " + bodyStyle.Render("jules-local not installed: "+m.installLocalErr.Error()))
			sb.WriteString("\n")
			sb.WriteString("    " + mutedStyle.Render("Install manually: uv tool install git+https://github.com/Jules-Solutions/jules-local.git"))
			sb.WriteString("\n")
		} else if setup.JulesLocalVersion() != "" {
			sb.WriteString("  " + successStyle.Render("✓") + " " + bodyStyle.Render("jules-local installed ("+setup.JulesLocalVersion()+")"))
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")

	// Next steps — tier-specific.
	vaultPath := ""
	if m.setupVaultInput != nil {
		vaultPath = m.setupVaultInput.Value()
	}
	sb.WriteString(subtitleStyle.Render("  Next steps:"))
	sb.WriteString("\n")
	if m.tier == config.TierRemote {
		sb.WriteString("  " + bodyStyle.Render("  1. Restart Claude Code (it re-reads ~/.claude/.mcp.json on launch)"))
		sb.WriteString("\n")
		sb.WriteString("  " + bodyStyle.Render("  2. In any CC session, try: ") + codeStyle.Render(" /mcp ") + bodyStyle.Render(" to confirm jules.solutions is connected"))
		sb.WriteString("\n")
		sb.WriteString("  " + mutedStyle.Render("  Later, upgrade to full install with: ") + codeStyle.Render(" jules-setup --tier 1 "))
	} else if vaultPath != "" {
		sb.WriteString("  " + bodyStyle.Render("  1. cd "+vaultPath))
		sb.WriteString("\n")
		sb.WriteString("  " + bodyStyle.Render("  2. claude  (start Claude Code in your vault)"))
	} else {
		sb.WriteString("  " + bodyStyle.Render("  1. Your API key is saved — CLI tools will pick it up automatically"))
		sb.WriteString("\n")
		sb.WriteString("  " + bodyStyle.Render("  2. Run claude in your vault directory"))
	}
	sb.WriteString("\n\n")

	// Launch prompt — Tier 2 has no vault to launch CC in, so the prompt is different.
	if m.tier == config.TierRemote {
		sb.WriteString("  " + mutedStyle.Render("Press ") + highlightStyle.Render("Enter") + mutedStyle.Render(" or ") +
			highlightStyle.Render("q") + mutedStyle.Render(" to exit."))
		sb.WriteString("\n")
		return sb.String()
	}

	// Tier 1: offer to launch CC in the vault directory.
	if m.launchAttempted {
		if m.launchErr != nil {
			// Launch failed — show manual instructions.
			sb.WriteString("  " + warningStyle.Render("! Could not open terminal automatically."))
			sb.WriteString("\n\n")
			sb.WriteString("  " + subtitleStyle.Render("To start Claude Code manually:"))
			sb.WriteString("\n")
			if vaultPath != "" {
				sb.WriteString("  " + codeStyle.Render("  cd "+vaultPath+"  "))
				sb.WriteString("\n")
				sb.WriteString("  " + codeStyle.Render("  claude  "))
			} else {
				sb.WriteString("  " + codeStyle.Render("  claude  "))
			}
			sb.WriteString("\n\n")
			sb.WriteString("  " + mutedStyle.Render("Press ") + highlightStyle.Render("q") + mutedStyle.Render(" to exit."))
		} else {
			sb.WriteString("  " + successStyle.Render("✓ Launching Claude Code…"))
			sb.WriteString("\n")
			sb.WriteString("  " + mutedStyle.Render("Press ") + highlightStyle.Render("q") + mutedStyle.Render(" to exit."))
		}
	} else {
		sb.WriteString("  " + mutedStyle.Render("Press ") + highlightStyle.Render("Enter") + mutedStyle.Render(" to launch Claude Code, or ") +
			highlightStyle.Render("q") + mutedStyle.Render(" to exit."))
	}
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
