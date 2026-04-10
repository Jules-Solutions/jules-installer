// Package setup — config.go writes the final config file and Claude Code MCP config.
package setup

import (
	"fmt"

	"github.com/Jules-Solutions/jules-installer/internal/config"
)

// WriteInstallerConfig saves the completed config to disk.
//
// TODO(Phase 2): also write the Claude Code .mcp.json pointing at jules-local.
func WriteInstallerConfig(cfg config.Config) error {
	// Stub: log intent but do not write.
	fmt.Printf("[stub] would write config: auth.api_url=%s vault=%s\n",
		cfg.Auth.APIURL, cfg.Local.VaultPath)
	return nil
}

// WriteMCPConfig writes the .mcp.json file into the vault root so Claude Code
// connects to the local jules-local MCP server.
//
// TODO(Phase 2): implement using encoding/json to write:
//
//	{
//	  "mcpServers": {
//	    "jules-local": {
//	      "command": "jules-local",
//	      "args": ["mcp"],
//	      "env": { "JULES_API_KEY": "<key>" }
//	    }
//	  }
//	}
func WriteMCPConfig(vaultPath, apiKey string) error {
	fmt.Printf("[stub] would write .mcp.json in %s\n", vaultPath)
	return nil
}
