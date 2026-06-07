# Create a self-signed certificate for MSIX testing
# Run as Administrator to install cert to LocalMachine
param(
    [switch]$Install
)

$ErrorActionPreference = "Stop"

$certSubject = "CN=Kabirz"
$pfxPath = "$PSScriptRoot\test_cert.pfx"
$cerPath = "$PSScriptRoot\test_cert.cer"
$certPwd = "test"

# Remove existing cert if present
$existing = Get-ChildItem Cert:\CurrentUser\My | Where-Object { $_.Subject -eq $certSubject }
if ($existing) {
    $existing | Remove-Item -Force
    Write-Host "Removed existing certificate"
}

# Create self-signed certificate
$cert = New-SelfSignedCertificate `
    -Type Custom `
    -Subject $certSubject `
    -KeyUsage DigitalSignature `
    -FriendlyName "LaserRangeTool MSIX Test Cert" `
    -CertStoreLocation "Cert:\CurrentUser\My" `
    -TextExtension @("2.5.29.37={text}1.3.6.1.5.5.7.3.3", "2.5.29.19={text}")

Write-Host "Certificate created: $($cert.Thumbprint)"

# Export PFX (for signing)
$pwd = ConvertTo-SecureString -String $certPwd -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath $pfxPath -Password $pwd | Out-Null
Write-Host "PFX exported: $pfxPath"

# Export CER (for trust)
Export-Certificate -Cert $cert -FilePath $cerPath | Out-Null
Write-Host "CER exported: $cerPath"

if ($Install) {
    # Install to Trusted Root (requires admin)
    $rootCerts = Get-ChildItem Cert:\LocalMachine\Root | Where-Object { $_.Subject -eq $certSubject }
    if ($rootCerts) {
        $rootCerts | Remove-Item -Force
    }
    Import-Certificate -FilePath $cerPath -CertStoreLocation "Cert:\LocalMachine\Root" | Out-Null
    Write-Host "Certificate installed to Trusted Root CA store"
    Write-Host ""
    Write-Host "Now run: wails3 task windows:package -- FORMAT=msix"
    Write-Host "Then install the MSIX package normally."
}
