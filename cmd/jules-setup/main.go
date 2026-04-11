// Command jules-setup is the Jules.Solutions cross-platform installer.
//
// It authenticates the user, audits their environment, downloads their vault,
// and configures Claude Code for first use.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

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
	flag.Parse()

	if *versionFlag || *vFlag {
		fmt.Println("jules-setup", version.String())
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
	m := tui.NewModel(authURL, version.String())
	if *resumeFlag {
		m = tui.NewModelWithResume(authURL, version.String())
	}

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
