// Package setup — config.go writes the MCP config and Claude Code settings.
package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// WriteMCPConfig writes .mcp.json into the vault root so Claude Code
// auto-connects to the jules-local MCP server.
//
// The API key is NOT stored here — jules-local reads it from
// ~/.config/jules/config.toml at runtime.
func WriteMCPConfig(vaultPath string) error {
	mcpConfig := map[string]interface{}{
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

	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	mcpPath := filepath.Join(vaultPath, ".mcp.json")
	if err := os.WriteFile(mcpPath, data, 0o644); err != nil {
		return err
	}

	// Also write a minimal .claude/settings.json if it doesn't exist.
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
		// Ensure .claude/ dir exists.
		_ = os.MkdirAll(filepath.Dir(settingsPath), 0o755)
		_ = os.WriteFile(settingsPath, sData, 0o644)
	}

	return nil
}
