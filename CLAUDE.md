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
cmd/jules-setup/main.go  →  creates TUI app, runs tea.Program
internal/tui/            →  Bubbletea model, styles, screens, components
internal/auth/           →  3 auth methods (browser, device code, API key paste)
internal/audit/          →  environment detection (git, docker, python, etc.)
internal/setup/          →  interactive questions, vault download, config writing
internal/config/         →  config types, paths, file I/O (~/.config/jules/config.toml)
internal/update/         →  self-update checker (GitHub releases)
pkg/version/             →  build-time version injection via ldflags
```

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

## Phase 2 TODOs

- Implement real audit checks (git, disk, docker, python, node)
- Stream device code progress to TUI in real time
- Vault download with git clone + progress reporting
- Write ~/.config/jules/config.toml
- Write .mcp.json into vault root for Claude Code
- Self-update checker on startup
- Homebrew tap + Scoop bucket for distribution
