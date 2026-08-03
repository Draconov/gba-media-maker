# Embedded GBA Player

The player is a freestanding ARM7TDMI program compiled with LLVM. GBV6 adds a tile-based collection menu, dynamic 120×80, 180×120, or 240×160 Mode 4 video, DMA framebuffer uploads, OBJ-based transient HUD elements, hybrid raw/byte-delta/repeat/tile-delta decoding, blocked IMA ADPCM streaming, scene-palette switching, seeking, playlist navigation, controls help, and SRAM resume support.

## Build on Linux/macOS

```bash
./player/build.sh
```

Requirements: `clang`, `ld.lld`, and `llvm-objcopy`.

## Build on Windows PowerShell

```powershell
./player/build.ps1
```

The resulting `assets/player_stub.bin` must be exactly **45,056 bytes (44 KiB)**. The converter patches the GBA header and the 64-byte GBV6 global metadata block at `0xAF00`; clip descriptors and assets begin at `0xB000`.

The linked player code must remain below the metadata address. The padded template intentionally leaves several KiB of code headroom before `0xAF00`.

Each GBV6 clip descriptor supplies its source width, source height, and frame-byte count. The player validates these fields, sizes codec operations dynamically, uploads rows with DMA3, and calculates the BG2 affine matrix for that clip. The maximum pair of EWRAM frame buffers is sized for native 240×160.
