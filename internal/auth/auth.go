// Package auth implements the three authentication flows for the jules-installer:
//
//  1. Browser callback (primary): open browser → receive api_key via localhost callback
//  2. Device code (fallback): display code → user enters at auth site → poll for key
//  3. API key paste (last resort): user pastes a dck_... key directly
package auth

import "fmt"

// Method identifies which auth flow succeeded.
type Method string

const (
	MethodBrowser  Method = "browser"
	MethodDevice   Method = "device_code"
	MethodAPIKey   Method = "api_key_paste"
	MethodFailed   Method = "failed"
)

// Result is the outcome of a successful authentication.
type Result struct {
	APIKey string
	Method Method
}

// Authenticate runs the auth flows in priority order:
//  1. Browser callback
//  2. Device code
//  3. API key paste
//
// authURL is the base URL of the auth service (e.g. "https://auth.jules.solutions").
// Returns the API key and the method that succeeded.
//
// Note: for TUI-driven flows, prefer calling the Public wrappers below so that
// progress can be displayed incrementally inside Bubbletea Cmd goroutines.
func Authenticate(authURL string) (apiKey string, method Method, err error) {
	// Attempt 1: browser callback.
	key, bErr := browserFlow(authURL)
	if bErr == nil {
		return key, MethodBrowser, nil
	}

	// Attempt 2: device code (no progressFn in headless mode).
	key, dErr := deviceFlow(authURL, nil)
	if dErr == nil {
		return key, MethodDevice, nil
	}

	return "", MethodFailed, fmt.Errorf(
		"all auth methods failed:\n  browser: %v\n  device code: %v",
		bErr, dErr,
	)
}

// BrowserFlowPublic is the exported entry point used by the TUI's Cmd runner.
func BrowserFlowPublic(authURL string) (string, error) {
	return browserFlow(authURL)
}

// DeviceFlowPublic is the exported entry point used by the TUI's Cmd runner.
// progressFn is nil for the Phase 1 single-shot Cmd approach.
func DeviceFlowPublic(authURL string) (string, error) {
	return deviceFlow(authURL, nil)
}

// APIKeyFlowPublic is the exported entry point used by the TUI's Cmd runner.
func APIKeyFlowPublic(authURL, key string) (string, error) {
	return apiKeyFlow(authURL, key)
}
