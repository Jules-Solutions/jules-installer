// Package setup tests for the tier-aware MCP config writer.
package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Jules-Solutions/jules-installer/internal/config"
)

// mcpPayload is the minimum shape we assert in tests.
type mcpPayload struct {
	MCPServers map[string]struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	} `json:"mcpServers"`
}

func TestWriteMCPConfigForTier_Tier1(t *testing.T) {
	vault := t.TempDir()
	apiKey := "dck_t1_sample_key"

	path, err := WriteMCPConfigForTier(config.TierFull, vault, apiKey, "https://mcp.test/sse")
	if err != nil {
		t.Fatalf("WriteMCPConfigForTier: %v", err)
	}

	wantPath := filepath.Join(vault, ".mcp.json")
	if path != wantPath {
		t.Errorf("tier1 mcp path = %q, want %q", path, wantPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .mcp.json: %v", err)
	}
	var got mcpPayload
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse .mcp.json: %v", err)
	}
	srv, ok := got.MCPServers["jules"]
	if !ok {
		t.Fatal("expected mcpServers.jules entry")
	}
	if srv.URL != "https://mcp.test/sse" {
		t.Errorf("url = %q, want https://mcp.test/sse", srv.URL)
	}
	if srv.Headers["X-API-Key"] != apiKey {
		t.Errorf("X-API-Key = %q, want %q", srv.Headers["X-API-Key"], apiKey)
	}

	// Tier 1 also drops a .claude/settings.json stub when missing.
	settingsPath := filepath.Join(vault, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Errorf("tier1 should create .claude/settings.json: %v", err)
	}
}

func TestWriteMCPConfigForTier_Tier2(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}

	apiKey := "dck_t2_sample_key"
	path, err := WriteMCPConfigForTier(config.TierRemote, "", apiKey, "")
	if err != nil {
		t.Fatalf("WriteMCPConfigForTier: %v", err)
	}

	// Must land at ~/.claude/.mcp.json
	wantPath := filepath.Join(home, ".claude", ".mcp.json")
	if path != wantPath {
		t.Errorf("tier2 mcp path = %q, want %q", path, wantPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tier2 .mcp.json: %v", err)
	}
	var got mcpPayload
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse tier2 .mcp.json: %v", err)
	}
	srv, ok := got.MCPServers["jules"]
	if !ok {
		t.Fatal("expected mcpServers.jules entry in tier2 config")
	}
	// Default URL when empty mcpURL passed.
	if !strings.HasSuffix(srv.URL, "/sse") {
		t.Errorf("tier2 default URL must end in /sse; got %q", srv.URL)
	}
	if srv.Headers["X-API-Key"] != apiKey {
		t.Errorf("tier2 X-API-Key = %q, want %q", srv.Headers["X-API-Key"], apiKey)
	}

	// Tier 2 must NOT create a settings.json (there's no vault).
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		t.Error("tier2 should not create .claude/settings.json (no vault)")
	}
}

func TestWriteMCPConfigForTier_Tier1_MissingVault(t *testing.T) {
	if _, err := WriteMCPConfigForTier(config.TierFull, "", "dck_sample", ""); err == nil {
		t.Error("tier1 must reject empty vault path")
	}
}

func TestWriteMCPConfigForTier_EmptyAPIKey(t *testing.T) {
	vault := t.TempDir()
	if _, err := WriteMCPConfigForTier(config.TierFull, vault, "", ""); err == nil {
		t.Error("must reject empty API key")
	}
}

func TestWriteMCPConfigForTier_UnknownTier(t *testing.T) {
	vault := t.TempDir()
	if _, err := WriteMCPConfigForTier(config.Tier("tier42"), vault, "dck_x", ""); err == nil {
		t.Error("must reject unknown tier")
	}
}

// TestWriteMCPConfigForTier_FileMode verifies the 0600 permission on the written
// file (it holds a credential). Skipped on Windows.
func TestWriteMCPConfigForTier_FileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode doesn't apply on Windows")
	}
	vault := t.TempDir()
	path, err := WriteMCPConfigForTier(config.TierFull, vault, "dck_x", "")
	if err != nil {
		t.Fatalf("WriteMCPConfigForTier: %v", err)
	}
	info, _ := os.Stat(path)
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("mcp file mode = %o, want 0600", mode)
	}
}
