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

// mcpPayloadWithLocal is a superset of mcpPayload that captures the Tier 1
// local-tools-mcp opt-in shape (stdio command + args + env).
type mcpPayloadWithLocal struct {
	MCPServers map[string]struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
	} `json:"mcpServers"`
}

func TestWriteMCPConfigForTier_Tier1_WithLocalTools(t *testing.T) {
	vault := t.TempDir()
	path, err := WriteMCPConfigForTier(
		config.TierFull, vault, "dck_t1_with_local", "https://mcp.test/sse",
		MCPWriteOptions{LocalToolsMCP: true},
	)
	if err != nil {
		t.Fatalf("WriteMCPConfigForTier: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .mcp.json: %v", err)
	}
	var got mcpPayloadWithLocal
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse .mcp.json: %v", err)
	}
	if len(got.MCPServers) != 2 {
		t.Fatalf("expected 2 mcpServers entries, got %d (%v)", len(got.MCPServers), got.MCPServers)
	}

	jules, ok := got.MCPServers["jules"]
	if !ok {
		t.Fatal("expected mcpServers.jules entry")
	}
	if jules.URL != "https://mcp.test/sse" {
		t.Errorf("jules.url = %q, want https://mcp.test/sse", jules.URL)
	}
	if jules.Headers["X-API-Key"] != "dck_t1_with_local" {
		t.Errorf("jules X-API-Key missing or wrong: %q", jules.Headers["X-API-Key"])
	}

	local, ok := got.MCPServers["jules-local"]
	if !ok {
		t.Fatal("expected mcpServers[jules-local] entry when LocalToolsMCP=true")
	}
	if local.Command != "jules-local" {
		t.Errorf("jules-local.command = %q, want jules-local", local.Command)
	}
	if len(local.Args) != 3 || local.Args[0] != "mcp" || local.Args[1] != "--vault" || local.Args[2] != "." {
		t.Errorf("jules-local.args = %v, want [mcp --vault .]", local.Args)
	}
	if local.Env["JULES_CONFIG"] == "" {
		t.Error("jules-local.env.JULES_CONFIG must be set")
	}
	// jules-local entry must NOT contain an embedded API key — stdio bridge
	// reads config.toml at runtime.
	if local.Headers != nil && local.Headers["X-API-Key"] != "" {
		t.Error("jules-local entry should not have headers.X-API-Key — it's stdio, not SSE")
	}
}

func TestWriteMCPConfigForTier_Tier2_IgnoresLocalToolsFlag(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}

	// Tier 2 must NOT write a jules-local entry even if the flag is set —
	// stdio bridges are meaningless without a vault to cd into.
	path, err := WriteMCPConfigForTier(
		config.TierRemote, "", "dck_t2_local_ignored", "",
		MCPWriteOptions{LocalToolsMCP: true},
	)
	if err != nil {
		t.Fatalf("WriteMCPConfigForTier: %v", err)
	}
	data, _ := os.ReadFile(path)
	var got mcpPayloadWithLocal
	_ = json.Unmarshal(data, &got)
	if len(got.MCPServers) != 1 {
		t.Errorf("Tier 2 should have exactly one server even with LocalToolsMCP=true; got %d", len(got.MCPServers))
	}
	if _, has := got.MCPServers["jules-local"]; has {
		t.Error("Tier 2 must never write a jules-local stdio entry")
	}
}

// Default behaviour: LocalToolsMCP not passed -> Tier 1 still writes just the
// remote SSE entry (single server). This guards against the v0.3.0 ship
// behaviour regressing into always-on bridge registration.
func TestWriteMCPConfigForTier_Tier1_DefaultNoLocalTools(t *testing.T) {
	vault := t.TempDir()
	path, err := WriteMCPConfigForTier(config.TierFull, vault, "dck_default", "")
	if err != nil {
		t.Fatalf("WriteMCPConfigForTier: %v", err)
	}
	data, _ := os.ReadFile(path)
	var got mcpPayloadWithLocal
	_ = json.Unmarshal(data, &got)
	if len(got.MCPServers) != 1 {
		t.Errorf("default Tier 1 must have exactly one server, got %d", len(got.MCPServers))
	}
	if _, has := got.MCPServers["jules-local"]; has {
		t.Error("default Tier 1 must not register jules-local stdio bridge")
	}
}
