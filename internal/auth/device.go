// Package auth — device.go implements the device code auth flow (RFC 8628 style).
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// deviceCodeResponse is returned by POST /api/auth/device/code.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// deviceTokenResponse is returned by GET /api/auth/device/token.
type deviceTokenResponse struct {
	Status string `json:"status"` // "pending", "complete", "expired"
	APIKey string `json:"api_key"`
	Error  string `json:"error"`
}

// DeviceFlowProgress reports state to the caller during polling.
type DeviceFlowProgress struct {
	UserCode        string
	VerificationURI string
	Elapsed         time.Duration
	Polling         bool
}

// DeviceFlowProgressFunc is called each poll cycle with current status.
type DeviceFlowProgressFunc func(DeviceFlowProgress)

// deviceFlow requests a device code, displays it to the user, and polls until
// the user completes auth in their browser or the code expires.
//
// progressFn is called on each poll so the caller can update the TUI.
// Pass nil to skip progress reporting.
func deviceFlow(authURL string, progressFn DeviceFlowProgressFunc) (string, error) {
	// Step 1: Request a device code.
	dc, err := requestDeviceCode(authURL)
	if err != nil {
		return "", fmt.Errorf("requesting device code: %w", err)
	}

	pollInterval := time.Duration(dc.Interval) * time.Second
	if pollInterval < 5*time.Second {
		pollInterval = 5 * time.Second
	}

	deadline := time.Now().Add(15 * time.Minute)
	start := time.Now()

	// Notify caller of the user code to display.
	if progressFn != nil {
		progressFn(DeviceFlowProgress{
			UserCode:        dc.UserCode,
			VerificationURI: dc.VerificationURI,
			Elapsed:         0,
			Polling:         false,
		})
	}

	// Step 2: Poll for completion.
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("device code expired after 15 minutes")
		}

		if progressFn != nil {
			progressFn(DeviceFlowProgress{
				UserCode:        dc.UserCode,
				VerificationURI: dc.VerificationURI,
				Elapsed:         time.Since(start),
				Polling:         true,
			})
		}

		token, err := pollDeviceToken(authURL, dc.DeviceCode)
		if err != nil {
			return "", fmt.Errorf("polling device token: %w", err)
		}

		switch token.Status {
		case "complete":
			if token.APIKey == "" {
				return "", fmt.Errorf("server returned complete status but no api_key")
			}
			return token.APIKey, nil

		case "expired":
			return "", fmt.Errorf("device code expired on server")

		case "pending":
			// Continue polling.

		default:
			return "", fmt.Errorf("unexpected status from server: %q", token.Status)
		}

		<-ticker.C
	}
}

// requestDeviceCode POSTs to /api/auth/device/code and returns the response.
func requestDeviceCode(authURL string) (*deviceCodeResponse, error) {
	url := strings.TrimRight(authURL, "/") + "/api/auth/device/code"

	resp, err := http.Post(url, "application/json", nil) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("POST device/code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device/code returned HTTP %d", resp.StatusCode)
	}

	var dc deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, fmt.Errorf("decoding device code response: %w", err)
	}

	if dc.DeviceCode == "" || dc.UserCode == "" {
		return nil, fmt.Errorf("server returned incomplete device code response")
	}

	return &dc, nil
}

// pollDeviceToken GETs /api/auth/device/token?device_code=X and returns the status.
func pollDeviceToken(authURL, deviceCode string) (*deviceTokenResponse, error) {
	url := fmt.Sprintf("%s/api/auth/device/token?device_code=%s",
		strings.TrimRight(authURL, "/"), deviceCode)

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("GET device/token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device/token returned HTTP %d", resp.StatusCode)
	}

	var tr deviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	return &tr, nil
}
