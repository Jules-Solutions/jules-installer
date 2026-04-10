// Package audit — python.go checks for Python and uv package manager.
package audit

import (
	"os/exec"
	"runtime"
	"strings"
)

// CheckPython verifies Python is installed and checks for uv.
func CheckPython() Check {
	var pyVersion string

	// Try python3 first (Unix default), then python (Windows default).
	cmds := []string{"python3", "python"}
	if runtime.GOOS == "windows" {
		cmds = []string{"python", "python3", "py"}
	}

	for _, cmd := range cmds {
		out, err := exec.Command(cmd, "--version").Output()
		if err == nil {
			pyVersion = strings.TrimSpace(string(out))
			pyVersion = strings.TrimPrefix(pyVersion, "Python ")
			break
		}
	}

	if pyVersion == "" {
		return Check{Name: "Python", Status: StatusFail, Detail: "not found — install at https://python.org"}
	}

	// Check for uv.
	detail := ""
	uvOut, err := exec.Command("uv", "--version").Output()
	if err == nil {
		uvVersion := strings.TrimSpace(string(uvOut))
		uvVersion = strings.TrimPrefix(uvVersion, "uv ")
		detail = "uv " + uvVersion
	} else {
		detail = "uv not found (recommended: https://docs.astral.sh/uv/)"
	}

	status := StatusPass
	if detail != "" && strings.Contains(detail, "not found") {
		status = StatusWarn
	}

	return Check{Name: "Python", Status: status, Version: pyVersion, Detail: detail}
}
