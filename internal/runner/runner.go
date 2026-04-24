// Package runner implements the non-interactive "--yes" installer flow.
//
// It runs the same logical pipeline as the Bubbletea TUI — authenticate,
// audit, tier-branched setup, write artifacts — but without any user
// prompts. Every decision must come from flags, environment variables, or
// a pre-existing config.toml.
//
// Callers that want the interactive experience should keep using the tui
// package. This package exists for:
//
//   - Scripted org-wide installs (CI bootstrap, unattended re-runs)
//   - Integration tests that need a hermetic end-to-end flow
//   - Future automation use cases (e.g. remote-managed tenant onboarding)
//
// The headless flow is intentionally stricter than the TUI: if a required
// input is missing, Run returns an error rather than falling back to an
// interactive prompt (because there is no TTY by definition).
package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Jules-Solutions/jules-installer/internal/audit"
	"github.com/Jules-Solutions/jules-installer/internal/config"
	"github.com/Jules-Solutions/jules-installer/internal/setup"
)

// Options controls what the headless installer does.
type Options struct {
	// AuthURL is the auth service base URL (e.g. https://auth.jules.solutions).
	// Used only for the config file record; headless mode never opens a browser.
	AuthURL string
	// Tier is required — Run refuses to guess. Pass config.TierFull or
	// config.TierRemote.
	Tier config.Tier
	// LocalToolsMCP, when non-nil and Tier == TierFull, overrides the config
	// value. Nil means "inherit from the existing config.toml (default false)."
	LocalToolsMCP *bool
	// Resume is accepted for parity with the TUI but has no effect on the
	// headless flow — the runner always does the same sequence of steps and
	// is idempotent: re-running with the same options is safe.
	Resume bool
	// VaultPath overrides the default vault location for Tier 1. Empty
	// preserves any existing config value; falls back to config.DefaultVaultPath().
	VaultPath string
	// APIKey, when non-empty, is written into config.toml before the run.
	// Use JULES_API_KEY env var as a convenience in test harnesses.
	// Usually left empty and the caller relies on an already-authenticated
	// config.toml.
	APIKey string
}

// Run executes the installer in non-interactive mode.
//
// Output is streamed to w as human-readable progress lines so CI logs
// and interactive runs both show useful breadcrumbs.
//
// Returns nil on success. On failure, returns an error describing the
// first unrecoverable problem; partial state is left in config.toml for
// a subsequent Run to continue from.
func Run(w io.Writer, opts Options) error {
	if !opts.Tier.Valid() {
		return fmt.Errorf("runner.Run requires a valid Tier (got %q)", opts.Tier)
	}
	if opts.AuthURL == "" {
		opts.AuthURL = "https://auth.jules.solutions"
	}

	// --- Config: load, merge, save early so the tier is persisted even if
	// a later step fails. ---
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	fillConfigDefaults(&cfg)
	cfg.Auth.AuthURL = opts.AuthURL

	// API key precedence: explicit flag > env var > existing config.
	if opts.APIKey != "" {
		cfg.Auth.APIKey = opts.APIKey
	} else if env := os.Getenv("JULES_API_KEY"); env != "" && cfg.Auth.APIKey == "" {
		cfg.Auth.APIKey = env
	}
	if cfg.Auth.APIKey == "" || !strings.HasPrefix(cfg.Auth.APIKey, "dck_") {
		return fmt.Errorf("no valid API key — headless mode requires one in ~/.config/jules/config.toml, JULES_API_KEY env, or via caller opts")
	}

	cfg.Local.Tier = opts.Tier
	if opts.VaultPath != "" {
		cfg.Local.VaultPath = opts.VaultPath
	} else if cfg.Local.VaultPath == "" {
		cfg.Local.VaultPath = config.DefaultVaultPath()
	}
	if opts.LocalToolsMCP != nil {
		cfg.Local.LocalToolsMCP = *opts.LocalToolsMCP
	}

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(w, "✓ Tier: %s\n", cfg.Local.Tier)

	// --- Audit: always runs, never interactive. Install offers are skipped
	// (headless mode cannot ask "install Python now?") — caller sees the
	// summary and decides whether to intervene. ---
	checks := audit.RunAudit()
	tierStr := string(cfg.Local.Tier)
	pass, warn, fail := 0, 0, 0
	for _, c := range checks {
		switch c.StatusForTier(tierStr) {
		case audit.StatusPass:
			pass++
		case audit.StatusWarn:
			warn++
		case audit.StatusFail:
			fail++
		}
	}
	fmt.Fprintf(w, "✓ Audit: %d passed, %d warnings, %d failed\n", pass, warn, fail)
	// We don't abort on fails — a Tier 2 user might legitimately run on a box
	// that fails Python checks, and a Tier 1 user might want to install
	// jules-local manually later. The caller sees the summary and can decide.

	// --- Tier branch ---
	switch cfg.Local.Tier {
	case config.TierFull:
		return runTier1(w, &cfg)
	case config.TierRemote:
		return runTier2(w, &cfg)
	}

	return fmt.Errorf("unreachable: unknown tier %q", cfg.Local.Tier)
}

