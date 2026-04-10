// Package auth — apikey.go implements the manual API key paste flow.
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// apiKeyVerifyResponse is returned by /api/auth/api-key/verify.
type apiKeyVerifyResponse struct {
	Valid  bool   `json:"valid"`
	Error  string `json:"error"`
	UserID string `json:"user_id"`
}

// apiKeyFlow validates a user-provided API key against the auth service.
//
// key must be provided by the caller (typically read from a TUI text input).
// Returns an error if the key is structurally invalid or fails server validation.
func apiKeyFlow(authURL, key string) (string, error) {
	key = strings.TrimSpace(key)

	if !strings.HasPrefix(key, "dck_") {
		return "", fmt.Errorf("API keys must start with dck_ — got %q", truncate(key, 12))
	}

	if err := verifyAPIKey(authURL, key); err != nil {
		return "", fmt.Errorf("key validation failed: %w", err)
	}

	return key, nil
}

// verifyAPIKey sends the key to the auth service for server-side validation.
// The auth service endpoint is POST /api/auth/api-key/verify with JSON body { "key": "dck_..." }.
func verifyAPIKey(authURL, key string) error {
	url := strings.TrimRight(authURL, "/") + "/api/auth/api-key/verify"

	body := fmt.Sprintf(`{"key":%q}`, key)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "jules-installer")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("calling verify endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("key rejected by server (invalid or revoked)")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify endpoint returned HTTP %d", resp.StatusCode)
	}

	var vr apiKeyVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		return fmt.Errorf("decoding verify response: %w", err)
	}

	if !vr.Valid {
		msg := vr.Error
		if msg == "" {
			msg = "key marked invalid by server"
		}
		return fmt.Errorf("%s", msg)
	}

	return nil
}

// truncate returns s shortened to at most n runes, appending "..." if cut.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
