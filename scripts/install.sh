#!/bin/sh
# Jules.Solutions Installer — macOS/Linux Download Script
# Usage: curl -fsSL https://raw.githubusercontent.com/Jules-Solutions/jules-installer/main/scripts/install.sh | sh
#
# Downloads the latest jules-setup binary for your platform and runs it.

set -e

REPO="Jules-Solutions/jules-installer"

echo "Jules.Solutions Installer"
echo "Downloading latest release..."

# Detect OS and architecture.
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
esac

# Build asset name pattern.
# GoReleaser names: jules-setup_{version}_{os}_{arch}.tar.gz
PATTERN="${OS}_${ARCH}"

# Get GitHub auth token if gh CLI is available (for private repos).
AUTH_HEADER=""
if command -v gh >/dev/null 2>&1; then
    TOKEN=$(gh auth token 2>/dev/null || true)
    if [ -n "$TOKEN" ]; then
        AUTH_HEADER="Authorization: Bearer $TOKEN"
    fi
fi

# Fetch latest release.
if [ -n "$AUTH_HEADER" ]; then
    RELEASE=$(curl -fsSL -H "$AUTH_HEADER" -H "Accept: application/vnd.github+json" \
        "https://api.github.com/repos/$REPO/releases/latest")
else
    RELEASE=$(curl -fsSL -H "Accept: application/vnd.github+json" \
        "https://api.github.com/repos/$REPO/releases/latest")
fi

# Extract download URL for our platform.
URL=$(echo "$RELEASE" | grep -o "\"browser_download_url\"[^\"]*${PATTERN}[^\"]*" | head -1 | cut -d'"' -f4)

if [ -z "$URL" ]; then
    # Try direct binary name (for single-platform releases like v0.1.0).
    URL=$(echo "$RELEASE" | grep -o '"browser_download_url"[^"]*jules-setup[^"]*' | head -1 | cut -d'"' -f4)
fi

if [ -z "$URL" ]; then
    echo "Error: No binary found for ${OS}/${ARCH} in latest release."
    echo "If the repo is private, make sure you're logged in with: gh auth login"
    exit 1
fi

VERSION=$(echo "$RELEASE" | grep -o '"tag_name"[^"]*"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Version: $VERSION"
echo "Platform: ${OS}/${ARCH}"

# Download and extract.
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

if echo "$URL" | grep -q '\.tar\.gz$'; then
    curl -fsSL "$URL" -o "$TMPDIR/jules-setup.tar.gz"
    tar xzf "$TMPDIR/jules-setup.tar.gz" -C "$TMPDIR"
    BINARY="$TMPDIR/jules-setup"
else
    curl -fsSL "$URL" -o "$TMPDIR/jules-setup"
    BINARY="$TMPDIR/jules-setup"
fi

chmod +x "$BINARY"
echo "Downloaded to: $BINARY"
echo ""

# Run the installer.
exec "$BINARY"
