// Package setup — vault.go handles downloading/scaffolding the user's vault.
package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ScaffoldParams holds user-specific values substituted into vault templates.
type ScaffoldParams struct {
	UserName string // e.g. "Alice"
	VaultName string // e.g. "Alice.Life"
	UserEmail string
	APIURL   string // e.g. "https://api.jules.solutions"
	MCPURL   string // e.g. "https://mcp.jules.solutions"
	APIKey   string // e.g. "dck_..."
}

// DownloadVault attempts to set up the vault at the given path.
// Tries strategies in order: existing dir → git clone → empty scaffold.
// Returns the method used ("existing", "git_clone", "scaffold") and any error.
func DownloadVault(vaultPath string) (string, error) {
	return DownloadVaultWithParams(vaultPath, ScaffoldParams{})
}

// DownloadVaultWithParams is the full version that accepts scaffold parameters
// for template substitution in the fallback scaffold path.
func DownloadVaultWithParams(vaultPath string, params ScaffoldParams) (string, error) {
	// Derive username and vault name from path if not supplied.
	if params.VaultName == "" {
		params.VaultName = filepath.Base(vaultPath)
	}
	if params.UserName == "" {
		name := params.VaultName
		// Strip ".Life" suffix to get username.
		if strings.HasSuffix(name, ".Life") {
			name = name[:len(name)-5]
		}
		params.UserName = name
	}

	// Apply defaults for service URLs.
	if params.APIURL == "" {
		params.APIURL = "https://api.jules.solutions"
	}
	if params.MCPURL == "" {
		params.MCPURL = "https://mcp.jules.solutions"
	}

	// Strategy 0: vault directory already exists (re-run case).
	if info, err := os.Stat(vaultPath); err == nil && info.IsDir() {
		// Check if it has content (not just an empty dir).
		entries, _ := os.ReadDir(vaultPath)
		if len(entries) > 0 {
			return "existing", nil
		}
	}

	// Strategy 1: git clone via SSH (requires git + SSH keys + GitHub access).
	if hasGit() {
		// Try SSH clone from org repo.
		username := params.UserName
		repoSSH := fmt.Sprintf("git@github.com:Jules-Solutions/%s-vault.git", username)
		err := gitClone(repoSSH, vaultPath)
		if err == nil {
			return "git_clone", nil
		}

		// Try HTTPS clone (might work if user has gh credential helper).
		repoHTTPS := fmt.Sprintf("https://github.com/Jules-Solutions/%s-vault.git", username)
		err = gitClone(repoHTTPS, vaultPath)
		if err == nil {
			return "git_clone", nil
		}
	}

	// Strategy 2: scaffold a functional vault structure (offline fallback).
	if err := scaffoldVault(vaultPath, params); err != nil {
		return "", fmt.Errorf("scaffold failed: %w", err)
	}
	return "scaffold", nil
}

