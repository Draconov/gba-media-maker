$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$build = Join-Path $PSScriptRoot "build"
Remove-Item $build -Recurse -Force -ErrorAction SilentlyContinue
New-Item $build -ItemType Directory -Force | Out-Null

$common = @("--target=arm-none-eabi", "-mcpu=arm7tdmi", "-marm", "-ffreestanding", "-fno-builtin", "-fno-stack-protector", "-Os", "-Wall", "-Wextra")
& clang @common -c (Join-Path $PSScriptRoot "startup.S") -o (Join-Path $build "startup.o")
& clang @common -c (Join-Path $PSScriptRoot "metadata.S") -o (Join-Path $build "metadata.o")
& clang @common -c (Join-Path $PSScriptRoot "main.c") -o (Join-Path $build "main.o")
& ld.lld -T (Join-Path $PSScriptRoot "linker.ld") --gc-sections -o (Join-Path $build "player.elf") (Join-Path $build "startup.o") (Join-Path $build "main.o") (Join-Path $build "metadata.o")
& llvm-objcopy -O binary (Join-Path $build "player.elf") (Join-Path $build "player.bin")
Copy-Item (Join-Path $build "player.bin") (Join-Path $root "assets\player_stub.bin") -Force
$file = [System.IO.File]::Open((Join-Path $root "assets\player_stub.bin"), [System.IO.FileMode]::Open)
try { $file.SetLength(45056) } finally { $file.Dispose() }
Write-Host "Built assets\player_stub.bin"
