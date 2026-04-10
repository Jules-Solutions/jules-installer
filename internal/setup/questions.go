// Package setup handles interactive setup questions, vault download, and config writing.
package setup

import (
	"os"
	"path/filepath"

	"github.com/Jules-Solutions/jules-installer/internal/config"
)

// Answers holds the user's responses to setup questions.
type Answers struct {
	VaultPath string
	Shell     string
}

// DefaultAnswers returns pre-filled defaults for all setup questions.
// These are shown to the user as editable suggestions.
func DefaultAnswers() Answers {
	return Answers{
		VaultPath: config.DefaultVaultPath(),
		Shell:     detectDefaultShell(),
	}
}

// detectDefaultShell returns the most likely shell name for the current platform.
// TODO(Phase 2): use the audit result from audit.CheckPlatform() instead.
func detectDefaultShell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		return filepath.Base(sh)
	}
	if os.Getenv("PSModulePath") != "" {
		return "powershell"
	}
	return "bash"
}
