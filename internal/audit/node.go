// Package audit — node.go checks for Node.js and pnpm.
package audit

import (
	"os/exec"
	"strings"
)

// CheckNode verifies Node.js is installed. Node is optional — only needed
// for dashboard local development.
func CheckNode() Check {
	out, err := exec.Command("node", "--version").Output()
	if err != nil {
		return Check{Name: "Node.js", Status: StatusSkip, Detail: "optional — only for dashboard local dev"}
	}

	version := strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "v")

	// Check for pnpm.
	detail := ""
	pnpmOut, err := exec.Command("pnpm", "--version").Output()
	if err == nil {
		detail = "pnpm " + strings.TrimSpace(string(pnpmOut))
	}

	return Check{Name: "Node.js", Status: StatusPass, Version: version, Detail: detail}
}
