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
