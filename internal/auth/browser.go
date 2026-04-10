// Package auth — browser.go implements the browser callback auth flow.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// BrowserResult is returned when the browser callback flow completes.
type BrowserResult struct {
	APIKey string
}

// browserFlow starts a local HTTP server, opens the user's browser to the auth
// URL, and waits for the callback containing the API key.
//
// authURL is the base URL of the auth service (e.g. https://auth.jules.solutions).
func browserFlow(authURL string) (string, error) {
	// Generate a cryptographically random state parameter to prevent CSRF.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Bind to a random available port on localhost.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("starting callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	// Handle the OAuth callback.
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// Validate state parameter.
		if q.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- fmt.Errorf("state parameter mismatch — possible CSRF attack")
			return
		}

		apiKey := q.Get("api_key")
		if apiKey == "" {
			http.Error(w, "missing api_key", http.StatusBadRequest)
			errCh <- fmt.Errorf("callback did not include api_key parameter")
			return
		}

		// Send a friendly success page to the browser.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, successHTML)

		resultCh <- apiKey
	})

	// Start the server in the background.
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			// Ignore "use of closed network connection" which is expected on shutdown.
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Build the full authorization URL.
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	authorizationURL := fmt.Sprintf(
		"%s/installer/authorize?redirect_uri=%s&state=%s",
		authURL,
		redirectURI,
		state,
	)

	// Open the URL in the default browser.
	if err := openBrowser(authorizationURL); err != nil {
		_ = srv.Shutdown(context.Background())
		return "", fmt.Errorf("opening browser: %w", err)
	}

	// Wait for callback or timeout.
	timeout := time.After(120 * time.Second)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	select {
	case apiKey := <-resultCh:
		return apiKey, nil
	case err := <-errCh:
		return "", err
	case <-timeout:
		return "", fmt.Errorf("browser auth timed out after 120 seconds")
	}
}

// openBrowser opens url in the system default browser using the appropriate
// platform-specific command.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux and others
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// successHTML is the page shown in the browser after successful authentication.
const successHTML = `<!DOCTYPE html>
<html>
<head>
  <title>Jules.Solutions — Authenticated</title>
  <style>
    body { font-family: -apple-system, sans-serif; background: #0F0F23; color: #E2E8F0;
           display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
    .card { text-align: center; padding: 2rem; }
    h1 { color: #22C55E; font-size: 2rem; margin-bottom: 0.5rem; }
    p { color: #64748B; }
  </style>
</head>
<body>
  <div class="card">
    <h1>&#x2713; Authenticated</h1>
    <p>You can close this tab and return to the installer.</p>
  </div>
</body>
</html>`
