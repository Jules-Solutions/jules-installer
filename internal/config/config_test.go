// Package config tests for the tier-split additions.
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTier_Valid(t *testing.T) {
	cases := []struct {
		tier Tier
		want bool
	}{
		{TierFull, true},
		{TierRemote, true},
		{Tier(""), false},
		{Tier("tier3"), false},
		{Tier("full"), false}, // only canonical values are valid
	}
	for _, tc := range cases {
		if got := tc.tier.Valid(); got != tc.want {
			t.Errorf("Tier(%q).Valid() = %v, want %v", tc.tier, got, tc.want)
		}
	}
}

// TestConfigRoundTrip ensures a Config with tier and mcp_path fields can be
// saved to disk and loaded back with those fields intact. Protects against
// future refactors that accidentally drop TOML tags.
func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// Override config dir via XDG (Linux) / HOME for cross-platform TempDir use.
	t.Setenv("XDG_CONFIG_HOME", dir)
	if runtime.GOOS == "windows" {
		// Windows doesn't honour XDG; override USERPROFILE so ~/.config resolves
		// inside the temp dir.
		t.Setenv("USERPROFILE", dir)
	} else {
		t.Setenv("HOME", dir)
	}

	original := Config{
		Auth: AuthConfig{
			APIKey:  "dck_test_12345",
			APIURL:  "https://api.example.test",
			AuthURL: "https://auth.example.test",
			MCPURL:  "https://mcp.example.test/sse",
		},
		Local: LocalConfig{
			Tier:      TierRemote,
			VaultPath: "/tmp/test.Life",
			Shell:     "bash",
			MCPPath:   "/home/test/.claude/.mcp.json",
		},
	}

	if err := SaveConfig(original); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Local.Tier != TierRemote {
		t.Errorf("round-trip Tier: got %q, want %q", loaded.Local.Tier, TierRemote)
	}
	if loaded.Local.MCPPath != original.Local.MCPPath {
		t.Errorf("round-trip MCPPath: got %q, want %q", loaded.Local.MCPPath, original.Local.MCPPath)
	}
	if loaded.Auth.MCPURL != original.Auth.MCPURL {
		t.Errorf("round-trip MCPURL: got %q, want %q", loaded.Auth.MCPURL, original.Auth.MCPURL)
	}
}

// TestConfigPath_Mode ensures the written config file has mode 0600 (secrets).
// Skipped on Windows — NTFS doesn't expose POSIX modes the same way.
func TestConfigPath_Mode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode check doesn't apply on Windows")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)

	if err := SaveConfig(DefaultConfig()); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	path, _ := ConfigPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("config file mode: got %o, want 0600", mode)
	}
}

func TestDefaultTier2MCPPath(t *testing.T) {
	home := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	} else {
		t.Setenv("HOME", home)
	}
	got, err := DefaultTier2MCPPath()
	if err != nil {
		t.Fatalf("DefaultTier2MCPPath: %v", err)
	}
	want := filepath.Join(home, ".claude", ".mcp.json")
	if got != want {
		t.Errorf("DefaultTier2MCPPath() = %q, want %q", got, want)
	}
	if !strings.HasSuffix(got, ".mcp.json") {
		t.Errorf("Tier 2 MCP path must end in .mcp.json, got %q", got)
	}
}

// TestDefaultConfig_HasNoTier asserts that a fresh default config does NOT
// pre-select a tier — the installer must force the user to choose.
func TestDefaultConfig_HasNoTier(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Local.Tier != "" {
		t.Errorf("DefaultConfig.Local.Tier must be empty, got %q", cfg.Local.Tier)
	}
	if cfg.Auth.MCPURL == "" {
		t.Error("DefaultConfig.Auth.MCPURL must not be empty (needed by re-run menu)")
	}
	if cfg.Local.LocalToolsMCP {
		t.Error("DefaultConfig must have LocalToolsMCP=false (opt-in)")
	}
}

// TestLocalToolsMCP_RoundTrip is a narrow regression test for the Tier 1
// opt-in flag: if TOML serialisation drops local_tools_mcp, the re-run menu's
// toggle action silently does nothing.
func TestLocalToolsMCP_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	} else {
		t.Setenv("HOME", dir)
	}

	cfg := DefaultConfig()
	cfg.Auth.APIKey = "dck_x"
	cfg.Local.Tier = TierFull
	cfg.Local.LocalToolsMCP = true
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	reloaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !reloaded.Local.LocalToolsMCP {
		t.Error("local_tools_mcp did not survive TOML round-trip")
	}
}
