$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$Version = (Get-Content (Join-Path $root "VERSION") -Raw).Trim()
if ($Version -notmatch '^[0-9]+\.[0-9]+\.[0-9]+$') {
    throw "VERSION must contain a semantic version such as 1.2.3."
}
$stage = Join-Path $root "dist\GBA_Media_Maker_v${Version}_Windows_x64"
$zip = Join-Path $root "dist\GBA_Media_Maker_v${Version}_Windows_x64.zip"

& (Join-Path $PSScriptRoot "build-windows.ps1")
Remove-Item $stage -Recurse -Force -ErrorAction SilentlyContinue
New-Item $stage -ItemType Directory -Force | Out-Null
$ffmpeg = Join-Path $root "ffmpeg.exe"
if (-not (Test-Path $ffmpeg)) {
    throw "ffmpeg.exe is required beside the project before packaging. Run .\scripts\fetch-ffmpeg.ps1 to fetch the pinned build."
}
$decoders = (& $ffmpeg -hide_banner -decoders 2>&1 | Out-String)
if ($LASTEXITCODE -ne 0) {
    throw "ffmpeg.exe could not list its decoders."
}
if ($decoders -notmatch "libdav1d" -and $decoders -notmatch "libaom-av1") {
    throw "ffmpeg.exe has no software AV1 decoder. Run .\scripts\fetch-ffmpeg.ps1 to install the pinned compatible build."
}
Copy-Item (Join-Path $root "GBA Media Maker.exe") $stage
Copy-Item $ffmpeg $stage
Copy-Item (Join-Path $root "README.md") $stage
Copy-Item (Join-Path $root "LICENSE") $stage
Copy-Item (Join-Path $root "THIRD_PARTY_NOTICES.md") $stage

Remove-Item $zip -Force -ErrorAction SilentlyContinue
Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $zip
(Get-FileHash $zip -Algorithm SHA256).Hash.ToLowerInvariant() | Set-Content "$zip.sha256" -NoNewline
Write-Host "Created: $zip"
