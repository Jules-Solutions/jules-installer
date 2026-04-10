// Package audit — platform.go detects operating system, architecture, and available shells.
package audit

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// CheckPlatform detects the OS, architecture, and primary shell.
// This check always passes — we just record what we find.
func CheckPlatform() Check {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	shell := detectShell()

	detail := fmt.Sprintf("OS: %s, Arch: %s, Shell: %s", goos, goarch, shell)

	return Check{
		Name:    "Platform",
		Status:  StatusPass,
		Version: fmt.Sprintf("%s/%s", goos, goarch),
		Detail:  detail,
	}
}

// detectShell identifies the user's active shell by inspecting environment variables
// and the existence of common shell executables.
func detectShell() string {
	// SHELL env var is reliable on Unix.
	if sh := os.Getenv("SHELL"); sh != "" {
		parts := strings.Split(sh, "/")
		return parts[len(parts)-1]
	}

	// On Windows, check COMSPEC and PSModulePath.
	if runtime.GOOS == "windows" {
		if ps := os.Getenv("PSModulePath"); ps != "" {
			return "powershell"
		}
		if cs := os.Getenv("COMSPEC"); cs != "" {
			return "cmd"
		}
	}

	return "unknown"
}
