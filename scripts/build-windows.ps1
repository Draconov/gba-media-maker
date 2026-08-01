$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    go test ./...
    go vet ./...

    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    go build -trimpath -ldflags "-H windowsgui -s -w" -o "GBA Video Maker.exe" .
    go run ./tools/embedicon -exe "GBA Video Maker.exe" -ico "assets/app_icon.ico"

    Write-Host "Built: $root\GBA Video Maker.exe"
}
finally {
    Pop-Location
}
