// Package runner tests — unit tests only. Integration tests that invoke
// the real binary live at cmd/jules-setup/integration_test.go (build tag).
package runner

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Jules-Solutions/jules-installer/internal/config"
)

// withIsolatedHome points HOME / USERPROFILE / XDG_CONFIG_HOME at a temp
// directory so the runner's config reads/writes don't touch the user's real
// ~/.config/jules or ~/.claude.
func withIsolatedHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	} else {
		t.Setenv("HOME", dir)
	}
	return dir
}

func TestRun_RequiresTier(t *testing.T) {
	withIsolatedHome(t)
	err := Run(io.Discard, Options{Tier: config.Tier("")})
	if err == nil {
		t.Fatal("runner.Run with no tier must error")
	}
	if !strings.Contains(err.Error(), "Tier") {
		t.Errorf("error should mention Tier, got: %v", err)
	}
}

func TestRun_RequiresAPIKey(t *testing.T) {
	withIsolatedHome(t)
	// Make sure JULES_API_KEY from the developer's environment doesn't leak in.
	t.Setenv("JULES_API_KEY", "")
	err := Run(io.Discard, Options{Tier: config.TierRemote})
	if err == nil {
		t.Fatal("runner.Run without an API key must error")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("error should mention 'API key', got: %v", err)
	}
}

func TestRun_Tier2_Minimal(t *testing.T) {
	home := withIsolatedHome(t)

	// Pre-seed a Tier 2 config like a previously-authenticated user.
	seed := config.DefaultConfig()
	seed.Auth.APIKey = "dck_tier2_runner"
	seed.Local.Tier = config.TierRemote
	if err := config.SaveConfig(seed); err != nil {
		t.Fatalf("seed SaveConfig: %v", err)
	}

	var buf strings.Builder
	if err := Run(&buf, Options{Tier: config.TierRemote}); err != nil {
		t.Fatalf("runner.Run Tier 2: %v", err)
	}

	// Verify the user-global MCP file is there and has the expected shape.
	mcpPath := filepath.Join(home, ".claude", ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("expected MCP file at %s: %v", mcpPath, err)
	}
	var parsed struct {
		MCPServers map[string]struct {
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse MCP file: %v", err)
	}
	jules := parsed.MCPServers["jules"]
	if jules.Headers["X-API-Key"] != "dck_tier2_runner" {
		t.Errorf("MCP file missing or wrong X-API-Key; got headers=%v", jules.Headers)
	}
	if !strings.HasSuffix(jules.URL, "/sse") {
		t.Errorf("MCP url should end /sse, got %q", jules.URL)
	}

	// Config should now record the MCP path.
	post, _ := config.LoadConfig()
	if post.Local.MCPPath != mcpPath {
		t.Errorf("config.local.mcp_path = %q, want %q", post.Local.MCPPath, mcpPath)
	}
}

// TestRun_Tier2ToTier1_Upgrade is the core upgrade-path regression test.
// Seeds a Tier 2 config, runs the runner with Tier=TierFull, asserts the
// post-state matches a freshly-installed Tier 1.
//
// This is the hermetic unit-test equivalent of the scripted binary-level
// integration test at cmd/jules-setup/integration_test.go; they cover
// overlapping ground but catch different classes of regression.
func TestRun_Tier2ToTier1_Upgrade(t *testing.T) {
	if testing.Short() {
		t.Skip("runs git clone / uv install; slow on cold networks")
	}
	home := withIsolatedHome(t)

	// Start as Tier 2.
	seed := config.DefaultConfig()
	seed.Auth.APIKey = "dck_upgrade_test"
	seed.Local.Tier = config.TierRemote
	if err := config.SaveConfig(seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Point vault at a temp location so we don't need write access to HOME/User.Life.
	vault := filepath.Join(home, "TestUser.Life")

	// Upgrade to Tier 1.
	var buf strings.Builder
	err := Run(&buf, Options{Tier: config.TierFull, VaultPath: vault})
	if err != nil {
		// Vault download / jules-local install may fail in hermetic test
		// environments without network. We care that the *config* transition
		// happened and .mcp.json was written; the install-step error is
		// acceptable as long as the tier flipped.
		t.Logf("runner.Run Tier 1 upgrade returned (possibly hermetic failure): %v", err)
	}

	// Reload config — tier must have flipped.
	post, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if post.Local.Tier != config.TierFull {
		t.Errorf("tier should flip to tier1, got %q", post.Local.Tier)
	}
	if post.Local.VaultPath != vault {
		t.Errorf("vault_path = %q, want %q", post.Local.VaultPath, vault)
	}

	// The .mcp.json must now live in the vault, NOT ~/.claude/ (that's Tier 2).
	wantMCP := filepath.Join(vault, ".mcp.json")
	if _, err := os.Stat(wantMCP); err != nil {
		t.Errorf("Tier 1 upgrade should write %s: %v", wantMCP, err)
	}
	if post.Local.MCPPath != wantMCP {
		t.Errorf("config.local.mcp_path = %q, want %q", post.Local.MCPPath, wantMCP)
	}
}
