# ***************************************************************************
# * XRU (XyPriss Rule Unit) Installer Proxy (Windows)
# * Fetches and executes the official Nehonix installation workflow
# * @license MIT
# ***************************************************************************
$ErrorActionPreference = "Stop"

$officialInstaller = "https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.ps1"
$altInstaller = "https://dll.nehonix.com/xru/install.ps1"

try {
    $script = Invoke-RestMethod -Uri $officialInstaller -UseBasicParsing -ErrorAction Stop
    & ([scriptblock]::Create($script)) -Action "xru" @args
} catch {
    try {
        $script = Invoke-RestMethod -Uri $altInstaller -UseBasicParsing -ErrorAction Stop
        & ([scriptblock]::Create($script)) @args
    } catch {
        Write-Error "Failed to download installer from dll.nehonix.com: $_"
        exit 1
    }
}
