// Package config manages the jules-installer configuration file at ~/.config/jules/config.toml.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

// Tier identifies which onboarding path the user picked.
//
//   - TierFull (tier1): local vault + jules-local CLI installed + .mcp.json in vault root
//   - TierRemote (tier2): remote MCP only, no vault, .mcp.json in user-global CC config
type Tier string

const (
	// TierFull is the full local install: vault scaffold/clone + jules-local + vault-root .mcp.json.
	TierFull Tier = "tier1"
	// TierRemote is the MCP-only remote install: just writes a CC-global .mcp.json pointing at the SSE endpoint.
	TierRemote Tier = "tier2"
)

// Valid returns true if the Tier value is a recognised onboarding tier.
func (t Tier) Valid() bool {
	return t == TierFull || t == TierRemote
}

// Config is the top-level configuration structure persisted to config.toml.
type Config struct {
	Auth  AuthConfig  `toml:"auth"`
	Local LocalConfig `toml:"local"`
}

// AuthConfig holds authentication credentials and service endpoints.
type AuthConfig struct {
	APIKey  string `toml:"api_key"`
	APIURL  string `toml:"api_url"`
	AuthURL string `toml:"auth_url"`
	// MCPURL is the remote MCP SSE endpoint (e.g. "https://mcp.jules.solutions/sse").
	// Stored in config so re-runs know where to point generated .mcp.json files.
	MCPURL string `toml:"mcp_url"`
}

// LocalConfig holds paths and local machine settings.
type LocalConfig struct {
	// Tier is the chosen onboarding path ("tier1" | "tier2"). Empty on fresh installs
	// prior to the tier-pick screen, populated immediately after the user chooses.
	Tier       Tier   `toml:"tier"`
	VaultPath  string `toml:"vault_path"`
	Shell      string `toml:"shell"`
	InstallDir string `toml:"install_dir"`
	// MCPPath records where the installer wrote .mcp.json for re-run detection.
	// Tier 1: vault root. Tier 2: ~/.claude/.mcp.json (user-global).
	MCPPath string `toml:"mcp_path"`
	// LocalToolsMCP, when true on Tier 1, adds a second "jules-local" MCP server
	// (stdio bridge to the installed jules-local CLI) to .mcp.json alongside the
	// remote "jules" SSE entry. Exposes local-only tools — exec_manage,
	// file_manage, terminal_spawn, git_manage — to Claude Code. Default false
	// matches the v0.3.0 ship behaviour. Tier 2 ignores this field.
	LocalToolsMCP bool `toml:"local_tools_mcp"`
}

// ConfigDir returns the path to the jules config directory.
// Respects XDG_CONFIG_HOME on Linux, falls back to ~/.config/jules.
func ConfigDir() (string, error) {
	if runtime.GOOS == "linux" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "jules"), nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	return filepath.Join(home, ".config", "jules"), nil
}

// ConfigPath returns the full path to config.toml.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// LoadConfig reads and parses the config file. Returns a zero-value Config
// (not an error) if the file does not exist yet.
func LoadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	var cfg Config

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig writes cfg to disk, creating the config directory if needed.
func SaveConfig(cfg Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("opening config file for write: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return nil
}

// DefaultConfig returns a Config with sensible defaults.
// Tier is intentionally left empty — the installer must force the user to pick
// one (via TUI screen or --tier flag) before writing this.
func DefaultConfig() Config {
	return Config{
		Auth: AuthConfig{
			APIURL:  "https://api.jules.solutions",
			AuthURL: "https://auth.jules.solutions",
			MCPURL:  "https://mcp.jules.solutions/sse",
		},
		Local: LocalConfig{
			VaultPath: DefaultVaultPath(),
		},
	}
}

// DefaultTier2MCPPath returns the user-global Claude Code MCP config path.
// On all platforms this is ~/.claude/.mcp.json — Claude Code reads this file
// in every CC session regardless of cwd.
func DefaultTier2MCPPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".claude", ".mcp.json"), nil
}

// DefaultVaultPath returns the recommended vault location: ~/{Username}.Life
// Uses the OS username (last component of the home directory path).
func DefaultVaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "My.Life"
	}
	username := filepath.Base(home) // e.g. "felix" from /home/felix or C:\Users\felix
	// Capitalize first letter for the directory name.
	if len(username) > 0 {
		username = strings.ToUpper(username[:1]) + username[1:]
	}
	return filepath.Join(home, username+".Life")
}
