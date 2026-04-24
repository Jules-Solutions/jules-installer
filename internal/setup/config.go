// Package setup — config.go writes the MCP config and Claude Code settings.
package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Jules-Solutions/jules-installer/internal/config"
)

// WriteMCPConfigForTier writes the Claude Code MCP server config for the given tier.
//
// Unified shape across tiers (per the 2026-04-24 tier-split design): both tiers
// write a direct-SSE entry with the API key embedded as an X-API-Key header.
// The only difference is the file location:
//
//   - Tier 1 (full install): vaultPath/.mcp.json — active in CC sessions
//     launched from the vault root.
//   - Tier 2 (remote only): ~/.claude/.mcp.json — active in every CC session
//     on this machine regardless of cwd.
//
// For Tier 1 the function also drops a minimal .claude/settings.json into the
// vault if one is missing, matching v0.2.0 behaviour.
//
// mcpURL should be the SSE URL (e.g. "https://mcp.jules.solutions/sse"). Pass
// an empty string to use the production default.
//
// The file is written with mode 0600 because it contains the API key.
// Returns the absolute path of the written file.
func WriteMCPConfigForTier(tier config.Tier, vaultPath, apiKey, mcpURL string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("api key is required to write MCP config")
	}
	if mcpURL == "" {
		mcpURL = "https://mcp.jules.solutions/sse"
	}

	// Unified direct-SSE payload — both tiers use the same shape.
	payload := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"jules": map[string]interface{}{
				"url": mcpURL,
				"headers": map[string]string{
					"X-API-Key": apiKey,
				},
			},
		},
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding MCP config: %w", err)
	}
	data = append(data, '\n')

	// Tier-specific output path.
	var mcpPath string
	switch tier {
	case config.TierFull:
		if vaultPath == "" {
			return "", fmt.Errorf("vault path is required for tier 1 MCP config")
		}
		mcpPath = filepath.Join(vaultPath, ".mcp.json")
		// Ensure vault dir exists (offline scaffold creates this too, but be
		// defensive for re-writes against a manually-moved vault).
		if err := os.MkdirAll(vaultPath, 0o755); err != nil {
			return "", fmt.Errorf("creating vault dir: %w", err)
		}

	case config.TierRemote:
		p, err := config.DefaultTier2MCPPath()
		if err != nil {
			return "", err
		}
		mcpPath = p
		// ~/.claude/ may not exist yet on a fresh machine.
		if err := os.MkdirAll(filepath.Dir(mcpPath), 0o755); err != nil {
			return "", fmt.Errorf("creating ~/.claude dir: %w", err)
		}

	default:
		return "", fmt.Errorf("unknown tier %q", tier)
	}

	// Write with 0600 — the file embeds a credential.
	if err := os.WriteFile(mcpPath, data, 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", mcpPath, err)
	}

	// Tier 1 only: make sure the vault has a minimal .claude/settings.json.
	if tier == config.TierFull {
		settingsPath := filepath.Join(vaultPath, ".claude", "settings.json")
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
			settings := map[string]interface{}{
				"permissions": map[string]interface{}{
					"allow": []string{},
					"deny":  []string{},
				},
			}
			sData, _ := json.MarshalIndent(settings, "", "  ")
			sData = append(sData, '\n')
			_ = os.MkdirAll(filepath.Dir(settingsPath), 0o755)
			_ = os.WriteFile(settingsPath, sData, 0o644)
		}
	}

	return mcpPath, nil
}

// WriteMCPConfig is retained as a backward-compatible wrapper. New code should
// use WriteMCPConfigForTier. Defaults to Tier 1 (vault-rooted .mcp.json) with
// the v0.2.0 jules-local command-bridge shape preserved for legacy callers.
//
// Deprecated: prefer WriteMCPConfigForTier. This function preserves the old
// command-bridge format for any legacy caller that relied on it, but no
// in-tree caller invokes it after the 2026-04-24 tier-split refactor.
func WriteMCPConfig(vaultPath string) error {
	legacy := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"jules": map[string]interface{}{
				"command": "jules-local",
				"args":    []string{"mcp", "--vault", "."},
				"env": map[string]string{
					"JULES_CONFIG": "~/.config/jules/config.toml",
				},
			},
		},
	}

	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	mcpPath := filepath.Join(vaultPath, ".mcp.json")
	if err := os.WriteFile(mcpPath, data, 0o644); err != nil {
		return err
	}

	settingsPath := filepath.Join(vaultPath, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		settings := map[string]interface{}{
			"permissions": map[string]interface{}{
				"allow": []string{},
				"deny":  []string{},
			},
		}
		sData, _ := json.MarshalIndent(settings, "", "  ")
		sData = append(sData, '\n')
		_ = os.MkdirAll(filepath.Dir(settingsPath), 0o755)
		_ = os.WriteFile(settingsPath, sData, 0o644)
	}

	return nil
}
