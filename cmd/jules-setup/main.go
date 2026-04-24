// Command jules-setup is the Jules.Solutions cross-platform installer.
//
// It authenticates the user, audits their environment, downloads their vault,
// and configures Claude Code for first use.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Jules-Solutions/jules-installer/internal/config"
	"github.com/Jules-Solutions/jules-installer/internal/runner"
	"github.com/Jules-Solutions/jules-installer/internal/tui"
	"github.com/Jules-Solutions/jules-installer/internal/update"
	"github.com/Jules-Solutions/jules-installer/pkg/version"
)

const authURL = "https://auth.jules.solutions"

func main() {
	// --- CLI flags ---
	versionFlag := flag.Bool("version", false, "Print version and exit")
	vFlag := flag.Bool("v", false, "Print version and exit (short)")
	resumeFlag := flag.Bool("resume", false, "Resume from existing config, skipping completed steps")
	// --tier skips the interactive Tier picker. Accepts '1'/'2' (friendly),
	// 'tier1'/'tier2' (canonical), or 'full'/'remote' (descriptive).
	tierFlag := flag.String("tier", "", "Skip tier picker: 1|tier1|full (local install) or 2|tier2|remote (MCP-only)")
	// --local-tools-mcp (Tier 1 only) — when true, .mcp.json also registers a
	// jules-local stdio server. Lets Claude Code call exec/file/terminal tools
	// against the local machine. Default false. Use an empty string to leave
	// the choice to the TUI / config; "true"/"false" force the value.
	localToolsFlag := flag.String("local-tools-mcp", "", "(Tier 1) expose jules-local stdio MCP server: true|false (default: prompt in TUI)")
	// --yes runs the installer non-interactively. Requires --tier. Auth key
	// must already be in config.toml or supplied via JULES_API_KEY env var.
	yesFlag := flag.Bool("yes", false, "Non-interactive mode: skip all prompts. Requires --tier and a pre-existing API key.")
	flag.Parse()

	if *versionFlag || *vFlag {
		fmt.Println("jules-setup", version.String())
		os.Exit(0)
	}

	// Resolve --tier flag → canonical config.Tier value.
	// Empty means "no flag, show the picker screen."
	tierChoice, err := resolveTierFlag(*tierFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	// Resolve --local-tools-mcp to a *bool. nil = "prompt" (TUI decides);
	// non-nil = flag wins over any config value or TUI default.
	localToolsChoice, err := resolveBoolFlag(*localToolsFlag, "local-tools-mcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	// --yes mode: dispatch to the headless runner instead of the TUI.
	// Fail fast on the preconditions the runner needs.
	if *yesFlag {
		if tierChoice == "" {
			fmt.Fprintln(os.Stderr, "error: --yes requires --tier (installer cannot pick a tier for you)")
			os.Exit(2)
		}
		runOpts := runner.Options{
			AuthURL:       authURL,
			Tier:          tierChoice,
			LocalToolsMCP: localToolsChoice,
			Resume:        *resumeFlag,
		}
		if err := runner.Run(os.Stdout, runOpts); err != nil {
			fmt.Fprintf(os.Stderr, "headless install failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// --- Self-update check (non-blocking, best-effort) ---
	// Run concurrently so it doesn't slow startup on slow networks.
	updateCh := make(chan string, 1)
	go func() {
		info, _ := update.CheckForUpdate(version.Version)
		msg := update.FormatUpdateMessage(info)
		updateCh <- msg
	}()

	// --- Build the root TUI model ---
	opts := tui.ModelOptions{
		AuthURL:       authURL,
		Version:       version.String(),
		Resume:        *resumeFlag,
		Tier:          tierChoice,
		LocalToolsMCP: localToolsChoice,
	}
	m := tui.NewModelWithOptions(opts)

	// --- Create and run the Bubbletea program ---
	// AltScreen gives us a clean full-screen canvas; MouseCellMotion
	// enables future mouse support without breaking keyboard input.
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "installer error: %v\n", err)
		os.Exit(1)
	}

	// Print update notice after TUI exits (avoids cluttering the TUI canvas).
	select {
	case msg := <-updateCh:
		if msg != "" {
			fmt.Fprintln(os.Stderr, msg)
		}
	default:
		// Update check still in flight or returned empty — skip.
	}
}

// resolveTierFlag maps user-facing --tier flag values to canonical config.Tier.
// Accepts: "" (no flag), "1"/"tier1"/"full", "2"/"tier2"/"remote"
// Returns empty Tier for empty flag (signals "show picker in TUI").
func resolveTierFlag(raw string) (config.Tier, error) {
	if raw == "" {
		return "", nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "tier1", "full":
		return config.TierFull, nil
	case "2", "tier2", "remote":
		return config.TierRemote, nil
	default:
		return "", fmt.Errorf(
			"invalid --tier value %q (expected: 1|tier1|full or 2|tier2|remote)",
			raw,
		)
	}
}

// resolveBoolFlag parses a string CLI flag as an optional bool.
//   "" → nil pointer (signals "not set; use default or TUI prompt")
//   "true"/"t"/"yes"/"y"/"1"/"on" → true
//   "false"/"f"/"no"/"n"/"0"/"off" → false
//   anything else → error
//
// We use this instead of flag.Bool because flag.Bool can't distinguish "not
// passed" from "passed as false" — and for --local-tools-mcp we need that
// distinction so the TUI knows when to prompt vs. when to just apply the flag.
func resolveBoolFlag(raw, name string) (*bool, error) {
	if raw == "" {
		return nil, nil
	}
	truthy := map[string]bool{"true": true, "t": true, "yes": true, "y": true, "1": true, "on": true}
	falsy := map[string]bool{"false": true, "f": true, "no": true, "n": true, "0": true, "off": true}
	s := strings.ToLower(strings.TrimSpace(raw))
	if truthy[s] {
		v := true
		return &v, nil
	}
	if falsy[s] {
		v := false
		return &v, nil
	}
	return nil, fmt.Errorf(
		"invalid --%s value %q (expected: true|false|yes|no|1|0|on|off)",
		name, raw,
	)
}
