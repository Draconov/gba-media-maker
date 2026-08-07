$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    & (Join-Path $root "player\build.ps1")
    go test ./...
    go vet ./...

    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    Remove-Item (Join-Path $root "GBA Video Maker.exe") -Force -ErrorAction SilentlyContinue
    go build -trimpath -buildvcs=false -ldflags "-H windowsgui -s -w" -o "GBA Video Maker.exe" .
    go run ./tools/embedicon -exe "GBA Video Maker.exe" -ico "assets/app_icon.ico"

    Write-Host "Built: $root\GBA Video Maker.exe"
}
finally {
    Pop-Location
}
