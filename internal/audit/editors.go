// Package audit — editors.go detects installed code editors and Obsidian.
package audit

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CheckEditors detects VS Code, Cursor, and Obsidian.
// These are optional — we just report what's available.
func CheckEditors() Check {
	var found []string

	// VS Code
	if v, ok := checkEditor("code", "--version"); ok {
		found = append(found, "VS Code "+v)
	}

	// Cursor
	if v, ok := checkEditor("cursor", "--version"); ok {
		found = append(found, "Cursor "+v)
	}

	// Obsidian — no CLI, check install paths.
	if checkObsidian() {
		found = append(found, "Obsidian")
	}

	if len(found) == 0 {
		return Check{
			Name:   "Editors",
			Status: StatusSkip,
			Detail: "none detected (recommend Obsidian for vault, VS Code for code)",
		}
	}

	return Check{
		Name:   "Editors",
		Status: StatusPass,
		Detail: strings.Join(found, ", "),
	}
}

// checkEditor runs `cmd --version` and returns the first line if successful.
func checkEditor(cmd, flag string) (string, bool) {
	out, err := exec.Command(cmd, flag).Output()
	if err != nil {
		return "", false
	}
	lines := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)
	if len(lines) > 0 && lines[0] != "" {
		return lines[0], true
	}
	return "", false
}

// checkObsidian checks platform-specific install paths for Obsidian.
func checkObsidian() bool {
	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			if _, err := os.Stat(filepath.Join(localAppData, "Obsidian")); err == nil {
				return true
			}
		}
	case "darwin":
		if _, err := os.Stat("/Applications/Obsidian.app"); err == nil {
			return true
		}
	default: // linux
		// Check common locations.
		paths := []string{
			"/usr/bin/obsidian",
			"/usr/local/bin/obsidian",
			"/snap/bin/obsidian",
		}
		home, _ := os.UserHomeDir()
		if home != "" {
			paths = append(paths, filepath.Join(home, ".local/bin/obsidian"))
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	}
	return false
}
