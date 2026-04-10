// Package setup — vault.go handles downloading the user's vault from GitHub.
package setup

import "fmt"

// VaultSource identifies where the vault should be cloned from.
type VaultSource struct {
	RepoURL   string // e.g. git@github.com:user/Jules.Life.git
	Branch    string // e.g. main
	TargetDir string // local path to clone into
}

// DownloadVault clones the user's vault repository to the target directory.
//
// TODO(Phase 2): implement using:
//   - exec.Command("git", "clone", "--depth=1", src.RepoURL, src.TargetDir)
//   - Stream git output to progressFn so the TUI can show progress
//   - Handle SSH key setup and GitHub authentication
//   - Support sparse-checkout for large vaults
func DownloadVault(src VaultSource) error {
	// Stub: log intent but do not execute.
	fmt.Printf("[stub] would clone %s (branch: %s) → %s\n",
		src.RepoURL, src.Branch, src.TargetDir)
	return nil
}
