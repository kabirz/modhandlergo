# Create MSIX package using makeappx
param(
    [Parameter(Mandatory = $true)]
    [string]$ExePath,
    [Parameter(Mandatory = $true)]
    [string]$OutPath,
    [string]$Arch = "amd64"
)

$ErrorActionPreference = "Stop"

$msixDir = $PSScriptRoot
$staging = Join-Path $msixDir "staging"
$assetsSrc = Join-Path $msixDir "Assets"
$manifest = Join-Path $msixDir "app_manifest.xml"

# Clean and create staging
if (Test-Path $staging) { Remove-Item -Recurse -Force $staging }
New-Item -ItemType Directory -Force -Path (Join-Path $staging "Assets") | Out-Null

# Copy exe
Copy-Item $ExePath $staging

# Copy assets
if (Test-Path $assetsSrc) {
    Copy-Item (Join-Path $assetsSrc "*") (Join-Path $staging "Assets") -Recurse -Force
}

# Copy manifest
Copy-Item $manifest (Join-Path $staging "AppxManifest.xml")

# Find makeappx
$makeappx = Get-ChildItem "C:\Program Files (x86)\Windows Kits\10\bin\*\x64\makeappx.exe" `
    -ErrorAction SilentlyContinue | Sort-Object FullName -Descending | Select-Object -First 1

if (-not $makeappx) {
    Write-Error "makeappx.exe not found. Install Windows SDK."
    exit 1
}

# Create MSIX
Write-Host "Using makeappx: $($makeappx.FullName)"
& $makeappx.FullName pack /d $staging /p $OutPath /o

# Cleanup
Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $staging

if (-not (Test-Path $OutPath)) {
    Write-Error "MSIX package not created"
    exit 1
}

# Sign with test certificate if available
$pfxPath = Join-Path $msixDir "test_cert.pfx"
if (Test-Path $pfxPath) {
    $signtool = Get-ChildItem "C:\Program Files (x86)\Windows Kits\10\bin\*\x64\signtool.exe" `
        -ErrorAction SilentlyContinue | Sort-Object FullName -Descending | Select-Object -First 1
    if ($signtool) {
        Write-Host "Using signtool: $($signtool.FullName)"
        & $signtool.FullName sign /fd SHA256 /a /f $pfxPath /p test $OutPath 2>&1 | Write-Host
        Write-Host "MSIX package signed"
    } else {
        Write-Host "WARNING: signtool.exe not found, package is unsigned"
    }
} else {
    Write-Host "WARNING: test_cert.pfx not found, package is unsigned"
    Write-Host "Run: powershell -File build/windows/msix/create_test_cert.ps1 -Install"
}

$size = (Get-Item $OutPath).Length
Write-Host "MSIX package created: $OutPath ($([math]::Round($size / 1MB, 2)) MB)"
