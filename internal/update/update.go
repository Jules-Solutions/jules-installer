// Package update handles self-update checking against GitHub releases.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const releasesURL = "https://api.github.com/repos/Jules-Solutions/jules-installer/releases/latest"

// UpdateInfo describes the result of an update check.
type UpdateInfo struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	DownloadURL    string
}

// ReleaseInfo contains metadata about the latest GitHub release.
type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// CheckForUpdate queries GitHub releases and compares against currentVersion.
// Returns UpdateInfo indicating whether a newer version is available.
// Network errors are handled gracefully — on timeout or connectivity issue,
// the function returns (nil, nil) so startup is never blocked.
func CheckForUpdate(currentVersion string) (*UpdateInfo, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		// Network errors are non-fatal — skip silently.
		return nil, nil //nolint:nilerr
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	// If current version is "dev" or otherwise unparseable, skip the comparison.
	if current == "dev" || current == "" {
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  release.TagName,
			DownloadURL:    release.HTMLURL,
		}, nil
	}

	available, err := isNewer(latest, current)
	if err != nil {
		// Unparseable version tag — not an error worth surfacing.
		return nil, nil //nolint:nilerr
	}

	return &UpdateInfo{
		Available:      available,
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		DownloadURL:    release.HTMLURL,
	}, nil
}

// FormatUpdateMessage returns the one-liner shown at startup when an update is available.
func FormatUpdateMessage(info *UpdateInfo) string {
	if info == nil || !info.Available {
		return ""
	}
	return fmt.Sprintf(
		"Update available: %s (current: %s) — download at %s",
		info.LatestVersion,
		info.CurrentVersion,
		info.DownloadURL,
	)
}

// fetchLatestRelease hits the GitHub API to get the latest release metadata.
func fetchLatestRelease() (*ReleaseInfo, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, releasesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "jules-installer")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decoding release info: %w", err)
	}

	return &info, nil
}

// isNewer returns true if candidate is a newer semantic version than current.
// Both inputs should be stripped of their "v" prefix before calling.
// Supports versions of the form MAJOR, MAJOR.MINOR, or MAJOR.MINOR.PATCH.
func isNewer(candidate, current string) (bool, error) {
	cv, err := parseSemver(candidate)
	if err != nil {
		return false, fmt.Errorf("parsing candidate version %q: %w", candidate, err)
	}
	cu, err := parseSemver(current)
	if err != nil {
		return false, fmt.Errorf("parsing current version %q: %w", current, err)
	}

	// Compare major, minor, patch in order.
	for i := 0; i < 3; i++ {
		if cv[i] > cu[i] {
			return true, nil
		}
		if cv[i] < cu[i] {
			return false, nil
		}
	}
	// Versions are equal.
	return false, nil
}

// parseSemver parses a version string (without "v" prefix) into a [3]int of
// [major, minor, patch]. Missing components default to 0.
func parseSemver(v string) ([3]int, error) {
	var result [3]int
	// Strip any pre-release suffix (e.g. "1.2.3-beta.1" → "1.2.3").
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.SplitN(v, ".", 3)
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return result, fmt.Errorf("non-numeric version component %q", p)
		}
		result[i] = n
	}
	return result, nil
}
