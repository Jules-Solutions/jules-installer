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

    # Detect architecture (default amd64; arm64 if running natively on ARM64).
    # GoReleaser ships per-arch assets named jules-setup_{version}_windows_{arch}.zip.
    $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
    $assetPattern = "jules-setup_*_windows_${arch}.zip"

    $asset = $release.assets | Where-Object { $_.name -like $assetPattern } | Select-Object -First 1

    if (-not $asset) {
        Write-Host "Error: No binary found for windows/$arch in latest release." -ForegroundColor Red
        Write-Host "Expected asset matching: $assetPattern" -ForegroundColor DarkGray
        exit 1
    }

    Write-Host "Version: $($release.tag_name)" -ForegroundColor DarkGray
    Write-Host "Platform: windows/$arch" -ForegroundColor DarkGray

    # Download the binary archive.
    $zipPath = Join-Path $tempDir "jules-setup.zip"
    $downloadHeaders = @{}
    if ($token) {
        $downloadHeaders["Authorization"] = "Bearer $token"
        $downloadHeaders["Accept"] = "application/octet-stream"
        $downloadUrl = "https://api.github.com/repos/$repo/releases/assets/$($asset.id)"
    } else {
        $downloadUrl = $asset.browser_download_url
    }

    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -Headers $downloadHeaders

    # Extract the archive (GoReleaser ships windows builds as zip-wrapped exe).
    Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force
    Remove-Item $zipPath -Force

    # Locate the extracted exe.
    $exe = Get-ChildItem -Path $tempDir -Filter "jules-setup.exe" -Recurse | Select-Object -First 1
    if (-not $exe) {
        Write-Host "Error: jules-setup.exe not found in archive after extraction." -ForegroundColor Red
        exit 1
    }

    Write-Host "Downloaded to: $($exe.FullName)" -ForegroundColor DarkGray

    # Run the installer.
    Write-Host ""
    & $exe.FullName
}
catch {
    Write-Host "Error: $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "If the repo is private, make sure you're logged in with: gh auth login" -ForegroundColor Yellow
    exit 1
}
