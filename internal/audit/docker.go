// Package audit — docker.go checks for Docker installation and daemon status.
package audit

import (
	"os/exec"
	"strings"
)

// CheckDocker verifies Docker is installed and the daemon is running.
func CheckDocker() Check {
	// Try docker info first (requires running daemon).
	out, err := exec.Command("docker", "info", "--format", "{{.ServerVersion}}").Output()
	if err == nil {
		version := strings.TrimSpace(string(out))
		if version != "" {
			return Check{Name: "Docker", Status: StatusPass, Version: version, Detail: "running"}
		}
	}

	// Daemon not running — try docker --version to see if it's installed.
	out, err = exec.Command("docker", "--version").Output()
	if err != nil {
		return Check{Name: "Docker", Status: StatusFail, Detail: "not found — install at https://docker.com/get-started"}
	}

	version := parseVersionPrefix(string(out), "Docker version ")
	return Check{Name: "Docker", Status: StatusWarn, Version: version, Detail: "installed but daemon not running"}
}

// parseVersionPrefix extracts the version after a known prefix.
// e.g. "Docker version 27.1.2, build ..." → "27.1.2"
func parseVersionPrefix(s, prefix string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, prefix); idx >= 0 {
		s = s[idx+len(prefix):]
	}
	if comma := strings.IndexByte(s, ','); comma >= 0 {
		s = s[:comma]
	}
	return strings.TrimSpace(s)
}
