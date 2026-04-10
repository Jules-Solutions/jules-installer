// Package audit — disk.go provides shared disk check helpers.
// Platform-specific CheckDisk() implementations are in disk_unix.go and disk_windows.go.
package audit

import (
	"fmt"
	"os"
)

// formatBytes returns a human-readable size string.
func formatBytes(b uint64) string {
	const gb = 1024 * 1024 * 1024
	if b >= gb {
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	}
	const mb = 1024 * 1024
	return fmt.Sprintf("%.0f MB", float64(b)/float64(mb))
}

// diskCheckPath returns the path to check disk space on.
func diskCheckPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}
