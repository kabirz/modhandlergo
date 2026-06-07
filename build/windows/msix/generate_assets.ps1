# Generate MSIX PNG icon assets from icon.ico
param(
    [string]$IcoPath = "$PSScriptRoot\..\icon.ico",
    [string]$AssetsDir = "$PSScriptRoot\Assets"
)

if (Test-Path "$AssetsDir\StoreLogo.png") {
    Write-Host "Icons already exist, skipping generation"
    exit 0
}

if (-not (Test-Path $IcoPath)) {
    Write-Host "icon.ico not found at $IcoPath, skipping"
    exit 0
}

Add-Type -AssemblyName System.Drawing

$ico = [System.Drawing.Icon]::New($IcoPath, 256, 256)
$bmp = $ico.ToBitmap()

New-Item -ItemType Directory -Force -Path $AssetsDir | Out-Null

# Generate fixed-size icons
$sizes = @(
    @{ Name = "StoreLogo";         W = 50;  H = 50  },
    @{ Name = "Square44x44Logo";   W = 44;  H = 44  },
    @{ Name = "Square150x150Logo"; W = 150; H = 150 }
)

foreach ($sz in $sizes) {
    $img = New-Object System.Drawing.Bitmap($sz.W, $sz.H)
    $g = [System.Drawing.Graphics]::FromImage($img)
    $g.InterpolationMode = 'HighQualityBicubic'
    $g.DrawImage($bmp, 0, 0, $sz.W, $sz.H)
    $g.Dispose()
    $img.Save("$AssetsDir\$($sz.Name).png", [System.Drawing.Imaging.ImageFormat]::Png)
    $img.Dispose()
}

# Wide310x150Logo - centered icon
$w = New-Object System.Drawing.Bitmap(310, 150)
$g = [System.Drawing.Graphics]::FromImage($w)
$g.Clear([System.Drawing.Color]::Transparent)
$g.InterpolationMode = 'HighQualityBicubic'
$g.DrawImage($bmp, 80, 0, 150, 150)
$g.Dispose()
$w.Save("$AssetsDir\Wide310x150Logo.png", [System.Drawing.Imaging.ImageFormat]::Png)
$w.Dispose()

# SplashScreen 620x300 - centered icon
$s = New-Object System.Drawing.Bitmap(620, 300)
$g = [System.Drawing.Graphics]::FromImage($s)
$g.Clear([System.Drawing.Color]::Transparent)
$g.InterpolationMode = 'HighQualityBicubic'
$g.DrawImage($bmp, 210, 50, 200, 200)
$g.Dispose()
$s.Save("$AssetsDir\SplashScreen.png", [System.Drawing.Imaging.ImageFormat]::Png)
$s.Dispose()

$bmp.Dispose()
$ico.Dispose()

Write-Host "Icons generated in $AssetsDir"
