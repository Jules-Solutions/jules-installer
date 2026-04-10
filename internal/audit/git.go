// Package audit — git.go checks for a usable git installation.
package audit

// CheckGit verifies that git is installed and returns its version.
//
// TODO(Phase 2): implement real check using:
//
//	exec.Command("git", "--version").Output()
//
// Parse output like "git version 2.44.0" and warn if < 2.30 (sparse checkout
// support needed for vault partial clones).
func CheckGit() Check {
	return Check{
		Name:   "Git",
		Status: StatusSkip,
		Detail: "TODO: check git --version and validate minimum version 2.30",
	}
}
