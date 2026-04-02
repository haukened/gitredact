# install.ps1 — installs the latest gitredact binary for Windows
#
# Usage:
#   irm https://raw.githubusercontent.com/haukened/gitredact/main/install.ps1 | iex
#
#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$InstallDir = "$env:USERPROFILE\.local\bin"
)

$ErrorActionPreference = 'Stop'

$Repo       = 'haukened/gitredact'
$BinaryName = 'gitredact.exe'

function Write-Info { param([string]$Msg); Write-Host "==> $Msg" }
function Write-Err  { param([string]$Msg); Write-Error "error: $Msg" }

# ── detect arch ───────────────────────────────────────────────────────────────

$RawArch = $env:PROCESSOR_ARCHITECTURE
switch ($RawArch) {
    'AMD64' { $GoArch = 'amd64' }
    default {
        Write-Err "Unsupported architecture: $RawArch (only AMD64 is supported)"
        exit 1
    }
}

Write-Info "Detected: windows/$GoArch"

# ── fetch latest release tag ──────────────────────────────────────────────────

Write-Info 'Fetching latest release tag...'
$ApiUrl  = "https://api.github.com/repos/$Repo/releases/latest"
$Release = Invoke-RestMethod -Uri $ApiUrl -UseBasicParsing
$Tag     = $Release.tag_name

if (-not $Tag) {
    Write-Err 'Failed to retrieve latest release tag from GitHub API'
    exit 1
}

Write-Info "Latest release: $Tag"

# ── download binary ───────────────────────────────────────────────────────────

$Asset       = "gitredact-windows-$GoArch.exe"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Tag/$Asset"
$Dest        = Join-Path $InstallDir $BinaryName

Write-Info "Downloading $Asset..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Invoke-WebRequest -Uri $DownloadUrl -OutFile $Dest -UseBasicParsing

Write-Info "Installed: $Dest"

# ── PATH injection ────────────────────────────────────────────────────────────

$UserPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
if ($UserPath -notlike "*$InstallDir*") {
    $NewPath = "$InstallDir;$UserPath"
    [Environment]::SetEnvironmentVariable('PATH', $NewPath, 'User')
    $env:PATH = "$InstallDir;$env:PATH"
    $PathUpdated = $true
} else {
    $PathUpdated = $false
}

# ── summary ───────────────────────────────────────────────────────────────────

Write-Host ''
Write-Host "gitredact $Tag installed successfully!"
Write-Host ''
Write-Host "  Binary: $Dest"

if ($PathUpdated) {
    Write-Host "  User PATH updated — restart your terminal to use gitredact"
} else {
    Write-Host '  PATH: already configured'
}

Write-Host ''
Write-Host '  Run: gitredact --version'
Write-Host ''
