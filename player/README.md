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

## Native split-part title cards

Version 0.11 adds optional `TCD1` title-card assets for automatically split single-video ROMs. Before initializing the 120×80 video path, the player can show a full **240×160 RGB555** screen generated from the selected part's darkened first frame or another configured background.

The title card can wait for `A`, start after a timer, allow skipping, fade to black, initialize playback, and then fade the first video frame in. The converter supports Large, Medium, and Small title-card typography and defaults first-frame backgrounds to 50% darkening. Because the screen is pre-rendered during conversion, the GBA does not perform runtime image scaling, alpha blending, or font layout.

In multi-video menu ROMs, the selected clip uses a clear pixel-art arrow. Finishing a clip or pressing B returns to the menu.
