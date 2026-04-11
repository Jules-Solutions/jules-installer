// Package setup — local.go installs jules-local (the platform runtime) as a Python package.
//
// jules-local provides the CLI (`jules` / `js`) and the MCP stdio server that
// Claude Code connects to. Without it, the .mcp.json config has nothing to call.
//
// Install strategy (in priority order):
//   1. uv tool install — fastest, isolated, recommended
//   2. pipx install — similar isolation, common fallback
//   3. pip install --user — last resort, no isolation
//
// The package is installed from GitHub (private repo, user must have access):
//   git+https://github.com/Jules-Solutions/jules-local.git
//
// After install, `jules`, `js`, `jules-local`, and `jules-local-mcp` are on PATH.
package setup

import (
	"fmt"
	"os/exec"
	"strings"
)

const julesLocalRepo = "git+https://github.com/Jules-Solutions/jules-local.git"

// InstallJulesLocal installs the jules-local Python package.
// Returns nil on success, error with details on failure.
// The error message includes manual install instructions.
func InstallJulesLocal() error {
	// Check if already installed.
	if isJulesLocalInstalled() {
		return nil
	}

	// Try install methods in priority order.
	methods := []struct {
		name string
		fn   func() error
	}{
		{"uv tool install", installWithUV},
		{"pipx install", installWithPipx},
		{"pip install --user", installWithPip},
	}

	var lastErr error
	for _, m := range methods {
		if err := m.fn(); err == nil {
			// Verify it actually works after install.
			if isJulesLocalInstalled() {
				return nil
			}
			lastErr = fmt.Errorf("%s completed but jules-local not found on PATH", m.name)
		} else {
			lastErr = fmt.Errorf("%s: %w", m.name, err)
		}
	}

	return fmt.Errorf(
		"could not install jules-local: %w\n\nManual install:\n  uv tool install %s\n  # or: pip install %s",
		lastErr, julesLocalRepo, julesLocalRepo,
	)
}

// isJulesLocalInstalled checks if jules-local is available on PATH.
func isJulesLocalInstalled() bool {
	// Check for any of the entry points.
	for _, cmd := range []string{"jules-local", "jules", "jules-local-mcp"} {
		if _, err := exec.LookPath(cmd); err == nil {
			return true
		}
	}
	return false
}

// installWithUV uses `uv tool install` for isolated installation.
func installWithUV() error {
	if _, err := exec.LookPath("uv"); err != nil {
		return fmt.Errorf("uv not found")
	}
	cmd := exec.Command("uv", "tool", "install", julesLocalRepo)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// installWithPipx uses `pipx install` as a fallback.
func installWithPipx() error {
	if _, err := exec.LookPath("pipx"); err != nil {
		return fmt.Errorf("pipx not found")
	}
	cmd := exec.Command("pipx", "install", julesLocalRepo)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// installWithPip uses `pip install --user` as a last resort.
func installWithPip() error {
	// Try pip commands in platform order.
	pipCmds := []string{"pip3", "pip", "python3 -m pip", "python -m pip", "py -m pip"}
	for _, pipCmd := range pipCmds {
		parts := strings.Fields(pipCmd)
		if _, err := exec.LookPath(parts[0]); err != nil {
			continue
		}
		args := append(parts[1:], "install", "--user", julesLocalRepo)
		cmd := exec.Command(parts[0], args...)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		_ = out // try next pip variant
	}
	return fmt.Errorf("no working pip found")
}

// JulesLocalVersion returns the installed version, or empty string if not installed.
func JulesLocalVersion() string {
	out, err := exec.Command("jules-local", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
