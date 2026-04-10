//go:build windows

package audit

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// CheckDisk checks available disk space on the home directory's drive.
func CheckDisk() Check {
	path := diskCheckPath()

	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return Check{Name: "Disk Space", Status: StatusWarn, Detail: "could not check: " + err.Error()}
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(
		pathPtr,
		(*uint64)(unsafe.Pointer(&freeBytesAvailable)),
		(*uint64)(unsafe.Pointer(&totalBytes)),
		(*uint64)(unsafe.Pointer(&totalFreeBytes)),
	)
	if err != nil {
		return Check{Name: "Disk Space", Status: StatusWarn, Detail: "could not check: " + err.Error()}
	}

	freeStr := formatBytes(freeBytesAvailable)

	const oneGB = 1024 * 1024 * 1024
	if freeBytesAvailable < oneGB {
		return Check{Name: "Disk Space", Status: StatusWarn, Version: freeStr, Detail: "< 1 GB free — may not be enough"}
	}

	return Check{Name: "Disk Space", Status: StatusPass, Version: freeStr + " free"}
}