// hasGit checks if git is available.
func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// gitClone runs git clone with depth=1.
func gitClone(repoURL, targetDir string) error {
	cmd := exec.Command("git", "clone", "--depth=1", repoURL, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// scaffoldVault creates a comprehensive vault directory structure aligned with
// the user-vault provisioning manifest at .Life/templates/scaffolds/user-vault/.
// This is an OFFLINE FALLBACK — it won't be as rich as a fully provisioned vault
// (no ontology, pipelines, skills, agents, etc.) but is functional for first use.
func scaffoldVault(vaultPath string, p ScaffoldParams) error {
	// Create all required directories.
	dirs := []string{
		".claude",
		".claude/memory",
		".claude/rules",
		".claude/skills",
		".claude/skills/setup",
		".Life",
		".Life/templates",
		"Settings",
		"Workstreams",
		"Workstreams/getting-started",
		"Projects",
		"Inbox",
		"Areas",
		"Library",
		"Archive",
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(vaultPath, d), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}

	// Write .gitkeep files for empty leaf directories.
	gitkeepDirs := []string{"Projects", "Inbox", "Areas", "Library", "Archive"}
	for _, d := range gitkeepDirs {
		gitkeepPath := filepath.Join(vaultPath, d, ".gitkeep")
		if err := os.WriteFile(gitkeepPath, []byte{}, 0o644); err != nil {
			return fmt.Errorf("creating .gitkeep in %s: %w", d, err)
		}
	}

	// Write .Life/setup-state.json
	setupState := `{"status":"not_started"}` + "\n"
	if err := os.WriteFile(filepath.Join(vaultPath, ".Life", "setup-state.json"), []byte(setupState), 0o644); err != nil {
		return err
	}

	// Write CLAUDE.md — getting-started instructions for the user.
	claudeMD := substituteParams(`# {{vault_name}} — Agent Context

> Your Jules.Solutions vault. Claude Code reads this file on every session start.
> Customise this file as you learn the system.

---

## Identity Resolution

1. **Agent file** — launched with ` + "`--agent <name>`" + `? Your ` + "`.claude/agents/{name}.md`" + ` defines your role.
2. **Hook injection** — did a SessionStart hook assign your role? Follow that.
3. **Ask** — "What role should I take for this session?"

---

## Vault Structure

` + "```" + `
{{vault_name}}/
├── Workstreams/        # All active work lives here (STATE docs, designs, INSTs)
│   └── getting-started/
├── Projects/           # Active codebases (own CLAUDE.md, own git repos)
├── Areas/              # Life areas (Health, Work, Growth)
├── Library/            # External resources, cloned repos
├── Inbox/              # Quick capture, unprocessed items
├── Settings/           # Character-Sheet.md, Current_Focus.md (protected)
├── Archive/            # Completed/inactive items
├── .Life/              # System — setup state, templates, tools
└── .claude/            # CC runtime — settings, skills, rules, hooks, memory
` + "```" + `

---

## Principles

### SSOT (Single Source of Truth)
Every fact lives in exactly one place. Everything else links.

### Production-Ready
When choosing between quick and right, do it right.

### Autonomous Execution
{{user_name}} directs, you deliver. Execute without hand-holding.

### Modularity & Sustainability
Build things that compose. Small, focused units.

---

## Getting Started

1. Read ` + "`Workstreams/getting-started/getting-started.md`" + ` for orientation
2. Fill in ` + "`Settings/Character-Sheet.md`" + ` with your details
3. Ask Claude: "Help me get oriented in this vault"

Your API key is in ` + "`~/.config/jules/config.toml`" + ` — Jules.Solutions tools read it automatically.
`, p)
	if err := os.WriteFile(filepath.Join(vaultPath, "CLAUDE.md"), []byte(claudeMD), 0o644); err != nil {
		return err
	}

	// Write .claude/memory/MEMORY.md — empty memory index.
	memoryMD := `# Agent Memory Index

This file is loaded into every Claude Code session as the memory index.
Add pointers to memory files here as you build up your knowledge base.

Format: ` + "`- [Title](file.md) — one-line description`" + `
`
	if err := os.WriteFile(filepath.Join(vaultPath, ".claude", "memory", "MEMORY.md"), []byte(memoryMD), 0o644); err != nil {
		return err
	}

	// Write .claude/skills/setup/SKILL.md — setup skill placeholder.
	skillMD := `---
name: setup
description: Guide the user through initial vault setup and orientation
trigger: /setup
---

# Setup Skill

Use this skill to guide the user through:
1. Filling in their Character Sheet (Settings/Character-Sheet.md)
2. Creating their first workstream
3. Configuring Claude Code preferences

## Steps

1. Ask the user what they want to accomplish
2. Read Settings/Character-Sheet.md and ask them to fill it in
3. Create a workstream for their main goal
4. Show them how to use tasks and the MCP connection
`
	if err := os.WriteFile(filepath.Join(vaultPath, ".claude", "skills", "setup", "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		return err
	}

	// Write .claude/settings.json — MCP config with placeholder API key.
	settingsJSON := substituteParams(`{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "env": {
    "JULES_API_URL": "{{api_url}}",
    "JULES_MCP_URL": "{{mcp_url}}"
  }
}
`, p)
	if err := os.WriteFile(filepath.Join(vaultPath, ".claude", "settings.json"), []byte(settingsJSON), 0o644); err != nil {
		return err
	}

	// Write Settings/Character-Sheet.md — template for the user to fill in.
	charSheet := substituteParams(`---
type: character-sheet
owner: {{user_name}}
---

# Character Sheet

> Fill this in to help Claude Code understand who you are and how to work with you.
> The more detail you provide, the better Claude will tailor its responses.

## About Me

- **Name:** {{user_name}}
- **Email:** {{user_email}}
- **Role:** (e.g. Software engineer, designer, product manager...)
- **Location:** (timezone is helpful)

## Goals

- **Primary goal:** What are you trying to accomplish with this vault?
- **Short-term (this week):**
- **Medium-term (this quarter):**
- **Long-term (this year):**

## Communication Style

- **Verbosity:** (Concise / Detailed / Depends on topic)
- **Technical level:** (Beginner / Intermediate / Expert)
- **Preferred format:** (Bullet points / Prose / Code-first)
- **Things to avoid:**

## Working Hours

- **Timezone:**
- **Typical work hours:**
- **Focus time:**

## Tools & Stack

- **Primary languages:**
- **Preferred editor:**
- **Key tools:**
`, p)
	if err := os.WriteFile(filepath.Join(vaultPath, "Settings", "Character-Sheet.md"), []byte(charSheet), 0o644); err != nil {
		return err
	}

	// Write Settings/Current_Focus.md — empty focus file.
	currentFocus := `---
type: current-focus
---

# Current Focus

> Updated at the start of each week or when priorities shift.
> Keep this short — 3-5 bullets max.

## This Week

- (Add your weekly priorities here)

## Blocked On

- (Nothing yet)
`
	if err := os.WriteFile(filepath.Join(vaultPath, "Settings", "Current_Focus.md"), []byte(currentFocus), 0o644); err != nil {
		return err
	}

	// Write Workstreams/getting-started/getting-started.md — STATE doc.
	gettingStarted := substituteParams(`---
type: STATE
workstream: getting-started
status: active
priority: high
---

# Getting Started

> Your first workstream. Use this to orient yourself and make your first moves.
> When you're comfortable, archive this and start workstreams for your real goals.

## Objective

Get {{user_name}} fully set up and productive with Jules.Solutions.

## Status

Setup complete. Vault scaffolded offline (no git clone available at install time).

## Next Steps

- [ ] Fill in Settings/Character-Sheet.md with your details
- [ ] Run ` + "`claude`" + ` in this vault to start your first session
- [ ] Ask Claude: "Help me get oriented in this vault"
- [ ] Create your first real workstream for a goal you're working on

## Daily Log

### Setup
- Vault scaffolded by jules-setup installer
- MCP config written to .mcp.json
- API key saved to ~/.config/jules/config.toml
`, p)
	if err := os.WriteFile(filepath.Join(vaultPath, "Workstreams", "getting-started", "getting-started.md"), []byte(gettingStarted), 0o644); err != nil {
		return err
	}

	// Write .mcp.json — MCP server config.
	mcpJSON := substituteParams(`{
  "mcpServers": {
    "jules": {
      "command": "jules-local",
      "args": ["mcp", "--vault", "."],
      "env": {
        "JULES_CONFIG": "~/.config/jules/config.toml",
        "JULES_API_URL": "{{api_url}}",
        "JULES_MCP_URL": "{{mcp_url}}"
      }
    }
  }
}
`, p)
	if err := os.WriteFile(filepath.Join(vaultPath, ".mcp.json"), []byte(mcpJSON), 0o644); err != nil {
		return err
	}

	// Write .gitignore — comprehensive vault gitignore.
	gitignore := `# Obsidian
.obsidian/workspace
.obsidian/workspace.json
.obsidian/workspaces.json

# Claude Code ephemeral files
.claude/temp/
.claude/plans/
.claude/agent-memory/
.claude/memory/

# OS artifacts
.DS_Store
Thumbs.db
desktop.ini

# Secrets (never commit)
.env
*.env
*.secret
config.local.toml

# Python
__pycache__/
*.pyc
*.pyo
.venv/
venv/

# Node
node_modules/
.npm/

# Large files / caches
.Life/cache/
.Life/logs/
.Life/backups/
`
	if err := os.WriteFile(filepath.Join(vaultPath, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		return err
	}

	// Init git repo.
	if hasGit() {
		cmd := exec.Command("git", "init", vaultPath)
		_ = cmd.Run() // best-effort
	}

	return nil
}

// substituteParams replaces template placeholders in content with values from ScaffoldParams.
func substituteParams(content string, p ScaffoldParams) string {
	replacements := []string{
		"{{vault_name}}", p.VaultName,
		"{{user_name}}", p.UserName,
		"{{user_email}}", p.UserEmail,
		"{{api_url}}", p.APIURL,
		"{{mcp_url}}", p.MCPURL,
		"{{api_key}}", p.APIKey,
	}
	r := strings.NewReplacer(replacements...)
	return r.Replace(content)
}
