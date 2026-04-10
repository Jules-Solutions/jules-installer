// Package audit detects the user's development environment before installation.
package audit

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

// RunAudit runs all environment checks and returns their results.
// Checks are run in parallel where safe; results are returned in display order.
func RunAudit() []Check {
	// Collect results from all sub-checkers.
	platform := CheckPlatform()

	checks := []Check{
		platform,
		CheckGit(),
		CheckDisk(),
	}

	return checks
}
