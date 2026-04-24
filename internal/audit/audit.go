// Package audit detects the user's development environment before installation.
package audit

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Status values for a Check result.
const (
	StatusPass = "pass"
	StatusFail = "fail"
	StatusWarn = "warn"
	StatusSkip = "skip"
)

// Check holds the result of a single environment check.
type Check struct {
	Name    string // Human-readable check name (e.g. "Git")
	Status  string // One of: pass, fail, warn, skip
	Version string // Detected version string, empty if not found
	Detail  string // Additional detail or error message
}

// Installable returns true if there's an install function for this check
// and it needs installing (status is fail or warn).
func (c Check) Installable() bool {
	if c.Status == StatusPass || c.Status == StatusSkip {
		return false
	}
	_, ok := installers[c.Name]
	return ok
}

// Tier-aware severity.
//
// Tier 1 (full local install) treats Python/uv/Docker as critical — without
// them, jules-local won't install and the full runtime won't work.
//
// Tier 2 (remote MCP only) only needs Claude Code to be functional; Python,
// Docker, Node, Git, SSH, and editors are all nice-to-have. A missing tool
// should be visible in the audit but NOT flagged as a hard fail, and we
// shouldn't offer to auto-install it for a user who explicitly chose the
// minimal path.

// tier2Critical is the set of check names that a Tier 2 user genuinely needs.
// Anything not in this set gets demoted from fail → warn for Tier 2 display.
var tier2Critical = map[string]bool{
	"Platform":    true, // metadata, always pass
	"Claude Code": true, // THE consumer of the MCP config we're about to write
	"Disk Space":  true, // sanity check — ~/.claude/.mcp.json still needs to be writable
}

// StatusForTier returns the effective status for display purposes, adjusted
// for the onboarding tier. Pass Tier == "" to fall back to raw status.
func (c Check) StatusForTier(tier string) string {
	// Tier 1 or unset → raw status
	if tier != "tier2" {
		return c.Status
	}
	// Tier 2 — non-critical fails become warns.
	if c.Status == StatusFail && !tier2Critical[c.Name] {
		return StatusWarn
	}
	return c.Status
}

// InstallableForTier returns true only if the check is (a) installable AND
// (b) matters for the given tier. Tier 2 users don't get offered a Python
// install they don't need.
func (c Check) InstallableForTier(tier string) bool {
	if !c.Installable() {
		return false
	}
	if tier == "tier2" && !tier2Critical[c.Name] {
		return false
	}
	return true
}

// CountInstallableForTier counts checks that should be offered for auto-install
// given the chosen tier.
func CountInstallableForTier(checks []Check, tier string) int {
	n := 0
	for _, c := range checks {
		if c.InstallableForTier(tier) {
			n++
		}
	}
	return n
}

// InstallMissingForTier runs installers only for checks that are tier-relevant.
// For Tier 2 this is a narrow set (effectively just Claude Code).
func InstallMissingForTier(checks []Check, tier string) []InstallResult {
	var results []InstallResult
	for _, c := range checks {
		if !c.InstallableForTier(tier) {
			continue
		}
		fn := installers[c.Name]
		err := fn()
		if err != nil {
			results = append(results, InstallResult{Name: c.Name, Success: false, Detail: err.Error()})
		} else {
			results = append(results, InstallResult{Name: c.Name, Success: true, Detail: "installed"})
		}
	}
	return results
}

// RunAudit runs all environment checks concurrently and returns their results
// in a fixed display order.
func RunAudit() []Check {
	type indexedCheck struct {
		idx   int
		check Check
	}

	fns := []func() Check{
		CheckPlatform, // 0
		CheckGit,      // 1
		CheckDocker,   // 2
		CheckPython,   // 3
		CheckNode,     // 4
		CheckClaude,   // 5
		CheckEditors,  // 6
		CheckSSH,      // 7
		CheckDisk,     // 8
	}

	ch := make(chan indexedCheck, len(fns))
	for i, fn := range fns {
		go func(idx int, f func() Check) {
			ch <- indexedCheck{idx, f()}
		}(i, fn)
	}

	checks := make([]Check, len(fns))
	for range fns {
		r := <-ch
		checks[r.idx] = r.check
	}
	return checks
}

// CountInstallable returns how many checks can be auto-installed.
func CountInstallable(checks []Check) int {
	n := 0
	for _, c := range checks {
		if c.Installable() {
			n++
		}
	}
	return n
}

// InstallResult holds the outcome of an install attempt.
type InstallResult struct {
	Name    string
	Success bool
	Detail  string
}

// InstallMissing runs the installer for every check that needs it.
// Returns results in the same order.
func InstallMissing(checks []Check) []InstallResult {
	var results []InstallResult
	for _, c := range checks {
		if !c.Installable() {
			continue
		}
		fn := installers[c.Name]
		err := fn()
		if err != nil {
			results = append(results, InstallResult{Name: c.Name, Success: false, Detail: err.Error()})
		} else {
			results = append(results, InstallResult{Name: c.Name, Success: true, Detail: "installed"})
		}
	}
	return results
}

// --- Installer registry ---
// Each audit file registers its install function here.

var installers = map[string]func() error{
	"Git":         installGit,
	"Docker":      installDocker,
	"Python":      installPython,
	"Node.js":     installNode,
	"Claude Code": installClaude,
	"Editors":     installEditors,
	"SSH Keys":    installSSHKeys,
}

// --- Platform install helpers ---

// runInstallCmd runs a command and returns a friendly error if it fails.
func runInstallCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", name, err, string(out))
	}
	return nil
}

// hasCommand checks if a command is available on PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// --- Per-tool install functions ---

