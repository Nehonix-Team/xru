$repo = "Nehonix-Team/xru"
$binaryName = "xru"

# Detect architecture
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "AMD64") { "amd64" } else { "arm64" }
$os = "windows"

Write-Host "🚀 Fetching latest version from GitHub..." -ForegroundColor Cyan
try {
    $latest = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
    $version = $latest.tag_name
} catch {
    Write-Host "❌ Failed to fetch latest version." -ForegroundColor Red
    exit 1
}

$fileName = "${binaryName}-${os}-${arch}.exe"
$url = "https://github.com/${repo}/releases/latest/download/${fileName}"

$destDir = "$HOME\.xru\bin"
if (!(Test-Path $destDir)) { 
    New-Item -ItemType Directory -Force -Path $destDir | Out-Null
}
$destFile = "$destDir\${binaryName}.exe"

Write-Host "📥 Installing ${binaryName} ${version} for ${os}/${arch}..." -ForegroundColor Blue

# Download
try {
    Invoke-WebRequest -Uri $url -OutFile $destFile
} catch {
    Write-Host "❌ Download failed. Check your connection or release availability." -ForegroundColor Red
    exit 1
}

# Add to PATH if not already there
$path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($path -notlike "*$destDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$path;$destDir", "User")
    $env:Path += ";$destDir"
    Write-Host "💡 Added $destDir to PATH. Restart your terminal to apply changes." -ForegroundColor Yellow
}

Write-Host "✅ ${binaryName} successfully installed to $destFile" -ForegroundColor Green
Write-Host "Run '$binaryName version' to verify."
