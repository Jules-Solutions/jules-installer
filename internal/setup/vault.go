// Package setup — vault.go handles downloading/scaffolding the user's vault.
package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DownloadVault attempts to set up the vault at the given path.
// Tries strategies in order: existing dir → git clone → empty scaffold.
// Returns the method used ("existing", "git_clone", "scaffold") and any error.
func DownloadVault(vaultPath string) (string, error) {
	// Strategy 0: vault directory already exists (re-run case).
	if info, err := os.Stat(vaultPath); err == nil && info.IsDir() {
		// Check if it has content (not just an empty dir).
		entries, _ := os.ReadDir(vaultPath)
		if len(entries) > 0 {
			return "existing", nil
		}
	}

	// Strategy 1: git clone via SSH (requires git + SSH keys + GitHub access).
	if hasGit() {
		// Try SSH clone from org repo.
		username := filepath.Base(vaultPath)
		// Strip ".Life" suffix to get username.
		if len(username) > 5 && username[len(username)-5:] == ".Life" {
			username = username[:len(username)-5]
		}

		repoSSH := fmt.Sprintf("git@github.com:Jules-Solutions/%s-vault.git", username)
		err := gitClone(repoSSH, vaultPath)
		if err == nil {
			return "git_clone", nil
		}

		// Try HTTPS clone (might work if user has gh credential helper).
		repoHTTPS := fmt.Sprintf("https://github.com/Jules-Solutions/%s-vault.git", username)
		err = gitClone(repoHTTPS, vaultPath)
		if err == nil {
			return "git_clone", nil
		}
	}

	// Strategy 2: scaffold an empty vault structure.
	if err := scaffoldVault(vaultPath); err != nil {
		return "", fmt.Errorf("scaffold failed: %w", err)
	}
	return "scaffold", nil
}

// hasGit checks if git is available.
func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// gitClone runs git clone with depth=1.
func gitClone(repoURL, targetDir string) error {
	cmd := exec.Command("git", "clone", "--depth=1", repoURL, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// scaffoldVault creates a minimal vault directory structure.
func scaffoldVault(vaultPath string) error {
	dirs := []string{
		".claude",
		".claude/agents",
		".claude/skills",
		".claude/hooks",
		".claude/rules",
		".claude/memory",
		".Life",
		".Life/templates",
		".Life/pipelines",
		"Settings",
		"Workstreams",
		"Workstreams/getting-started",
		"Projects",
		"Inbox",
		"Areas",
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(vaultPath, d), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}

	// Write CLAUDE.md — minimal bootstrap instructions.
	claudeMD := `# Vault Instructions

> Your Jules.Solutions vault. Claude Code reads this file on startup.

## Getting Started

This vault was created by the Jules.Solutions installer.
Run Claude Code in this directory to get started.
`
	if err := os.WriteFile(filepath.Join(vaultPath, "CLAUDE.md"), []byte(claudeMD), 0o644); err != nil {
		return err
	}

	// Write Character-Sheet.md — placeholder.
	charSheet := `---
type: character-sheet
---

# Character Sheet

> Fill this in to customize how Claude Code interacts with you.

## About Me

- **Name:**
- **Role:**
- **Goals:**
`
	if err := os.WriteFile(filepath.Join(vaultPath, "Settings", "Character-Sheet.md"), []byte(charSheet), 0o644); err != nil {
		return err
	}

	// Init git repo.
	if hasGit() {
		cmd := exec.Command("git", "init", vaultPath)
		_ = cmd.Run() // best-effort
	}

	return nil
}