func installGit() error {
	switch runtime.GOOS {
	case "windows":
		return runInstallCmd("winget", "install", "--id", "Git.Git", "-e", "--accept-source-agreements", "--accept-package-agreements")
	case "darwin":
		if hasCommand("brew") {
			return runInstallCmd("brew", "install", "git")
		}
		// xcode-select --install triggers git install on macOS
		return runInstallCmd("xcode-select", "--install")
	default:
		if hasCommand("apt-get") {
			return runInstallCmd("sudo", "apt-get", "install", "-y", "git")
		}
		if hasCommand("dnf") {
			return runInstallCmd("sudo", "dnf", "install", "-y", "git")
		}
		return fmt.Errorf("install git manually: https://git-scm.com")
	}
}

func installDocker() error {
	switch runtime.GOOS {
	case "windows":
		return runInstallCmd("winget", "install", "--id", "Docker.DockerDesktop", "-e", "--accept-source-agreements", "--accept-package-agreements")
	case "darwin":
		if hasCommand("brew") {
			return runInstallCmd("brew", "install", "--cask", "docker")
		}
		return fmt.Errorf("install Docker Desktop: https://docker.com/get-started")
	default:
		// Docker's official install script
		return fmt.Errorf("install Docker: https://docs.docker.com/engine/install/")
	}
}

func installPython() error {
	switch runtime.GOOS {
	case "windows":
		// Install Python, then uv
		if err := runInstallCmd("winget", "install", "--id", "Python.Python.3.13", "-e", "--accept-source-agreements", "--accept-package-agreements"); err != nil {
			return err
		}
		// Install uv
		return runInstallCmd("powershell", "-Command", "irm https://astral.sh/uv/install.ps1 | iex")
	case "darwin":
		if hasCommand("brew") {
			if err := runInstallCmd("brew", "install", "python"); err != nil {
				return err
			}
		}
		// Install uv via curl
		return runInstallCmd("sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh")
	default:
		if hasCommand("apt-get") {
			if err := runInstallCmd("sudo", "apt-get", "install", "-y", "python3", "python3-pip"); err != nil {
				return err
			}
		}
		// Install uv
		return runInstallCmd("sh", "-c", "curl -LsSf https://astral.sh/uv/install.sh | sh")
	}
}

func installNode() error {
	switch runtime.GOOS {
	case "windows":
		if err := runInstallCmd("winget", "install", "--id", "OpenJS.NodeJS.LTS", "-e", "--accept-source-agreements", "--accept-package-agreements"); err != nil {
			return err
		}
		// pnpm via corepack
		return runInstallCmd("corepack", "enable")
	case "darwin":
		if hasCommand("brew") {
			if err := runInstallCmd("brew", "install", "node"); err != nil {
				return err
			}
			return runInstallCmd("corepack", "enable")
		}
		return fmt.Errorf("install Node.js: https://nodejs.org")
	default:
		if hasCommand("apt-get") {
			if err := runInstallCmd("sudo", "apt-get", "install", "-y", "nodejs", "npm"); err != nil {
				return err
			}
			return runInstallCmd("corepack", "enable")
		}
		return fmt.Errorf("install Node.js: https://nodejs.org")
	}
}

func installClaude() error {
	// Claude Code is installed via npm globally.
	if !hasCommand("npm") {
		return fmt.Errorf("npm required — install Node.js first, then: npm install -g @anthropic-ai/claude-code")
	}
	return runInstallCmd("npm", "install", "-g", "@anthropic-ai/claude-code")
}

func installEditors() error {
	// Install Obsidian (recommended) + VS Code.
	var lastErr error
	switch runtime.GOOS {
	case "windows":
		// Try both — either failing is not fatal.
		if err := runInstallCmd("winget", "install", "--id", "Obsidian.Obsidian", "-e", "--accept-source-agreements", "--accept-package-agreements"); err != nil {
			lastErr = err
		}
		if err := runInstallCmd("winget", "install", "--id", "Microsoft.VisualStudioCode", "-e", "--accept-source-agreements", "--accept-package-agreements"); err != nil {
			lastErr = err
		}
	case "darwin":
		if hasCommand("brew") {
			if err := runInstallCmd("brew", "install", "--cask", "obsidian"); err != nil {
				lastErr = err
			}
			if err := runInstallCmd("brew", "install", "--cask", "visual-studio-code"); err != nil {
				lastErr = err
			}
		} else {
			return fmt.Errorf("install Homebrew first: https://brew.sh")
		}
	default:
		return fmt.Errorf("install Obsidian: https://obsidian.md — VS Code: https://code.visualstudio.com")
	}
	return lastErr
}

func installSSHKeys() error {
	// Generate an ed25519 key with no passphrase (user can add one later).
	if hasCommand("ssh-keygen") {
		home, _ := exec.Command("sh", "-c", "echo $HOME").Output()
		if runtime.GOOS == "windows" {
			// Windows ssh-keygen
			return runInstallCmd("ssh-keygen", "-t", "ed25519", "-N", "", "-f",
				fmt.Sprintf("%s\\.ssh\\id_ed25519", getHomeDir()))
		}
		_ = home
		return runInstallCmd("ssh-keygen", "-t", "ed25519", "-N", "", "-f",
			fmt.Sprintf("%s/.ssh/id_ed25519", getHomeDir()))
	}
	return fmt.Errorf("ssh-keygen not found — install OpenSSH")
}

func getHomeDir() string {
	if h, err := exec.Command("sh", "-c", "echo $HOME").Output(); err == nil {
		s := string(h)
		if len(s) > 0 && s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}
		return s
	}
	return "~"
}
