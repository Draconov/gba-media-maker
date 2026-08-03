# Embedded GBA Player

The player is a freestanding ARM7TDMI program compiled with LLVM. It implements GBV5 clip menus, keyframe/delta decoding, scene-palette switching, audio seeking, HUD modes, controls help, volume/mute controls, and SRAM resume support.

## Build on Linux/macOS

```bash
./player/build.sh
```

Requirements: `clang`, `ld.lld`, and `llvm-objcopy`.

## Build on Windows PowerShell

```powershell
./player/build.ps1
```

The resulting `assets/player_stub.bin` must be exactly **32768 bytes**. The converter patches the GBA header and the global metadata block at `0x7F00`, then begins clip descriptors and assets at `0x8000`.


In multi-video menu ROMs, the selected clip uses a clear pixel-art arrow. Finishing a clip or pressing B returns to the menu.
