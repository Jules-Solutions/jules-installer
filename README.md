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
  -tier string    Skip tier picker. Accepts: 1|tier1|full, 2|tier2|remote
  -resume         Skip steps already completed (v0.2.0+ compat)
  -version, -v    Print version and exit
```

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

```sh
go test ./...
```

The test suite covers the tier contract end-to-end:
- `config_test.go` — TOML round-trip for `Tier`, `MCPPath`, `MCPURL`; file mode 0600
- `audit_test.go` — tier-aware severity demotion (Tier 2 Python-fail → warn)
- `setup/config_test.go` — `WriteMCPConfigForTier` produces the right file at the right path with mode 0600
- `main_test.go` — `--tier` flag alias parser

## License

Proprietary — Jules.Solutions
