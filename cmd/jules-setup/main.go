// Command jules-setup is the Jules.Solutions cross-platform installer.
//
// It authenticates the user, audits their environment, downloads their vault,
// and configures Claude Code for first use.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Jules-Solutions/jules-installer/internal/tui"
	"github.com/Jules-Solutions/jules-installer/pkg/version"
)

const authURL = "https://auth.jules.solutions"

func main() {
	// Print version info if requested.
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("jules-setup", version.String())
		os.Exit(0)
	}

	// Build the root TUI model.
	m := tui.NewModel(authURL, version.String())

	// Create and run the Bubbletea program.
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
}
