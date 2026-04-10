// Package audit — ssh.go checks for SSH keys and GitHub connectivity.
package audit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CheckSSH checks for SSH keys and GitHub connectivity.
func CheckSSH() Check {
	home, err := os.UserHomeDir()
	if err != nil {
		return Check{Name: "SSH Keys", Status: StatusWarn, Detail: "could not determine home directory"}
	}

	sshDir := filepath.Join(home, ".ssh")

	// Find SSH key files.
	keyTypes := []string{"id_ed25519", "id_rsa", "id_ecdsa"}
	var foundKey string
	for _, kt := range keyTypes {
		if _, err := os.Stat(filepath.Join(sshDir, kt)); err == nil {
			foundKey = kt
			break
		}
	}

	if foundKey == "" {
		return Check{
			Name:   "SSH Keys",
			Status: StatusWarn,
			Detail: "no keys found — needed for vault sync (ssh-keygen -t ed25519)",
		}
	}

	// Check GitHub connectivity.
	// ssh -T git@github.com exits with 1 on success ("Hi username! You've successfully authenticated")
	// and 255 on failure.
	ghStatus := "not tested"
	out, err := exec.Command("ssh", "-T", "-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=5", "git@github.com").CombinedOutput()
	outStr := string(out)
	if strings.Contains(outStr, "successfully authenticated") {
		ghStatus = "github.com ✓"
	} else if err != nil {
		ghStatus = "github.com unreachable"
	}

	detail := foundKey + " → " + ghStatus
	status := StatusPass
	if strings.Contains(ghStatus, "unreachable") {
		status = StatusWarn
	}

	return Check{Name: "SSH Keys", Status: status, Detail: detail}
}
