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
    Remove-Item (Join-Path $root "GBA Media Maker.exe") -Force -ErrorAction SilentlyContinue
    go build -trimpath -buildvcs=false -ldflags "-H windowsgui -s -w" -o "GBA Media Maker.exe" .
    go run ./tools/embedicon -exe "GBA Media Maker.exe" -ico "assets/icon.ico"
    Write-Host "Built: $root\GBA Media Maker.exe"
}
finally {
    Pop-Location
}
