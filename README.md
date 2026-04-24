# Jules.Solutions Installer

Cross-platform installer for [Jules.Solutions](https://jules.solutions).

Downloads as a single binary (~10MB). Authenticates you, checks your environment, and configures Claude Code to talk to the Jules.Solutions platform.

## Quick Install

**macOS / Linux:**

```sh
curl -fsSL https://jules.solutions/install | sh
```

**Windows (PowerShell):**

```powershell
irm jules.solutions/install | iex
```

## Two Install Tiers

The installer asks you to pick one right after launch:

| Tier | What you get | When to pick |
|------|--------------|--------------|
| **1. Full install** | Local vault (`~/You.Life`) + Claude Code MCP wired up + `jules-local` CLI (`jules`, `js`) for vault ops and playgrounds | Daily driver. You want the full Jules.Solutions workflow on this machine. |
| **2. Remote only (MCP)** | Just `~/.claude/.mcp.json` pointing at the hosted MCP endpoint. No vault, no CLI, no local Python. | Trying it out, headless boxes, or users who want the tools inside Claude Code without a local install. |

Both tiers use the same MCP shape: a direct SSE connection to `mcp.jules.solutions/sse` with your API key in the `X-API-Key` header. The only difference is *where* the `.mcp.json` lives (vault root vs. user-global).

**Upgrading Tier 2 → Tier 1 later:**

```sh
jules-setup --tier 1
```

Re-run the installer with the flag; it reuses the API key already stored in `~/.config/jules/config.toml` and adds the vault and `jules-local` install.

## What Happens When You Run It

1. **Tier pick** — 1 (full) or 2 (remote)
2. **Authenticate** — browser callback by default; falls back to device code, then API key paste
3. **Audit environment** — git, Docker, Python, Node, Claude Code, editors, SSH keys, disk. Tier 2 demotes non-essential tools to warnings rather than failures.
4. **Tier 1 only:** download/scaffold your vault, install `jules-local`, write `.mcp.json` inside the vault
5. **Tier 2 only:** write `~/.claude/.mcp.json`
6. **Done** — Tier 1 offers to launch Claude Code in the vault; Tier 2 tells you to restart CC.

## CLI Flags

```
jules-setup [flags]
  -tier string              Skip tier picker. Accepts: 1|tier1|full, 2|tier2|remote
  -local-tools-mcp string   (Tier 1) Also register jules-local stdio bridge in .mcp.json.
                            Accepts: true|false|yes|no|1|0|on|off. Default: prompt in TUI.
  -yes                      Non-interactive mode. Requires --tier and a pre-existing
                            API key in ~/.config/jules/config.toml or JULES_API_KEY env.
  -resume                   Skip steps already completed (v0.2.0+ compat)
  -version, -v              Print version and exit
```

### Non-interactive installs

For CI / scripted / unattended runs:

```sh
# Headless Tier 2 (remote MCP only) — API key already in config
jules-setup --tier 2 --yes

# Headless Tier 1 with local-tools bridge enabled
jules-setup --tier 1 --yes --local-tools-mcp true

# Upgrade from Tier 2 to Tier 1 non-interactively
jules-setup --tier 1 --yes
```

The `--yes` flag dispatches to a pure-Go runner that skips the Bubbletea TUI entirely. It's idempotent — re-running with the same flags is safe.

### Expose jules-local's local-only tools to Claude Code

Tier 1 installs `jules-local` as a CLI. By default, Claude Code's `.mcp.json` only registers the remote SSE server. To also let CC call local-machine tools (`exec_manage`, `file_manage`, `terminal_spawn`, `git_manage`) via a stdio bridge:

- **During install:** pick "Yes" on the "Also expose jules-local's local-only tools?" screen
- **After install (re-run):** pick "Toggle local-tools MCP" on the re-run menu
- **Scripted:** `jules-setup --tier 1 --local-tools-mcp true`

The default is **No** — unified direct-SSE matches the v0.3.0 ship behaviour.

## Re-running

If `~/.config/jules/config.toml` already exists and records a tier, re-running opens a menu:

- Change tier (Tier 2 → Tier 1 or vice versa)
- Re-run environment audit
- Re-write the `.mcp.json` at its recorded path (useful after an API key rotation)
- Exit

Passing `--resume` preserves the v0.2.0 behavior of skipping through to the first incomplete step linearly.

## Building from Source

Requires Go 1.24+:

```sh
git clone https://github.com/Jules-Solutions/jules-installer.git
cd jules-installer
make build
./bin/jules-setup
```

Cross-compile all platforms via GoReleaser:

```sh
goreleaser build --snapshot --clean
```

Or manually for a specific target:

```sh
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o jules-setup ./cmd/jules-setup
```

## Testing

Unit tests (fast, no network):

```sh
go test ./...
```

Integration tests (build the binary, spawn subprocesses — opt-in via build tag):

```sh
go test -tags integration ./cmd/jules-setup/
```

What's covered:

- `internal/config/config_test.go` — TOML round-trip for `Tier`, `MCPPath`, `MCPURL`, `LocalToolsMCP`; file mode 0600
- `internal/audit/audit_test.go` — tier-aware severity demotion (Tier 2 Python-fail → warn)
- `internal/setup/config_test.go` — `WriteMCPConfigForTier` produces the right file at the right path with mode 0600; Tier 1 with/without `LocalToolsMCP`; Tier 2 ignores the flag
- `internal/runner/runner_test.go` — headless mode: Tier 2 minimal, Tier 2 → Tier 1 upgrade, required-input guards
- `cmd/jules-setup/main_test.go` — `--tier` and `--local-tools-mcp` flag parsers
- `cmd/jules-setup/integration_test.go` (build tag `integration`) — scripted binary-level: Tier 2 headless, Tier 2 → Tier 1 upgrade, Tier 1 with local-tools, `--yes` refusing without `--tier`

## License

Proprietary — Jules.Solutions
