// Package audit — git.go checks for a usable git installation.
package audit

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckGit verifies that git is installed and returns its version and config.
func CheckGit() Check {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return Check{
			Name:   "Git",
			Status: StatusFail,
			Detail: "not found — install at https://git-scm.com",
		}
	}

	version := strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "git version ")
	// Strip trailing info like " (Apple Git-146)".
	if idx := strings.IndexByte(version, ' '); idx > 0 {
		version = version[:idx]
	}

	// Get user.name and user.email for display.
	name := gitConfig("user.name")
	email := gitConfig("user.email")

	detail := ""
	if name != "" && email != "" {
		detail = fmt.Sprintf("%s <%s>", name, email)
	} else if name != "" {
		detail = name + " (email not set)"
	} else {
		detail = "user.name not configured"
	}

	return Check{Name: "Git", Status: StatusPass, Version: version, Detail: detail}
}

// gitConfig runs git config --global <key> and returns the value.
func gitConfig(key string) string {
	out, err := exec.Command("git", "config", "--global", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
