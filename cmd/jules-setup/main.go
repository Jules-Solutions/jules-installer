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
		AuthURL: authURL,
		Version: version.String(),
		Resume:  *resumeFlag,
		Tier:    tierChoice,
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