// runTier1 is the full-install headless tail: vault download/scaffold,
// MCP config write, jules-local CLI install.
func runTier1(w io.Writer, cfg *config.Config) error {
	// Ensure vault dir exists (download handles the content).
	params := setup.ScaffoldParams{
		APIKey: cfg.Auth.APIKey,
		APIURL: cfg.Auth.APIURL,
		MCPURL: effectiveMCPURL(cfg),
	}
	params.VaultName = filepath.Base(cfg.Local.VaultPath)
	if strings.HasSuffix(params.VaultName, ".Life") {
		params.UserName = strings.TrimSuffix(params.VaultName, ".Life")
	} else {
		params.UserName = params.VaultName
	}

	method, err := setup.DownloadVaultWithParams(cfg.Local.VaultPath, params)
	if err != nil {
		return fmt.Errorf("vault download/scaffold: %w", err)
	}
	fmt.Fprintf(w, "✓ Vault: %s (%s)\n", cfg.Local.VaultPath, method)

	mcpPath, err := setup.WriteMCPConfigForTier(
		config.TierFull, cfg.Local.VaultPath, cfg.Auth.APIKey, effectiveMCPURL(cfg),
		setup.MCPWriteOptions{LocalToolsMCP: cfg.Local.LocalToolsMCP},
	)
	if err != nil {
		return fmt.Errorf("write .mcp.json: %w", err)
	}
	cfg.Local.MCPPath = mcpPath
	_ = config.SaveConfig(*cfg)
	fmt.Fprintf(w, "✓ MCP: %s (local_tools_mcp=%v)\n", mcpPath, cfg.Local.LocalToolsMCP)

	// jules-local install is best-effort in the TUI too — keep that.
	if err := setup.InstallJulesLocal(); err != nil {
		fmt.Fprintf(w, "⚠ jules-local install skipped: %v\n", err)
	} else if v := setup.JulesLocalVersion(); v != "" {
		fmt.Fprintf(w, "✓ jules-local: %s\n", v)
	}

	return nil
}

// runTier2 is the remote-only headless tail: just write ~/.claude/.mcp.json.
func runTier2(w io.Writer, cfg *config.Config) error {
	mcpPath, err := setup.WriteMCPConfigForTier(
		config.TierRemote, "", cfg.Auth.APIKey, effectiveMCPURL(cfg),
	)
	if err != nil {
		return fmt.Errorf("write .mcp.json: %w", err)
	}
	cfg.Local.MCPPath = mcpPath
	_ = config.SaveConfig(*cfg)
	fmt.Fprintf(w, "✓ MCP: %s\n", mcpPath)
	return nil
}

// fillConfigDefaults populates sensible defaults for any fields the caller
// didn't set. Idempotent — safe to call on a config loaded from disk.
func fillConfigDefaults(cfg *config.Config) {
	if cfg.Auth.APIURL == "" {
		cfg.Auth.APIURL = "https://api.jules.solutions"
	}
	if cfg.Auth.MCPURL == "" {
		cfg.Auth.MCPURL = "https://mcp.jules.solutions/sse"
	}
	if cfg.Auth.AuthURL == "" {
		cfg.Auth.AuthURL = "https://auth.jules.solutions"
	}
}

// effectiveMCPURL returns the MCP SSE URL from config, falling back to
// the production default.
func effectiveMCPURL(cfg *config.Config) string {
	if cfg.Auth.MCPURL != "" {
		return cfg.Auth.MCPURL
	}
	return "https://mcp.jules.solutions/sse"
}
