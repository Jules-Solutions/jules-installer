// Package update handles self-update checking against GitHub releases.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const releasesURL = "https://api.github.com/repos/Jules-Solutions/jules-installer/releases/latest"

// ReleaseInfo contains metadata about the latest GitHub release.
type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// Result is the outcome of an update check.
type Result struct {
	Current   string
	Latest    string
	UpdateURL string
	HasUpdate bool
}

// CheckForUpdate queries GitHub releases and compares against currentVersion.
// Returns a Result indicating whether an update is available.
// TODO(Phase 2): implement version comparison and notify the user at startup.
func CheckForUpdate(currentVersion string) (*Result, error) {
	// Stub: always report up to date for now.
	_ = currentVersion
	return &Result{
		Current:   currentVersion,
		Latest:    currentVersion,
		HasUpdate: false,
	}, nil
}

// fetchLatestRelease hits the GitHub API to get the latest release metadata.
// Used by CheckForUpdate when Phase 2 self-update is implemented.
func fetchLatestRelease() (*ReleaseInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

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
