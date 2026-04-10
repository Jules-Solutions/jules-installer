// Package audit — claude.go checks for Claude Code CLI.
package audit

import (
	"os/exec"
	"strings"
)

// CheckClaude verifies Claude Code is installed. This is a critical
// dependency — the entire platform is operated through Claude Code.
func CheckClaude() Check {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		return Check{
			Name:   "Claude Code",
			Status: StatusFail,
			Detail: "not found — install at https://claude.ai/download",
		}
	}

	version := strings.TrimSpace(string(out))
	// Output may be "claude 1.0.33" or just a version string.
	version = strings.TrimPrefix(version, "claude ")

	return Check{Name: "Claude Code", Status: StatusPass, Version: version}
}
