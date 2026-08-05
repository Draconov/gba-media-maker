param(
    [Parameter(Mandatory = $false)]
    [string]$Version = "0.10.0"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$stage = Join-Path $root "dist\GBA_Video_Maker_v$Version_Portable"
$zip = Join-Path $root "dist\GBA_Video_Maker_v$Version_Portable.zip"

& (Join-Path $PSScriptRoot "build-windows.ps1")

Remove-Item $stage -Recurse -Force -ErrorAction SilentlyContinue
New-Item $stage -ItemType Directory -Force | Out-Null
$ffmpeg = Join-Path $root "ffmpeg.exe"
if (-not (Test-Path $ffmpeg)) {
    throw "ffmpeg.exe is required beside the project before packaging. The app no longer downloads executables at runtime."
}

Copy-Item (Join-Path $root "GBA Video Maker.exe") $stage
Copy-Item $ffmpeg $stage
Copy-Item (Join-Path $root "README.md") $stage
Copy-Item (Join-Path $root "LICENSE") $stage
Copy-Item (Join-Path $root "THIRD_PARTY_NOTICES.md") $stage

Remove-Item $zip -Force -ErrorAction SilentlyContinue
Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $zip
(Get-FileHash $zip -Algorithm SHA256).Hash.ToLowerInvariant() | Set-Content "$zip.sha256" -NoNewline
Write-Host "Created: $zip"
