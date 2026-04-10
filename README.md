# Jules.Solutions Installer

Cross-platform installer for [Jules.Solutions](https://jules.solutions).

Downloads as a single binary (~10MB). Authenticates you, checks your environment, downloads your workspace, and gets you running with Claude Code.

## Quick Install

**macOS / Linux:**

```sh
curl -fsSL https://jules.solutions/install | sh
```

**Windows (PowerShell):**

```powershell
irm jules.solutions/install | iex
```

## What It Does

1. Authenticates with your Jules.Solutions account
2. Audits your development environment
3. Downloads your personalized vault
4. Configures Claude Code MCP connection
5. Hands off to Claude Code for first-run setup

## Building from Source

Requires Go 1.24+:

```sh
git clone https://github.com/Jules-Solutions/jules-installer.git
cd jules-installer
make build
./bin/jules-setup
```

## License

Proprietary — Jules.Solutions
