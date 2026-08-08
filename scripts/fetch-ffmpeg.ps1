$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$tag = "autobuild-2026-08-07-13-13"
$archiveName = "ffmpeg-master-latest-win64-lgpl.zip"
$baseUrl = "https://github.com/BtbN/FFmpeg-Builds/releases/download/$tag"
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("gbavm-ffmpeg-" + [guid]::NewGuid().ToString("N"))
$archive = Join-Path $temp $archiveName
$checksums = Join-Path $temp "checksums.sha256"
$extract = Join-Path $temp "extract"
$destination = Join-Path $root "ffmpeg.exe"

try {
    New-Item -ItemType Directory -Path $temp -Force | Out-Null
    Write-Host "Downloading pinned FFmpeg checksum list..."
    Invoke-WebRequest -Uri "$baseUrl/checksums.sha256" -OutFile $checksums
    Write-Host "Downloading pinned FFmpeg build..."
    Invoke-WebRequest -Uri "$baseUrl/$archiveName" -OutFile $archive

    $line = Get-Content $checksums | Where-Object { $_ -match ("^[0-9a-fA-F]{64}\s+\*?" + [regex]::Escape($archiveName) + "$") } | Select-Object -First 1
    if (-not $line) {
        throw "Could not find $archiveName in the pinned release checksum list."
    }
    $expected = ($line -split "\s+")[0].ToLowerInvariant()
    $actual = (Get-FileHash $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "FFmpeg archive checksum mismatch. Expected $expected but got $actual."
    }

    Expand-Archive -Path $archive -DestinationPath $extract -Force
    $candidate = Get-ChildItem $extract -Recurse -File -Filter ffmpeg.exe |
        Where-Object { $_.FullName -match "[\\/]bin[\\/]ffmpeg\.exe$" } |
        Select-Object -First 1
    if (-not $candidate) {
        throw "No ffmpeg.exe was found in the pinned archive."
    }

    Copy-Item $candidate.FullName $destination -Force
    $decoders = (& $destination -hide_banner -decoders 2>&1 | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "Downloaded ffmpeg.exe could not list its decoders."
    }
    if ($decoders -notmatch "libdav1d" -and $decoders -notmatch "libaom-av1") {
        throw "Downloaded ffmpeg.exe has no software AV1 decoder (libdav1d/libaom-av1)."
    }

    Write-Host "Installed software-AV1-capable ffmpeg.exe: $destination"
}
finally {
    Remove-Item $temp -Recurse -Force -ErrorAction SilentlyContinue
}
