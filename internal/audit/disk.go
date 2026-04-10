// Package audit — disk.go checks available disk space for the vault download.
package audit

// CheckDisk verifies there is sufficient free disk space for the vault.
//
// TODO(Phase 2): implement real check using:
//
//   - On Unix: syscall.Statfs(path, &stat) → stat.Bavail * uint64(stat.Bsize)
//   - On Windows: windows.GetDiskFreeSpaceEx(path, ...)
//
// Warn if free space < 2 GB. Fail if < 500 MB.
func CheckDisk() Check {
	return Check{
		Name:   "Disk Space",
		Status: StatusSkip,
		Detail: "TODO: check free disk space at vault install path (need ≥ 2 GB free)",
	}
}
