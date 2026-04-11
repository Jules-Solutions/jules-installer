// Package setup — launch.go handles launching Claude Code in the vault directory.
package setup

import (
	"fmt"
	"os/exec"
	"runtime"
)

// LaunchClaudeCode opens a new terminal window with Claude Code running in vaultPath.
// Returns an error if the launch fails, in which case the caller should display
// the manual command as a fallback.
func LaunchClaudeCode(vaultPath string) error {
	switch runtime.GOOS {
	case "windows":
		return launchWindows(vaultPath)
	case "darwin":
		return launchMacOS(vaultPath)
	default:
		return launchLinux(vaultPath)
	}
}

// ManualLaunchInstructions returns the instructions the user should follow
// if the automatic launch failed.
func ManualLaunchInstructions(vaultPath string) string {
	return fmt.Sprintf("To start Claude Code manually:\n  cd %s\n  claude", vaultPath)
}

// launchWindows opens a new cmd.exe window, changes to vaultPath, and runs claude.
func launchWindows(vaultPath string) error {
	// cmd /c start cmd /k "cd /d <path> && claude"
	// /k keeps the window open after claude exits.
	arg := fmt.Sprintf(`cd /d "%s" && claude`, vaultPath)
	cmd := exec.Command("cmd", "/c", "start", "cmd", "/k", arg)
	return cmd.Start()
}

// launchMacOS opens the default Terminal.app, changes to vaultPath, and runs claude.
func launchMacOS(vaultPath string) error {
	// Use osascript to tell Terminal to open a new window running our command.
	script := fmt.Sprintf(
		`tell application "Terminal" to do script "cd %q && claude"`,
		vaultPath,
	)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Start(); err != nil {
		// Fallback: try open -a Terminal (opens Terminal at default dir, no command).
		return exec.Command("open", "-a", "Terminal", vaultPath).Start()
	}
	return nil
}

// launchLinux tries common terminal emulators in preference order.
// Each is tried with an -e flag (or equivalent) to run claude in the vault dir.
func launchLinux(vaultPath string) error {
	type termSpec struct {
		bin  string
		args []string
	}

	// Build the shell command: cd into vault and run claude.
	shellCmd := fmt.Sprintf("cd %q && claude", vaultPath)

	terminals := []termSpec{
		// GNOME Terminal
		{"gnome-terminal", []string{"--working-directory=" + vaultPath, "--", "bash", "-c", shellCmd + "; exec bash"}},
		// KDE Konsole
		{"konsole", []string{"--workdir", vaultPath, "-e", "bash", "-c", shellCmd + "; exec bash"}},
		// xfce4-terminal
		{"xfce4-terminal", []string{"--working-directory=" + vaultPath, "-e", "bash -c '" + shellCmd + "; exec bash'"}},
		// xterm (widely available fallback)
		{"xterm", []string{"-e", "bash", "-c", shellCmd + "; exec bash"}},
		// kitty
		{"kitty", []string{"bash", "-c", shellCmd + "; exec bash"}},
		// alacritty
		{"alacritty", []string{"--working-directory", vaultPath, "-e", "bash", "-c", shellCmd + "; exec bash"}},
		// wezterm
		{"wezterm", []string{"start", "--cwd", vaultPath, "bash", "-c", shellCmd}},
	}

	for _, t := range terminals {
		if _, err := exec.LookPath(t.bin); err != nil {
			continue // not installed
		}
		cmd := exec.Command(t.bin, t.args...)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no supported terminal emulator found (tried gnome-terminal, konsole, xfce4-terminal, xterm, kitty, alacritty, wezterm)")
}
