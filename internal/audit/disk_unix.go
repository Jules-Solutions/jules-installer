//go:build !windows

package audit

import (
	"golang.org/x/sys/unix"
)

// CheckDisk checks available disk space on the home directory's filesystem.
func CheckDisk() Check {
	path := diskCheckPath()

	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return Check{Name: "Disk Space", Status: StatusWarn, Detail: "could not check: " + err.Error()}
	}

	freeBytes := stat.Bavail * uint64(stat.Bsize)
	freeStr := formatBytes(freeBytes)

	const oneGB = 1024 * 1024 * 1024
	if freeBytes < oneGB {
		return Check{Name: "Disk Space", Status: StatusWarn, Version: freeStr, Detail: "< 1 GB free — may not be enough"}
	}

	return Check{Name: "Disk Space", Status: StatusPass, Version: freeStr + " free"}
}
