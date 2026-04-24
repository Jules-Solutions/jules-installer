# jules-installer — Project Instructions

> Cross-platform installer binary for Jules.Solutions
> Stack: Go 1.24+, Bubbletea, Lipgloss, Bubbles
> Output: ~10MB static binary per platform (Windows .exe, macOS universal, Linux)

## Build

```bash
go build -o bin/jules-setup ./cmd/jules-setup   # dev build
make build                                       # with version info
make run                                         # build + run
goreleaser build --snapshot --clean              # cross-compile all platforms
```

## Architecture

```
cmd/jules-setup/main.go  →  creates TUI app, parses --tier / --resume, runs tea.Program
internal/tui/            →  Bubbletea model, styles, screens, components (tier + rerun screens)
internal/auth/           →  3 auth methods (browser, device code, API key paste)
internal/audit/          →  environment detection, tier-aware severity + install filters
internal/setup/          →  interactive questions, vault download, tier-aware MCP writer
internal/config/         →  config types (incl. Tier, MCPPath, MCPURL), file I/O (~/.config/jules/config.toml)
internal/update/         →  self-update checker (GitHub releases)
pkg/version/             →  build-time version injection via ldflags
```

## Tier Model (v0.3.0+)

The installer forces a tier choice right after the welcome screen (or via `--tier 1|2` flag).

- **Tier 1 (`config.TierFull`)** — local vault, jules-local CLI, `.mcp.json` in vault root
- **Tier 2 (`config.TierRemote`)** — MCP-only, `.mcp.json` at `~/.claude/.mcp.json`

Both tiers write the same `.mcp.json` shape: direct SSE URL + embedded `X-API-Key` header. The on-disk location is the only difference. File is mode 0600 (credential).

Flow:
```
Welcome → Tier pick → Auth → Audit → [Tier 1: Setup questions → Vault download → MCP write → Install jules-local → Done]
                                    → [Tier 2:                                   → MCP write                      → Done]
```

Re-run (no `--resume`, valid `config.toml` with tier present) shows a menu: Change tier / Re-run audit / Re-write MCP / Exit.

## Auth Flow

1. **Browser callback** (primary): start localhost server → open browser → receive API key
2. **Device code** (fallback): display code → user enters at auth.jules.solutions/device → poll for key
3. **API key paste** (last resort): user pastes `dck_...` key manually

Auth service base URL: `https://auth.jules.solutions`

Key endpoints:
- Browser callback redirect: `GET /installer/authorize?redirect_uri=...&state=...`
- Device code: `POST /api/auth/device/code`
- Device token poll: `GET /api/auth/device/token?device_code=X`
- API key verify: `GET /api/auth/api-key/verify` (Bearer token)

## Config

Stored at `~/.config/jules/config.toml`. On Linux, respects `XDG_CONFIG_HOME`.

## Conventions

- `internal/` for private packages, `pkg/` for importable utilities
- All external commands via `os/exec` (no CGO — static binaries)
- Bubbletea import: `github.com/charmbracelet/bubbletea`
- Lipgloss import: `github.com/charmbracelet/lipgloss`
- Bubbles import: `github.com/charmbracelet/bubbles`
- Brand colors in `internal/tui/styles.go`

## Status

- **v0.2.0** (2026-04-10): auth flows, environment audit, TUI, vault download, MCP config, jules-local install, CC launch, --resume
- **v0.3.0** (2026-04-24): Tier 1 vs Tier 2 split, tier-aware audit severity, re-run menu, --tier flag, unified direct-SSE MCP shape

## Deferred

- Stream device code progress to TUI in real time
- Homebrew tap + Scoop bucket for distribution
- Auto-download + replace binary on self-update (currently only notifies)
- Binary signing (Apple Developer + Windows Authenticode certs — Jules manual)
