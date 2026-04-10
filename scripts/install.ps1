# Jules.Solutions Installer — Windows Download Script
# Usage: irm https://raw.githubusercontent.com/Jules-Solutions/jules-installer/main/scripts/install.ps1 | iex
#
# Downloads the latest jules-setup.exe and runs it.

$ErrorActionPreference = "Stop"

$repo = "Jules-Solutions/jules-installer"
$tempDir = Join-Path $env:TEMP "jules-setup"
$exePath = Join-Path $tempDir "jules-setup.exe"

Write-Host "Jules.Solutions Installer" -ForegroundColor Cyan
Write-Host "Downloading latest release..." -ForegroundColor DarkGray

# Create temp directory.
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null

try {
    # Get latest release info from GitHub API.
    $headers = @{ "Accept" = "application/vnd.github+json" }

    # If gh CLI is available, use its auth token for private repos.
    $token = $null
    if (Get-Command gh -ErrorAction SilentlyContinue) {
        $token = (gh auth token 2>$null)
        if ($token) {
            $headers["Authorization"] = "Bearer $token"
        }
    }

    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers $headers
    $asset = $release.assets | Where-Object { $_.name -eq "jules-setup.exe" }

    if (-not $asset) {
        Write-Host "Error: No jules-setup.exe found in latest release." -ForegroundColor Red
        exit 1
    }

    Write-Host "Version: $($release.tag_name)" -ForegroundColor DarkGray

    # Download the binary.
    $downloadHeaders = @{}
    if ($token) {
        $downloadHeaders["Authorization"] = "Bearer $token"
        $downloadHeaders["Accept"] = "application/octet-stream"
        $downloadUrl = "https://api.github.com/repos/$repo/releases/assets/$($asset.id)"
    } else {
        $downloadUrl = $asset.browser_download_url
    }

    Invoke-WebRequest -Uri $downloadUrl -OutFile $exePath -Headers $downloadHeaders
    Write-Host "Downloaded to: $exePath" -ForegroundColor DarkGray

    # Run the installer.
    Write-Host ""
    & $exePath
}
catch {
    Write-Host "Error: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "If the repo is private, make sure you're logged in with: gh auth login" -ForegroundColor Yellow
    exit 1
}
